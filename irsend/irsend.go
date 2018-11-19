package irsend

import (
	"os/exec"
)

func Send(remoteName string, code string) error {
	cmd := exec.Command("irsend", "SEND_ONCE", remoteName, code)
	return cmd.Run()
}
