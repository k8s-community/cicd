package runners

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"bufio"
	"io"

	"github.com/Sirupsen/logrus"
	"github.com/k8s-community/cicd"
	"github.com/k8s-community/cicd/builder/task"
	ghIntegr "github.com/k8s-community/github-integration/client"
)

// Local represent simple local builder (it runs tasks on current environment)
type Local struct {
	log logrus.FieldLogger
}

// NewLocal returns an instance of Local runner
func NewLocal(log logrus.FieldLogger) *Local {
	return &Local{
		log: log,
	}
}

// Process do CICD work: go get of repo, git checkout to given commit, make test and make deploy
func (runner *Local) Process(taskItem task.CICD) {
	logger := runner.log.WithFields(logrus.Fields{"source": taskItem.Prefix, "namespace": taskItem.Namespace, "repo": taskItem.Repo, "commit": taskItem.Commit})

	// TODO: it's good to use something like build.Default.GOPATH, but it doesn't work with daemon
	gopath := os.Getenv("GOPATH")

	url := fmt.Sprintf("%s/%s/%s", taskItem.Prefix, taskItem.Namespace, taskItem.Repo)
	dir := fmt.Sprintf("%s/src/%s", gopath, url)

	logger.Infof("Remove dir %s", dir)
	err := os.RemoveAll(dir)
	processCommandResult(taskItem.ID, taskItem.Callback, "", err)
	if err != nil {
		logger.Errorf("Couldn't remove directory %s: %s", dir, err)
		return
	}

	var output string

	out, err := runCommand(logger, []string{}, gopath, "go", "get", "-v", "-d", url+"/...")
	output += out
	processCommandResult(taskItem.ID, taskItem.Callback, output, err)
	if err != nil {
		logger.Errorf("Go get returned error: %v", err)
	}

	out, err = runCommand(logger, []string{}, dir, "git", "checkout", taskItem.Commit)
	output += out
	processCommandResult(taskItem.ID, taskItem.Callback, output, err)
	if err != nil {
		return
	}

	buildPath, version, err := parseOriginalMakefile(gopath + "/src/" + url + "/Makefile")
	if err != nil {
		logger.Errorf("Makefile reading failed: %s", err)
		processCommandResult(
			taskItem.ID, taskItem.Callback, "", fmt.Errorf("couldn't open the original Makefile"),
		)
		return
	}
	if len(buildPath) == 0 {
		buildPath = "cmd"
	}

	// Prepare typical Makefile by template from k8s-community/k8sapp
	out, err = runCommand(
		logger, []string{}, dir, "cp",
		os.Getenv("GOPATH")+"/src/github.com/k8s-community/cicd/templates/Makefile.tpl", "./Makefile",
	)
	output += out
	processCommandResult(taskItem.ID, taskItem.Callback, output, err)
	if err != nil {
		return
	}

	if len(taskItem.Version) > 0 {
		version = taskItem.Version
	}

	userEnv := []string{
		"NAMESPACE=" + taskItem.Namespace,
		"APP=" + taskItem.Repo,
		"PROJECT=" + url,
		"BUILD_PATH=" + buildPath,
		"KUBE_CONTEXT=" + "gke_sofia-218312_europe-west1-b_sofia", // todo: remove this spike
		"RELEASE=" + version,
		"REGISTRY=" + "gcr.io/sofia-218312", // todo: remove this spike
	}

	out, err = runCommand(logger, userEnv, dir, "make", "test")
	output += out
	processCommandResult(taskItem.ID, taskItem.Callback, output, err)
	if err != nil {
		return
	}

	if taskItem.Type == cicd.TaskDeploy {
		out, err = runCommand(logger, userEnv, dir, "make", "deploy")
		output += out
		processCommandResult(taskItem.ID, taskItem.Callback, output, err)
		if err != nil {
			return
		}
	}

	taskItem.Callback(taskItem.ID, ghIntegr.StateSuccess, output)
}

func runCommand(logger logrus.FieldLogger, env []string, dir, name string, arg ...string) (string, error) {
	logger = logger.WithFields(logrus.Fields{
		"command":        name + " " + strings.Join(arg, " "),
		"additional_env": strings.Join(env, " "),
	})

	logger.Infof("Execute command...")
	command := exec.Command(name, arg...)

	osEnv := append(os.Environ(), env...)
	command.Env = osEnv
	command.Dir = dir

	out, err := command.CombinedOutput()
	commandOut := string(out)

	if len(out) > 0 {
		logger.Info(commandOut)
	}

	if err != nil {
		logger.Errorf("Command failed: %s", err)
		return commandOut, err
	}

	logger.Infof("Done")
	return commandOut, nil
}

func processCommandResult(taskID string, callback task.Callback, output string, err error) {
	if err != nil {
		callback(taskID, ghIntegr.StateError, output+" \n\nError: "+err.Error())
	} else {
		callback(taskID, ghIntegr.StatePending, output)
	}
}

func parseOriginalMakefile(path string) (string, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", "", err
	}
	defer file.Close()

	var buildPath string
	var version string
	reader := bufio.NewReader(file)
	for {
		line, _, err := reader.ReadLine()

		if err == io.EOF {
			break
		}

		if strings.HasPrefix(string(line), "BUILD_PATH?=") {
			buildPath = strings.TrimPrefix(string(line), "BUILD_PATH?=")
		}

		if strings.HasPrefix(string(line), "RELEASE?=") {
			version = strings.TrimPrefix(string(line), "RELEASE?=")
		}

		if len(buildPath) > 0 && len(version) > 0 {
			break
		}
	}

	return buildPath, version, nil
}
