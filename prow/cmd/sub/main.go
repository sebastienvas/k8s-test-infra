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

package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"k8s.io/test-infra/prow/kube"

	"golang.org/x/sync/errgroup"
	"k8s.io/test-infra/prow/pubsub/subscriber"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/logrusutil"
	"k8s.io/test-infra/prow/metrics"
)

var (
	flagOptions *options
)

type options struct {
	masterURL  string
	kubeConfig string

	port           int
	push           bool
	pushSecretFile string

	configPath    string
	jobConfigPath string
	pluginConfig  string

	dryRun      bool
	gracePeriod time.Duration
}

func init() {
	flagOptions = &options{}
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.StringVar(&flagOptions.masterURL, "masterurl", "", "URL to k8s master")
	fs.StringVar(&flagOptions.kubeConfig, "kubeconfig", "", "Cluster config for the cluster you want to connect to")

	fs.IntVar(&flagOptions.port, "port", 80, "HTTP Port.")
	fs.BoolVar(&flagOptions.push, "push", false, "Using Push Server.")
	fs.StringVar(&flagOptions.pushSecretFile, "push-secret-file", "", "Path to Pub/Sub Push secret file.")

	fs.StringVar(&flagOptions.configPath, "config-path", "../../config.yaml", "Path to config.yaml.")
	fs.StringVar(&flagOptions.jobConfigPath, "job-config-path", "../../../config/jobs", "Path to prow job configs.")

	fs.BoolVar(&flagOptions.dryRun, "dry-run", true, "Dry run for testing. Uses API tokens but does not mutate.")
	fs.DurationVar(&flagOptions.gracePeriod, "grace-period", 180*time.Second, "On shutdown, try to handle remaining events for the specified duration. ")
	fs.Parse(os.Args[1:])
}

func main() {

	logrus.SetFormatter(logrusutil.NewDefaultFieldsFormatter(nil, logrus.Fields{"component": "pubsub-subscriber"}))

	configAgent := &config.Agent{}
	if err := configAgent.Start(flagOptions.configPath, flagOptions.jobConfigPath); err != nil {
		logrus.WithError(err).Fatal("Error starting config agent.")
	}

	// Append the path of hmac and github secrets.
	var tokenGenerator func() []byte
	if flagOptions.pushSecretFile != "" {
		var tokens []string
		tokens = append(tokens, flagOptions.pushSecretFile)

		secretAgent := &config.SecretAgent{}
		if err := secretAgent.Start(tokens); err != nil {
			logrus.WithError(err).Fatal("Error starting secrets agent.")
		}
		tokenGenerator = secretAgent.GetTokenGenerator(flagOptions.pushSecretFile)
	}

	_, prowjobClient := kube.GetKubernetesClient(flagOptions.masterURL, flagOptions.kubeConfig)
	kubeClient := &subscriber.KubeClient{
		Client:    prowjobClient,
		Namespace: configAgent.Config().ProwJobNamespace,
		DryRun:    flagOptions.dryRun,
	}

	promMetrics := subscriber.NewMetrics()

	// Push metrics to the configured prometheus pushgateway endpoint.
	pushGateway := configAgent.Config().PushGateway
	if pushGateway.Endpoint != "" {
		go metrics.PushMetrics("sub", pushGateway.Endpoint, pushGateway.Interval)
	}

	s := &subscriber.Subscriber{
		ConfigAgent: configAgent,
		Metrics:     promMetrics,
		KubeClient:  kubeClient,
	}

	// Return 200 on / for health checks.
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {})
	http.Handle("/metrics", promhttp.Handler())

	// Will call shutdown which will stop the errGroup
	shutdownCtx, shutdown := context.WithCancel(context.Background())
	errGroup, derivedCtx := errgroup.WithContext(shutdownCtx)
	wg := sync.WaitGroup{}

	if flagOptions.push {
		logrus.Info("Setting up Push Server")
		pushServer := &subscriber.PushServer{
			Subscriber:     s,
			TokenGenerator: tokenGenerator,
		}
		http.Handle("/push", pushServer)
	} else {
		logrus.Info("Setting up Pull Server")
		// Using Pull Server
		pullServer := subscriber.PullServer{
			Subscriber:      s,
			ConfigCheckTick: time.NewTicker(time.Minute),
			Client:          &subscriber.PubSubClient{},
		}
		errGroup.Go(func() error {
			wg.Add(1)
			defer wg.Done()
			logrus.Info("Starting Pull Server")
			err := pullServer.Run(derivedCtx)
			logrus.WithError(err).Warn("Pull Server exited.")
			return err
		})
	}

	httpServer := &http.Server{Addr: ":" + strconv.Itoa(flagOptions.port)}
	errGroup.Go(func() error {
		wg.Add(1)
		defer wg.Done()
		logrus.Info("Starting HTTP Server")
		err := httpServer.ListenAndServe()
		logrus.WithError(err).Warn("HTTP Server exited.")
		return err
	})

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM, syscall.SIGABRT)
	var err error
	select {
	case <-shutdownCtx.Done():
		err = shutdownCtx.Err()
		break
	case <-derivedCtx.Done():
		err = derivedCtx.Err()
		break
	case <-sig:
		break
	}

	logrus.WithError(err).Warn("Starting Shutdown")
	shutdown()
	// Shutdown gracefully on SIGTERM or SIGINT
	timeoutCtx, cancel := context.WithTimeout(context.Background(), flagOptions.gracePeriod)
	defer cancel()
	httpServer.Shutdown(timeoutCtx)
	errGroup.Wait()
	wg.Wait()
}
