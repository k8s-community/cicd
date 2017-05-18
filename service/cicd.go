package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Sirupsen/logrus"
	"github.com/k8s-community/cicd/handlers"
	"github.com/k8s-community/cicd/version"
	common_handlers "github.com/k8s-community/handlers"
	"github.com/takama/daemon"
	"github.com/takama/router"
)

func main() {
	log := logrus.New()
	log.Formatter = new(logrus.TextFormatter)
	logger := log.WithFields(logrus.Fields{"service": "cicd"})

	status, err := daemonCommands()
	if err != nil {
		logger.Fatalf("%s: %s", status, err)
	}

	var errors []error

	serviceHost, err := getFromEnv("SERVICE_HOST")
	if err != nil {
		errors = append(errors, err)
	}

	servicePort, err := getFromEnv("SERVICE_PORT")
	if err != nil {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		logger.Fatalf("Couldn't start service because required parameters are not set: %+v", errors)
	}

	// TODO: add graceful shutdown

	buildHandler := handlers.NewBuild(logger)

	r := router.New()

	r.POST("/api/v1/build", buildHandler.Run)

	r.GET("/info", func(c *router.Control) {
		common_handlers.Info(c, version.RELEASE, version.REPO, version.COMMIT)
	})
	r.GET("/healthz", func(c *router.Control) {
		c.Code(http.StatusOK).Body(http.StatusText(http.StatusOK))
	})

	hostPort := fmt.Sprintf("%s:%s", serviceHost, servicePort)
	logger.Infof("Ready to listen %s. Routes: %+v", hostPort, r.Routes())
	go r.Listen(hostPort)

	// Set up channel on which to send signal notifications.
	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when the signal is sent.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, os.Kill, syscall.SIGTERM)
	killSignal := <-interrupt
	logger.Infof("Got signal: %s", killSignal)
	status, err = shutdown()
	if err != nil {
		logger.Fatalf("Error: %s Status: %s\n", err.Error(), status)
	}
	if killSignal == os.Kill {
		logger.Infof("Service was killed")
	} else {
		logger.Infof("Service was terminated by system signal")
	}
	logger.Infof(status)
}

func shutdown() (string, error) {
	return "Shutdown", nil
}

func daemonCommands() (string, error) {

	svc, err := daemon.New("cicd", "Simplest CI/CD service")
	if err != nil {
		return "Couldn't init daemon", err
	}

	// if received any kind of command, do it
	if len(os.Args) > 1 {
		command := os.Args[1]
		switch command {
		case "install":
			return svc.Install(os.Args[2:]...)
		case "remove":
			return svc.Remove()
		case "start":
			return svc.Start()
		case "stop":
			return svc.Stop()
		case "status":
			return svc.Status()
		}
	}

	return "Ok", nil
}

func getFromEnv(name string) (string, error) {
	value := os.Getenv(name)
	if len(value) == 0 {
		return "", fmt.Errorf("Environement variable %s must be set", name)
	}

	return value, nil
}
