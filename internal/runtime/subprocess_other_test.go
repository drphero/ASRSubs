//go:build !windows

package runtime

import (
	"os/exec"
	"testing"
)

func TestConfigureSubprocessIsNoopOffWindows(t *testing.T) {
	cmd := exec.Command("sh", "-c", "printf ok")

	ConfigureSubprocess(cmd)

	if cmd.SysProcAttr != nil {
		t.Fatal("expected non-Windows subprocess configuration to remain unset")
	}
	if subprocessRunsWithoutConsole(cmd) {
		t.Fatal("expected non-Windows subprocess helper to report false")
	}
}
