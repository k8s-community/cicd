package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/k8s-community/cicd"
	"github.com/k8s-community/cicd/builder"
	ghIntegr "github.com/k8s-community/github-integration/client"
	"github.com/satori/go.uuid"
	"github.com/takama/router"
)

// Build is a handler to process Build requests
type Build struct {
	log                     logrus.FieldLogger
	githubIntegrationClient *ghIntegr.Client
}

// NewBuild returns an instance of Build
func NewBuild(log logrus.FieldLogger, ghIntClient *ghIntegr.Client) *Build {
	return &Build{
		log: log,
		githubIntegrationClient: ghIntClient,
	}
}

// Run handles build running
func (b *Build) Run(c *router.Control) {
	requestID := uuid.NewV4().String()
	b.log = b.log.WithField("requestID", requestID)
	b.log.Infof("Processing request...")

	req := new(cicd.BuildRequest)
	err := json.NewDecoder(c.Request.Body).Decode(&req)

	if err != nil {
		c.Code(http.StatusBadRequest).Body("Couldn't parse request body.")
		return
	}

	if len(req.Username) == 0 || len(req.Repository) == 0 || len(req.CommitHash) == 0 {
		c.Code(http.StatusBadRequest).Body("The fields username, repository and commitHash are required.")
		return
	}

	// TODO: manage amount of goroutines!
	// TODO: add max execution time of goroutine!!!! If processing is too slow, we need to stop it
	go b.processBuild(req, requestID)

	data := &cicd.Build{RequestID: requestID}
	response := cicd.BuildResponse{Data: data}
	c.Code(http.StatusCreated).Body(response)
}

func (b *Build) processBuild(req *cicd.BuildRequest, requestID string) {
	_, err := builder.Process(b.log, "github.com", req.Username, req.Repository, req.CommitHash)

	var state string
	var description string

	// TODO: send result of processing to integration service!
	if err != nil {
		state = ghIntegr.StateFailure
		description = fmt.Sprintf("Build failed: %s. Please, read logs for request %s", err.Error(), requestID)
	} else {
		state = ghIntegr.StateSuccess
		description = "The service was released"
	}

	callbackData := ghIntegr.BuildCallback{
		Username:    strings.ToLower(req.Username),
		Repository:  req.Repository,
		CommitHash:  req.CommitHash,
		State:       state,
		BuildURL:    "https://k8s.community", // TODO: fix it!
		Description: description,
		Context:     "k8s-community/cicd", // TODO: fix it!
	}
	err = b.githubIntegrationClient.Build.BuildCallback(callbackData)

}
