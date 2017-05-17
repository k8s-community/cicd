package handlers

import (
	"encoding/json"
	"github.com/Sirupsen/logrus"
	"github.com/k8s-community/cicd"
	"github.com/k8s-community/cicd/builder"
	"github.com/satori/go.uuid"
	"github.com/takama/router"
	"net/http"
)

// Build is a handler to process Build requests
type Build struct {
	log logrus.FieldLogger
}

// NewBuild returns an instance of Build
func NewBuild(log logrus.FieldLogger) *Build {
	return &Build{
		log: log,
	}
}

// Run handles build running
func (b *Build) Run(c *router.Control) {
	requestID := uuid.NewV4().String()
	b.log = b.log.WithField("requestID", requestID)

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
	go b.processBuild(req)

	data := &cicd.Build{RequestID: requestID}
	response := cicd.BuildResponse{Data: data}
	c.Code(http.StatusCreated).Body(response)
}

func (b *Build) processBuild(req *cicd.BuildRequest) {
	err := builder.Process(b.log, "github.com", req.Username, req.Repository, req.CommitHash)

	// TODO: send result of processing to integration service!
	if err != nil {

	}
}
