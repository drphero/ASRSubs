//go:build windows

package runtime

import (
	"os/exec"
	"testing"
)

func TestConfigureSubprocessHidesWindowsConsole(t *testing.T) {
	cmd := exec.Command("cmd", "/c", "echo", "ok")

	ConfigureSubprocess(cmd)

	if !subprocessRunsWithoutConsole(cmd) {
		t.Fatal("expected subprocess to run without a visible console")
	}
}
