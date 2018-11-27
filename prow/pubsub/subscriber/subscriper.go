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
	"fmt"

	"k8s.io/test-infra/prow/client/clientset/versioned"

	"cloud.google.com/go/pubsub"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/kube"
	"k8s.io/test-infra/prow/pjutil"
	"k8s.io/test-infra/prow/pubsub/reporter"
)

const (
	prowEventType   = "ProwEventType"
	periodicProwJob = "ProwPeriodicProwJob"
	prowJobName     = "ProwJobName"
)

// kubeClientInterface mostly for testing.
type kubeClientInterface interface {
	CreateProwJob(job *kube.ProwJob) (*kube.ProwJob, error)
}

type KubeClient struct {
	Client    versioned.Interface
	Namespace string
	DryRun    bool
}

func (c *KubeClient) CreateProwJob(job *kube.ProwJob) (*kube.ProwJob, error) {
	if c.DryRun {
		return job, nil
	}
	return c.Client.ProwV1().ProwJobs(c.Namespace).Create(job)
}

// Subscriber handles Pub/Sub subscriptions, update metrics,
// validates them using Prow Configuration and
// use a kubeClientInterface to create Prow Jobs.
type Subscriber struct {
	ConfigAgent *config.Agent
	Metrics     *Metrics
	KubeClient  kubeClientInterface
}

type messageInterface interface {
	GetAttributes() map[string]string
	GetID() string
	Ack()
	Nack()
}

type pubSubMessage struct {
	msg *pubsub.Message
}

func (m *pubSubMessage) GetAttributes() map[string]string {
	return m.msg.Attributes
}

func (m *pubSubMessage) GetID() string {
	return m.msg.ID
}

func (m *pubSubMessage) Ack() {
	m.msg.Ack()
}

func (m *pubSubMessage) Nack() {
	m.msg.Nack()
}

func extractFromAttribute(attrs map[string]string, key string) (string, error) {
	value, ok := attrs[key]
	if !ok {
		return "", fmt.Errorf("unable to find %s from the attributes", key)
	}
	return value, nil
}

func extractPubsubMessage(attributes map[string]string) (*reporter.PubsubMessage, error) {
	var err error
	m := reporter.PubsubMessage{}
	if m.Topic, err = extractFromAttribute(attributes, reporter.PubsubTopicLabel); err != nil {
		return nil, err
	}
	if m.Project, err = extractFromAttribute(attributes, reporter.PubsubProjectLabel); err != nil {
		return nil, err
	}
	if m.RunID, err = extractFromAttribute(attributes, reporter.PubsubRunIDLabel); err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *Subscriber) handleMessage(msg messageInterface, subscription string) error {
	l := logrus.WithFields(logrus.Fields{
		"pubsub-subscription": subscription,
		"pubsub-id":           msg.GetID()})
	s.Metrics.MessageCounter.With(prometheus.Labels{subscriptionLabel: subscription}).Inc()
	l.Info("Received message")
	eType, err := extractFromAttribute(msg.GetAttributes(), prowEventType)
	if err != nil {
		l.WithError(err).Error("failed to read message")
		s.Metrics.ErrorCounter.With(prometheus.Labels{subscriptionLabel: subscription})
		return err
	}
	switch eType {
	case periodicProwJob:
		err := s.handlePeriodicJob(l, msg, subscription)
		if err != nil {
			l.WithError(err).Error("failed to create Prow Periodic Job")
			s.Metrics.ErrorCounter.With(prometheus.Labels{subscriptionLabel: subscription})
		}
		return err
	}
	err = fmt.Errorf("unsupported event type")
	l.WithError(err).Error("failed to read message")
	s.Metrics.ErrorCounter.With(prometheus.Labels{subscriptionLabel: subscription})
	return err
}

func (s *Subscriber) handlePeriodicJob(l *logrus.Entry, msg messageInterface, subscription string) error {
	l.Info("looking for periodic job")
	name, err := extractFromAttribute(msg.GetAttributes(), prowJobName)
	if err != nil {
		return err
	}
	var periodicJob *config.Periodic
	for _, job := range s.ConfigAgent.Config().AllPeriodics() {
		if job.Name == name {
			periodicJob = &job
			break
		}
	}
	if periodicJob == nil {
		return fmt.Errorf("failed to find associated periodic job")
	}
	prowJobSpec := pjutil.PeriodicSpec(*periodicJob)
	var prowJob kube.ProwJob
	if r, err := extractPubsubMessage(msg.GetAttributes()); err != nil {
		l.Warning("no pubsub information found to publish to")
		prowJob = pjutil.NewProwJob(prowJobSpec, nil)
	} else {
		// Add annotations
		prowJob = pjutil.NewProwJobWithAnnotation(prowJobSpec, nil, r.GetAnnotations())
	}
	_, err = s.KubeClient.CreateProwJob(&prowJob)
	if err != nil {
		l.WithError(err).Errorf("failed to create job %s", prowJob.Name)
	} else {
		l.Infof("periodic job %s created", prowJob.Name)
	}
	return err
}
