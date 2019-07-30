package helpers

import (
	"os/exec"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func CheckInstall(cmd string, install func() error) error {
	_, err := exec.LookPath(cmd)
	if err != nil {
		log.Printf("running installer for %s", cmd)
		err := install()
		if err != nil {
			return err
		}
		_, err = exec.LookPath(cmd)
		if err != nil {
			return errors.Errorf("installing %s: not found after running installer", cmd)
		}
	}
	return nil
}

func RunPackr(args ...string) error {
	mg.Deps(CheckInstallPackr)
	env := map[string]string{"GO111MODULE": "on"}
	return sh.RunWith(env, "packr2", args...)
}

func CheckInstallPackr() error {
	return CheckInstall("packr2", func() error {
		return sh.Run("go", "get", "github.com/gobuffalo/packr/v2/packr2")
	})
}

func RunLinter(args ...string) error {
	mg.Deps(CheckInstallLinter)
	return sh.RunV("golangci-lint", args...)
}

func CheckInstallLinter() error {
	return CheckInstall("golangci-lint", func() error {
		return sh.Run("go", "get", "github.com/golangci/golangci-lint/cmd/golangci-lint")
	})
}

func RunGoConvey(args ...string) error {
	mg.Deps(CheckInstallGoConvey)
	return sh.RunV("goconvey", args...)
}

func CheckInstallGoConvey() error {
	return CheckInstall("goconvey", func() error {
		return sh.Run("go", "get", "github.com/smartystreets/goconvey")
	})
}

func RunGit(args ...string) error {
	return sh.RunV("git", args...)
}
