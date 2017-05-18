package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/k8s-community/cicd/handlers"
	"github.com/k8s-community/cicd/version"
	common_handlers "github.com/k8s-community/handlers"
	"github.com/takama/router"
)

func main() {
	log := logrus.New()
	log.Formatter = new(logrus.TextFormatter)
	logger := log.WithFields(logrus.Fields{"service": "cicd"})

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
	r.Listen(hostPort)
}

func getFromEnv(name string) (string, error) {
	value := os.Getenv(name)
	if len(value) == 0 {
		return "", fmt.Errorf("Environement variable %s must be set", name)
	}

	return value, nil
}
