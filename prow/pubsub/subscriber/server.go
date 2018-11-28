/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package subscriber

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/sirupsen/logrus"

	"cloud.google.com/go/pubsub"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
	"k8s.io/test-infra/prow/config"
)

const (
	tokenLabel = "token"
)

type message struct {
	Attributes map[string]string
	Data       []byte
	ID         string `json:"message_id"`
}

// pushRequest is the format of the push Pub/Sub subscription received form the WebHook.
type pushRequest struct {
	Message      message
	Subscription string
}

// PushServer implements http.Handler. It validates incoming Pub/Sub subscriptions handle them.
type PushServer struct {
	Subscriber     *Subscriber
	TokenGenerator func() []byte
}

// PullServer listen to Pull Pub/Sub subscriptions and handle them.
type PullServer struct {
	Subscriber *Subscriber

	Client pubsubClientInterface
}

// ServeHTTP validates an incoming Push Pub/Sub subscription and handle them.
func (s *PushServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	HTTPCode := http.StatusOK
	subscription := "unknown-subscription"
	var finalError error

	defer func() {
		s.Subscriber.Metrics.ResponseCounter.With(prometheus.Labels{
			subscriptionLabel: subscription,
			responseCodeLabel: string(HTTPCode),
		}).Inc()
		if finalError != nil {
			http.Error(w, finalError.Error(), HTTPCode)
		}
	}()

	if s.TokenGenerator != nil {
		token := r.URL.Query().Get(tokenLabel)
		if token != string(s.TokenGenerator()) {
			finalError = fmt.Errorf("wrong token")
			HTTPCode = http.StatusForbidden
			return
		}
	}
	// Get the payload and act on it.
	pr := &pushRequest{}
	if err := json.NewDecoder(r.Body).Decode(pr); err != nil {
		finalError = err
		HTTPCode = http.StatusBadRequest
		return
	}

	msg := &pubsub.Message{
		Data:       pr.Message.Data,
		ID:         pr.Message.ID,
		Attributes: pr.Message.Attributes,
	}

	if err := s.Subscriber.handleMessage(&pubSubMessage{msg: msg}, pr.Subscription); err != nil {
		finalError = err
		HTTPCode = http.StatusNotModified
		return
	}
}

// For testing
type subscriptionInterface interface {
	String() string
	Receive(ctx context.Context, f func(context.Context, messageInterface)) error
}

type pubsubClientInterface interface {
	New(ctx context.Context, project string) (pubsubClientInterface, error)
	Subscription(id string) subscriptionInterface
}

type PubSubClient struct {
	client *pubsub.Client
}

type pubSubSubscription struct {
	sub *pubsub.Subscription
}

func (s *pubSubSubscription) String() string {
	return s.sub.String()
}

func (s *pubSubSubscription) Receive(ctx context.Context, f func(context.Context, messageInterface)) error {
	g := func(ctx2 context.Context, msg2 *pubsub.Message) {
		f(ctx2, &pubSubMessage{msg: msg2})
	}
	return s.sub.Receive(ctx, g)
}

func (c *PubSubClient) New(ctx context.Context, project string) (pubsubClientInterface, error) {
	client, err := pubsub.NewClient(ctx, project)
	if err != nil {
		return nil, err
	}
	c.client = client
	return c, nil
}

func (c *PubSubClient) Subscription(id string) subscriptionInterface {
	return &pubSubSubscription{
		sub: c.client.Subscription(id),
	}
}

// handlePulls pull for Pub/Sub subscriptions and handle them.
func (s *PullServer) handlePulls(ctx context.Context, projectSubscriptions config.PubsubSubscriptions) (*errgroup.Group, context.Context, error) {
	// Since config might change we need be able to cancel the current run
	errGroup, derivedCtx := errgroup.WithContext(ctx)
	for project, subscriptions := range projectSubscriptions {
		client, err := s.Client.New(ctx, project)
		if err != nil {
			return errGroup, derivedCtx, err
		}
		for _, subName := range subscriptions {
			sub := client.Subscription(subName)
			errGroup.Go(func() error {
				logrus.Infof("Listening for subscription %s on project %s", sub.String(), project)
				defer logrus.Warnf("Stopped Listening for subscription %s on project %s", sub.String(), project)
				err := sub.Receive(derivedCtx, func(ctx context.Context, msg messageInterface) {
					if err = s.Subscriber.handleMessage(msg, sub.String()); err != nil {
						s.Subscriber.Metrics.ACKMessageCounter.With(prometheus.Labels{subscriptionLabel: sub.String()}).Inc()
					} else {
						s.Subscriber.Metrics.NACKMessageCounter.With(prometheus.Labels{subscriptionLabel: sub.String()}).Inc()
					}
					msg.Ack()
				})
				if err != nil {
					logrus.WithError(err).Errorf("failed to listen for subscription %s on project %s", sub.String(), project)
					return err
				}
				return nil
			})
		}
	}
	return errGroup, derivedCtx, nil
}

// Run will block listening to all subscriptions and return once the context is cancelled
// or one of the subscription has a unrecoverable error.
func (s *PullServer) Run(ctx context.Context) error {
	configEvent := make(chan config.ConfigDelta, 2)
	s.Subscriber.ConfigAgent.Subscribe(configEvent)

	var err error
	defer func() {
		if err != nil {
			logrus.WithError(ctx.Err()).Error("Pull server shutting down")
		}
		logrus.Warn("Pull server shutting down")
	}()
	currentConfig := s.Subscriber.ConfigAgent.Config().PubSubSubscriptions
	errGroup, derivedCtx, err := s.handlePulls(ctx, currentConfig)
	if err != nil {
		return err
	}

	for {
		select {
		// Parent context. Shutdown
		case <-ctx.Done():
			return ctx.Err()
		// Current thread context, it may be failing already
		case <-derivedCtx.Done():
			err = errGroup.Wait()
			return err
		// Checking for update config
		case event := <-configEvent:
			newConfig := event.After.PubSubSubscriptions
			logrus.Info("Received new config")
			if !reflect.DeepEqual(currentConfig, newConfig) {
				logrus.Warn("New config found, reloading pull Server")
				// Making sure the current thread finishes before starting a new one.
				errGroup.Wait()
				// Starting a new thread with new config
				errGroup, derivedCtx, err = s.handlePulls(ctx, newConfig)
				if err != nil {
					return err
				}
				currentConfig = newConfig
			}
		}
	}
}
