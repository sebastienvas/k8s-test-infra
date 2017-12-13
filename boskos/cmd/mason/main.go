package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/boskos/gcp"
	"k8s.io/test-infra/boskos/mason"
)

func main() {
	logrus.SetFormatter(&logrus.JSONFormatter{})

	flag.Parse()
	mason := mason.NewMasonFromFlags()

	// Registering Config Converters
	mason.RegisterConfigConverter(gcp.ResourceConfigType, gcp.ConfigConverter)
	logrus.Info("Initialized boskos client!")

	mason.Start()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	mason.Stop()
}
