package builder

import (
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"strings"

	"github.com/Sirupsen/logrus"
)

// Process do CICD work: go get of repo, git checkout to given commit, make test and make deploy
func Process(log logrus.FieldLogger, prefix, user, repo, commit string) (string, error) {
	logger := log.WithFields(logrus.Fields{"source": prefix, "user": user, "repo": repo, "commit": commit})

	url := fmt.Sprintf("%s/%s/%s", prefix, user, repo)
	dir := fmt.Sprintf("%s/src/%s", build.Default.GOPATH, url)

	out, err := runCommand(logger, []string{}, "go", "get", "-u", url)
	if err != nil {
		return out, err
	}

	err = os.Chdir(dir)
	if err != nil {
		logger.Errorf("Couldn't change directory: %s", err)
		return "", err
	}

	out, err = runCommand(logger, []string{}, "git", "checkout", commit)
	if err != nil {
		return out, err
	}

	out, err = runCommand(logger, []string{}, "make", "test")
	if err != nil {
		return out, err
	}

	out, err = runCommand(logger, []string{"NAMESPACE=" + user}, "make", "deploy")
	if err != nil {
		return out, err
	}

	return out, nil
}

func runCommand(logger logrus.FieldLogger, env []string, name string, arg ...string) (string, error) {
	logger = logger.WithFields(logrus.Fields{
		"command":        name + " " + strings.Join(arg, " "),
		"additional_env": strings.Join(env, " "),
	})

	logger.Infof("Execute command...")
	command := exec.Command(name, arg...)

	osEnv := append(os.Environ(), env...)
	command.Env = osEnv

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
