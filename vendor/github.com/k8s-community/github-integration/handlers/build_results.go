package handlers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/k8s-community/github-integration/client"
	"github.com/k8s-community/github-integration/models"
	"github.com/takama/router"
)

// BuildResultsHandler handles and stores results of the building process.
func (h *Handler) BuildResultsHandler(c *router.Control) {
	h.Infolog.Print("Received build-results request...")

	var build client.BuildResults
	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		h.Errlog.Printf("couldn't read request body: %s", err)
		c.Code(http.StatusBadRequest).Body(nil)
		return
	}

	err = json.Unmarshal(body, &build)
	if err != nil {
		h.Errlog.Printf("couldn't validate request body: %s", err)
		c.Code(http.StatusBadRequest).Body(nil)
		return
	}

	var result = &models.Build{
		UUID:       build.UUID,
		Username:   build.Username,
		Repository: build.Repository,
		Commit:     build.CommitHash,
		Passed:     build.Passed,
		Log:        build.Log,
	}
	err = h.DB.Save(result)

	if err != nil {
		h.Errlog.Printf("Couldn't save results of build: '%+v', build: '%v'", err, result)
		c.Code(http.StatusInternalServerError).Body("Couldn't save results of build " + build.UUID)
		return
	}

	c.Code(http.StatusCreated).Body("Document uuid: " + build.UUID)
}
