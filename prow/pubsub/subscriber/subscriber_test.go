/*
Copyright 2018 The Kubernetes Authors.

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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/kube"
	"k8s.io/test-infra/prow/pubsub/reporter"
)

type kubeTestClient struct {
	pj *kube.ProwJob
}

type pubSubTestClient struct {
	messageChan chan fakeMessage
}

type fakeSubscription struct {
	name        string
	messageChan chan fakeMessage
}

type fakeMessage struct {
	attributes map[string]string
	id         string
}

func (m *fakeMessage) GetAttributes() map[string]string {
	return m.attributes
}

func (m *fakeMessage) GetID() string {
	return m.id
}

func (m *fakeMessage) Ack()  {}
func (m *fakeMessage) Nack() {}

func (s *fakeSubscription) String() string {
	return s.name
}

func (s *fakeSubscription) Receive(ctx context.Context, f func(context.Context, messageInterface)) error {
	derivedCtx, cancel := context.WithCancel(ctx)
	msg := <-s.messageChan
	go func() {
		f(derivedCtx, &msg)
		cancel()
	}()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-derivedCtx.Done():
			return fmt.Errorf("message processed")
		}
	}
}

func (c *pubSubTestClient) New(ctx context.Context, project string) (pubsubClientInterface, error) {
	return c, nil
}

func (c *pubSubTestClient) Subscription(id string) subscriptionInterface {
	return &fakeSubscription{name: id, messageChan: c.messageChan}
}

func (c *kubeTestClient) CreateProwJob(job *kube.ProwJob) (*kube.ProwJob, error) {
	c.pj = job
	return job, nil
}

func TestHandleMessage(t *testing.T) {
	for _, tc := range []struct {
		name   string
		msg    *pubsub.Message
		s      string
		config *config.Config
		err    string
		labels []string
	}{
		{
			name: "PeriodicJobNoPubsub",
			msg: &pubsub.Message{
				ID: "fakeID",
				Attributes: map[string]string{
					prowEventType: periodicProwJob,
					prowJobName:   "test",
				},
			},
			config: &config.Config{
				JobConfig: config.JobConfig{
					Periodics: []config.Periodic{
						{
							JobBase: config.JobBase{
								Name: "test",
							},
						},
					},
				},
			},
		},
		{
			name: "UnknownEventType",
			msg: &pubsub.Message{
				ID: "fakeID",
				Attributes: map[string]string{
					prowEventType: "UnknownEventType",
				},
			},
			config: &config.Config{},
			err:    "unsupported event type",
			labels: []string{reporter.PubSubTopicLabel, reporter.PubSubRunIDLabel, reporter.PubSubProjectLabel},
		},
	} {
		t.Run(tc.name, func(t1 *testing.T) {
			kc := &kubeTestClient{}
			ca := &config.Agent{}
			ca.Set(tc.config)
			s := Subscriber{
				Metrics:     NewMetrics(),
				KubeClient:  kc,
				ConfigAgent: ca,
			}
			if err := s.handleMessage(&pubSubMessage{tc.msg}, tc.s); err != nil {
				if err.Error() != tc.err {
					t1.Errorf("Expected error %v got %v", tc.err, err.Error())
				} else if tc.err == "" {
					if kc.pj == nil {
						t.Errorf("Prow job not created")
					}
					for _, k := range tc.labels {
						if _, ok := kc.pj.Labels[k]; !ok {
							t.Errorf("label %s is missing", k)
						}
					}
				}
			}
		})
	}
}

func TestHandlePeriodicJob(t *testing.T) {
	for _, tc := range []struct {
		name   string
		msg    *pubsub.Message
		s      string
		config *config.Config
		err    string
	}{
		{
			name: "PeriodicJobNoPubsub",
			msg: &pubsub.Message{
				ID: "fakeID",
				Attributes: map[string]string{
					prowEventType: periodicProwJob,
					prowJobName:   "test",
				},
			},
			config: &config.Config{
				JobConfig: config.JobConfig{
					Periodics: []config.Periodic{
						{
							JobBase: config.JobBase{
								Name: "test",
							},
						},
					},
				},
			},
		},
		{
			name: "PeriodicJobPubsubSet",
			msg: &pubsub.Message{
				ID: "fakeID",
				Attributes: map[string]string{
					prowEventType:               periodicProwJob,
					prowJobName:                 "test",
					reporter.PubSubProjectLabel: "project",
					reporter.PubSubRunIDLabel:   "runid",
					reporter.PubSubTopicLabel:   "topic",
				},
			},
			config: &config.Config{
				JobConfig: config.JobConfig{
					Periodics: []config.Periodic{
						{
							JobBase: config.JobBase{
								Name: "test",
							},
						},
					},
				},
			},
		},
		{
			name: "JobNotFound",
			msg: &pubsub.Message{
				ID: "fakeID",
				Attributes: map[string]string{
					prowEventType: periodicProwJob,
					prowJobName:   "test",
				},
			},
			config: &config.Config{},
			err:    "failed to find associated periodic job",
		},
	} {
		t.Run(tc.name, func(t1 *testing.T) {
			kc := &kubeTestClient{}
			ca := &config.Agent{}
			ca.Set(tc.config)
			s := Subscriber{
				Metrics:     NewMetrics(),
				KubeClient:  kc,
				ConfigAgent: ca,
			}
			if err := s.handlePeriodicJob(logrus.NewEntry(logrus.New()), &pubSubMessage{tc.msg}, tc.s); err != nil {
				if err.Error() != tc.err {
					t1.Errorf("Expected error %v got %v", tc.err, err.Error())
				} else if tc.err == "" {
					if kc.pj == nil {
						t.Errorf("Prow job not created")
					}
				}
			}
		})
	}
}

func TestPushServer_ServeHTTP(t *testing.T) {
	kc := &kubeTestClient{}
	pushServer := PushServer{
		Subscriber: &Subscriber{
			ConfigAgent: &config.Agent{},
			Metrics:     NewMetrics(),
			KubeClient:  kc,
		},
	}
	for _, tc := range []struct {
		name         string
		url          string
		secret       string
		pushRequest  interface{}
		expectedCode int
	}{
		{
			name:   "WrongToken",
			secret: "wrongToken",
			url:    "https://prow.k8s.io/push",
			pushRequest: pushRequest{
				Message: message{
					ID: "runid",
				},
			},
			expectedCode: http.StatusForbidden,
		},
		{
			name: "NoToken",
			url:  "https://prow.k8s.io/push",
			pushRequest: pushRequest{
				Message: message{
					ID: "runid",
				},
			},
			expectedCode: http.StatusNotModified,
		},
		{
			name:   "RightToken",
			secret: "secret",
			url:    "https://prow.k8s.io/push?token=secret",
			pushRequest: pushRequest{
				Message: message{
					ID: "runid",
				},
			},
			expectedCode: http.StatusNotModified,
		},
		{
			name:         "InvalidPushRequest",
			secret:       "secret",
			url:          "https://prow.k8s.io/push?token=secret",
			pushRequest:  "invalid",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:   "SuccessToken",
			secret: "secret",
			url:    "https://prow.k8s.io/push?token=secret",
			pushRequest: pushRequest{
				Message: message{
					ID: "fakeID",
					Attributes: map[string]string{
						prowEventType: periodicProwJob,
						prowJobName:   "test",
					},
				},
			},
			expectedCode: http.StatusOK,
		},
		{
			name: "SuccessNoToken",
			url:  "https://prow.k8s.io/push",
			pushRequest: pushRequest{
				Message: message{
					ID: "fakeID",
					Attributes: map[string]string{
						prowEventType: periodicProwJob,
						prowJobName:   "test",
					},
				},
			},
			expectedCode: http.StatusOK,
		},
	} {
		t.Run(tc.name, func(t1 *testing.T) {
			c := &config.Config{
				JobConfig: config.JobConfig{
					Periodics: []config.Periodic{
						{
							JobBase: config.JobBase{
								Name: "test",
							},
						},
					},
				},
			}
			pushServer.Subscriber.ConfigAgent.Set(c)
			pushServer.TokenGenerator = func() []byte { return []byte(tc.secret) }
			kc.pj = nil

			body := new(bytes.Buffer)

			if err := json.NewEncoder(body).Encode(tc.pushRequest); err != nil {
				t1.Errorf(err.Error())
			}
			req := httptest.NewRequest(http.MethodPost, tc.url, body)
			w := httptest.NewRecorder()
			pushServer.ServeHTTP(w, req)
			resp := w.Result()
			if resp.StatusCode != tc.expectedCode {
				t1.Errorf("exected code %d got %d", tc.expectedCode, resp.StatusCode)
			}
		})
	}
}

func TestPullServer_RunShutdown(t *testing.T) {
	kc := &kubeTestClient{}
	s := &Subscriber{
		ConfigAgent: &config.Agent{},
		KubeClient:  kc,
		Metrics:     NewMetrics(),
	}
	c := &config.Config{}
	s.ConfigAgent.Set(c)
	pullServer := PullServer{
		Subscriber:      s,
		ConfigCheckTick: time.NewTicker(time.Millisecond),
		Client:          &pubSubTestClient{},
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	errChan := make(chan error)
	go func() {
		errChan <- pullServer.Run(ctx)
	}()
	time.Sleep(10 * time.Millisecond)
	cancel()
	err := <-errChan
	if err != nil {
		if !strings.HasPrefix(err.Error(), "context canceled") {
			t.Errorf("unexpected error: %v", err)
		}
	}
}

func TestPullServer_RunHandlePullFail(t *testing.T) {
	kc := &kubeTestClient{}
	s := &Subscriber{
		ConfigAgent: &config.Agent{},
		KubeClient:  kc,
		Metrics:     NewMetrics(),
	}
	c := &config.Config{
		ProwConfig: config.ProwConfig{
			PubSubSubscriptions: map[string][]string{
				"project": {"test"},
			},
		},
	}
	messageChan := make(chan fakeMessage, 1)
	s.ConfigAgent.Set(c)
	pullServer := PullServer{
		Subscriber:      s,
		ConfigCheckTick: time.NewTicker(100 * time.Millisecond),
		Client:          &pubSubTestClient{messageChan: messageChan},
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	errChan := make(chan error)
	messageChan <- fakeMessage{
		attributes: map[string]string{},
		id:         "test",
	}
	defer cancel()
	go func() {
		errChan <- pullServer.Run(ctx)
	}()
	err := <-errChan
	// Should fail since Pub/Sub cred are not set
	if !strings.HasPrefix(err.Error(), "message processed") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPullServer_RunConfigChange(t *testing.T) {
	kc := &kubeTestClient{}
	s := &Subscriber{
		ConfigAgent: &config.Agent{},
		KubeClient:  kc,
		Metrics:     NewMetrics(),
	}
	c := &config.Config{}
	messageChan := make(chan fakeMessage, 1)
	s.ConfigAgent.Set(c)
	pullServer := PullServer{
		Subscriber:      s,
		ConfigCheckTick: time.NewTicker(50 * time.Millisecond),
		Client:          &pubSubTestClient{messageChan: messageChan},
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	errChan := make(chan error)
	go func() {
		errChan <- pullServer.Run(ctx)
	}()
	select {
	case <-errChan:
		t.Error("should not fail")
	case <-time.After(100 * time.Millisecond):
		newConfig := &config.Config{
			ProwConfig: config.ProwConfig{
				PubSubSubscriptions: map[string][]string{
					"project": {"test"},
				},
			},
		}
		s.ConfigAgent.Set(newConfig)
		messageChan <- fakeMessage{
			attributes: map[string]string{},
			id:         "test",
		}
		err := <-errChan
		if !strings.HasPrefix(err.Error(), "message processed") {
			t.Errorf("unexpected error: %v", err)
		}
	}
}
