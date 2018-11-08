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

	"github.com/prometheus/client_golang/prometheus"

	"cloud.google.com/go/pubsub"
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
	Subscriber Subscriber
	PushSecret string
}

// PullServer listen to Pull Pub/Sub subscriptions and handle them.
type PullServer struct {
	Subscriber             Subscriber
	Subscriptions          config.PubsubSubscriptions
	MaxOutstandingMessages int
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

	if s.PushSecret != "" {
		token := r.URL.Query().Get(tokenLabel)
		if token != s.PushSecret {
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

	if err := s.Subscriber.handleMessage(msg, pr.Subscription); err != nil {
		finalError = err
		HTTPCode = http.StatusNotModified
		return
	}
}

// HandlePulls pull for Pub/Sub subscriptions and handle them.
func (s *PullServer) HandlePulls(ctx context.Context) error {
	errGroup, derivedCtx := errgroup.WithContext(ctx)
	for project, subscriptions := range s.Subscriptions {
		client, err := pubsub.NewClient(ctx, project)
		if err != nil {
			return err
		}
		for _, subName := range subscriptions {
			sub := client.Subscription(subName)
			if s.MaxOutstandingMessages > 0 {
				sub.ReceiveSettings.MaxOutstandingMessages = s.MaxOutstandingMessages
			}
			f := func() error {
				err := sub.Receive(derivedCtx, func(ctx context.Context, msg *pubsub.Message) {
					if err = s.Subscriber.handleMessage(msg, subName); err != nil {
						msg.Nack()
					} else {
						msg.Ack()
					}
				})
				if err != nil {
					return err
				}
				return nil
			}
			errGroup.Go(f)
		}
	}
	return errGroup.Wait()
}
