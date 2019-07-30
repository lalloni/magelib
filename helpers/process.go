package helpers

import (
	"bytes"
	"io"
	"os/exec"

	log	"github.com/sirupsen/logrus"
)

func RunFilter(data []byte, command string, args ...string) ([]byte, error) {
	cmd := exec.Command(command, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	go func() {
		defer stdin.Close()
		_, err := io.Copy(stdin, bytes.NewReader(data))
		if err != nil {
			log.Printf("error sending input to %s: %s", command, err)
		}
	}()
	return cmd.CombinedOutput()
}
