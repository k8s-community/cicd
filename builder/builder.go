package builder

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Sirupsen/logrus"
)

// Process do CICD work: go get of repo, git checkout to given commit, make test and make deploy
func Process(log logrus.FieldLogger, prefix, user, repo, commit string) (string, error) {
	logger := log.WithFields(logrus.Fields{"source": prefix, "user": user, "repo": repo, "commit": commit})

	// TODO: it's good to use something like build.Default.GOPATH, but it doesn't work with daemon
	gopath := os.Getenv("GOPATH")

	url := fmt.Sprintf("%s/%s/%s", prefix, user, repo)
	dir := fmt.Sprintf("%s/src/%s", gopath, url)

	logger.Infof("Remove dir %s", dir)
	err := os.RemoveAll(dir)
	if err != nil {
		logger.Errorf("Couldn't remove directory %s: %s", dir, err)
		return "", err
	}

	var output string

	out, err := runCommand(logger, []string{}, gopath, "go", "get", "-u", url)
	output += out
	if err != nil {
		return out, err
	}

	out, err = runCommand(logger, []string{}, dir, "git", "checkout", commit)
	output += out
	if err != nil {
		return out, err
	}

	out, err = runCommand(logger, []string{}, dir, "make", "test")
	output += out
	if err != nil {
		return out, err
	}

	out, err = runCommand(logger, []string{"NAMESPACE=" + user}, dir, "make", "deploy")
	output += out
	if err != nil {
		return out, err
	}

	return output, nil
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
