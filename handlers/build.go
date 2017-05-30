package handlers

import (
	"encoding/json"
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
	state                   *builder.State
	log                     logrus.FieldLogger
	githubIntegrationClient *ghIntegr.Client
}

// NewBuild returns an instance of Build
func NewBuild(state *builder.State, log logrus.FieldLogger, ghIntClient *ghIntegr.Client) *Build {
	return &Build{
		state: state,
		log:   log,
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
	namespace := strings.ToLower(req.Username)

	callback := func(state string, description string) {
		// TODO: send result of processing to integration service too!
		callbackData := ghIntegr.BuildCallback{
			Username:    req.Username,
			Repository:  req.Repository,
			CommitHash:  req.CommitHash,
			State:       state,
			BuildURL:    "https://k8s.community/" + requestID, // TODO: fix it!
			Description: "task...",                            // TODO: less than 120 symbols
			Context:     "k8s-community/cicd",                 // TODO: fix it!
		}
		err := b.githubIntegrationClient.Build.BuildCallback(callbackData)
		if err != nil {
			b.log.Error(err)
		}
	}

	// task types: test, build, release etc (instead of test)
	b.state.AddTask(callback, requestID, "test", "github.com", namespace, req.Repository, req.CommitHash)
	callback(ghIntegr.StatePending, "Task was queued")
}
