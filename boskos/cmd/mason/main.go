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

package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/boskos/gcp"
	"k8s.io/test-infra/boskos/mason"
)

var storagePath = flag.String("storage", "", "Path to persistent volume to load configs")

func main() {
	logrus.SetFormatter(&logrus.JSONFormatter{})

	flag.Parse()
	mason := mason.NewMasonFromFlags()

	// Registering Masonable Converters
	if err := mason.RegisterConfigConverter(gcp.ResourceConfigType, gcp.ConfigConverter); err != nil {
		logrus.WithError(err).Fatal("unable tp register config converter")
	}
	logrus.Info("Initialized boskos client!")

	if *storagePath != "" {
		go func() {
			for range time.NewTicker(time.Minute * 10).C {
				if err := mason.UpdateConfigs(*storagePath); err != nil {
					logrus.WithError(err).Warning("failed to update mason config")
				}
			}
		}()
	}

	mason.Start()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	mason.Stop()
}
