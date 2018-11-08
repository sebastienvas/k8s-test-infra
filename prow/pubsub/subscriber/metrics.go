/*
Copyright 2017 The Kubernetes Authors.

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
	"github.com/prometheus/client_golang/prometheus"
)

const (
	responseCodeLabel = "response_code"
	subscriptionLabel = "subscription"
)

var (
	// Define all metrics for pubsub subscriptions here.
	messageCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "prow_pubsub_message_counter",
		Help: "A counter of the webhooks made to prow.",
	}, []string{subscriptionLabel})
	errorCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "prow_pubsub_error_counter",
		Help: "A counter of the webhooks made to prow.",
	}, []string{subscriptionLabel})
	responseCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "prow_pubsub_response_codes",
		Help: "A counter of the different responses server has responded to Push Events with.",
	}, []string{responseCodeLabel, subscriptionLabel})
)

func init() {
	prometheus.MustRegister(messageCounter)
	prometheus.MustRegister(responseCounter)
	prometheus.MustRegister(errorCounter)
}

type Metrics struct {
	MessageCounter  *prometheus.CounterVec
	ResponseCounter *prometheus.CounterVec
	ErrorCounter    *prometheus.CounterVec
}

func NewMetrics() *Metrics {
	return &Metrics{
		MessageCounter:  messageCounter,
		ResponseCounter: responseCounter,
		ErrorCounter:    errorCounter,
	}
}
