package main

import (
	"flag"

	"github.com/Sirupsen/logrus"
	"github.com/k8s-community/cicd/builder"
)

var (
	fPrefix = flag.String("prefix", "github.com", "Source code storage (to deal with 'go get')")
	fUser   = flag.String("user", "rumyantseva", "Username (part of path to repo)")
	fRepo   = flag.String("repo", "myapp", "Repository name")
	fCommit = flag.String("commit", "develop", "Commit hash or branch name")
)

// This example doesn't deal with API, it just calls processing
func main() {
	flag.Parse()
	log := logrus.New()
	builder.Process(log, *fPrefix, *fUser, *fRepo, *fCommit)
}
