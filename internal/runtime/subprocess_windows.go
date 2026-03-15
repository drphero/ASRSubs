//go:build windows

package runtime

import (
	"os/exec"
	"syscall"
)

const windowsCreateNoWindow = 0x08000000

// ConfigureSubprocess keeps helper binaries in the background for Windows GUI runs.
func ConfigureSubprocess(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}

	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}

	cmd.SysProcAttr.CreationFlags |= windowsCreateNoWindow
	cmd.SysProcAttr.HideWindow = true
}

func subprocessRunsWithoutConsole(cmd *exec.Cmd) bool {
	return cmd != nil &&
		cmd.SysProcAttr != nil &&
		cmd.SysProcAttr.HideWindow &&
		cmd.SysProcAttr.CreationFlags&windowsCreateNoWindow != 0
}
