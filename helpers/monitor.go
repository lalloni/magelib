package helpers

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bmatcuk/doublestar"
	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
)

type Event struct {
	Path string
	Type string
}

func (e *Event) String() string {
	return fmt.Sprintf(`"%s": %s`, e.Path, e.Type)
}

func Monitor(path string, c chan Event, patterns ...string) error {
	paths, err := dirs(path)
	if err != nil {
		return err
	}
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	for _, path := range paths {
		if apply(path, patterns) {
			log.Infof("Monitoring %q", path)
			err := w.Add(path)
			if err != nil {
				return err
			}
		}
	}
	go func() {
		for e := range w.Events {
			log.Infof("Received %+v", e)
			if e.Op == fsnotify.Create {
				i, err := os.Stat(e.Name)
				if err != nil && !os.IsNotExist(err) {
					log.Error(err)
				}
				if i.IsDir() {
					err := w.Add(i.Name())
					if err != nil && !os.IsNotExist(err) {
						log.Error(err)
					}
				}
			}
			c <- Event{
				Path: e.Name,
				Type: e.Op.String(),
			}
		}
	}()
	return nil
}

func dirs(base string) ([]string, error) {
	res := []string{}
	err := filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			res = append(res, path)
		}
		return nil
	})
	return res, err
}

func apply(path string, patterns []string) bool {
	include := true
	for _, pattern := range patterns {
		m, err := doublestar.Match(pattern[1:], path)
		if err != nil {
			log.Error(err)
			continue
		}
		if pattern[0] == '+' && m {
			log.Infof("Including %q after matching %q", path, pattern)
			include = true
			break
		}
		if pattern[0] == '-' && m {
			log.Infof("Excluding %q after matching %q", path, pattern)
			include = false
			break
		}
	}
	return include
}
