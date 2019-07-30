package magelib

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/apex/log"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/pkg/errors"

	"github.com/lalloni/go-archiver"
	"github.com/lalloni/magelib/helpers"
)

// Clean target
func Clean() {
	sh.Rm("target")
}

// Run tests
func Test() error {
	return sh.RunV("go", "test", "./...")
}

// Run source code static analysis
func Lint() error {
	return helpers.RunLinter("run")
}

// Run source code static analysis and tests
func Verify() {
	mg.Deps(Lint, Test)
}

// Run compilation of all packages
func Compile() error {
	return sh.Run("go", "build", "./...")
}

// Run command binaries compilation from cmd/* into target/bin
func Build() error {
	ss, err := filepath.Glob("cmd/*")
	if err != nil {
		return err
	}
	for _, s := range ss {
		stat, err := os.Stat(s)
		if err != nil {
			return err
		}
		if stat.IsDir() {
			for _, os := range []string{"linux", "windows", "darwin"} {
				env := map[string]string{"GOOS": os, "GOARCH": "amd64"}
				out := filepath.Base(s)
				if os == "windows" {
					out += ".exe"
				}
				err := sh.RunWith(env, "go", "build", "-o", filepath.Join("target", "bin", os+"-amd64", out), s)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Package commands binaries
func Package() error {
	mg.Deps(Clean, Build)
	p := filepath.Join("target", "commands.zip")
	log.Infof("packaging binaries into %s", p)
	os.MkdirAll(filepath.Dir(p), 0777)
	a, err := archiver.NewZip(p)
	if err != nil {
		return err
	}
	defer a.Close()
	fs, err := filepath.Glob(filepath.Join("target", "bin", "*", "*"))
	if err != nil {
		return err
	}
	err = a.AddAll(fs, func(n string) string {
		d, f := filepath.Split(n)
		d = filepath.Base(d)
		nn := filepath.Join(d, f)
		log.Infof("adding %s as %s", n, nn)
		return nn
	})
	if err != nil {
		return err
	}
	return a.Close()
}

// Start GoConvey (http://goconvey.co/)
func Convey() error {
	err := helpers.RunPackr("clean")
	if err != nil {
		return err
	}
	return helpers.RunGoConvey("-port=9999", "-watchedSuffixes=.go,.yaml", "-packages=1")
}

// Execute release process
func Release() error {
	log.Info("checking parameters")
	version := os.Getenv("ver")
	if version == "" {
		return errors.New(`Version is required for release.
You must set the version to be released using the environment variable 'ver'.
On unix-like shells you could do something like:
    env ver=1.2.3 mage release`)
	}
	if _, err := semver.NewVersion(version); err != nil {
		return errors.Wrapf(err, "checking syntax of version %q", version)
	}

	tag := "v" + version
	log.Infof("releasing version %s with tag %s", version, tag)

	log.Info("checking release tag does not exist")
	out, err := sh.Output("git", "tag")
	if err != nil {
		return errors.Wrap(err, "getting git tags")
	}
	s := bufio.NewScanner(strings.NewReader(out))
	for s.Scan() {
		if tag == s.Text() {
			return errors.Errorf("release tag %q already exists", tag)
		}
	}

	log.Info("updating generated resources")

	log.Info("checking working tree is not dirty")
	out, err = sh.Output("git", "status", "-s")
	if err != nil {
		return errors.Wrap(err, "getting git status")
	}
	if len(out) > 0 {
		return errors.Errorf("working directory is dirty")
	}

	log.Info("running linter, compiler & tests")
	mg.Deps(Compile, Lint, Test)

	log.Infof("creating tag %s", tag)
	if err := sh.RunV("git", "tag", "-s", "-m", "Release "+version, tag); err != nil {
		return errors.Wrap(err, "creating git tag")
	}

	log.Infof("pushing tag %s to 'origin' remote", tag)
	if err := sh.RunV("git", "push", "origin", tag); err != nil {
		return errors.Wrap(err, "pushing tag to origin remote")
	}

	log.Infof("pushing current branch to 'origin' remote", tag)
	if err := sh.RunV("git", "push", "origin"); err != nil {
		return errors.Wrap(err, "pushing current branch to origin remote")
	}

	log.Info("release successfuly completed")

	return nil
}

// Build a static binary for the build program (this program)
func Buildbuild() error {
	return sh.RunV("mage", "-compile", "magestatic")
}
