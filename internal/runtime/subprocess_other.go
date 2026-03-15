//go:build !windows

package runtime

import "os/exec"

func ConfigureSubprocess(cmd *exec.Cmd) {
}

func subprocessRunsWithoutConsole(cmd *exec.Cmd) bool {
	return false
}
