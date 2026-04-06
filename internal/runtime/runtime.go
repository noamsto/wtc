package runtime

import (
	"os"
	"os/exec"
)

// Runtime holds which optional tools are available.
type Runtime struct {
	HasTmux   bool
	HasZoxide bool
	HasGh     bool
	InTmux    bool
	NoSwitch  bool
	Quiet     bool
	Yes       bool
}

// Detect probes the environment for available tools.
func Detect() Runtime {
	_, hasTmux := exec.LookPath("tmux")
	_, hasZoxide := exec.LookPath("zoxide")
	_, hasGh := exec.LookPath("gh")

	return Runtime{
		HasTmux:   hasTmux == nil,
		HasZoxide: hasZoxide == nil,
		HasGh:     hasGh == nil,
		InTmux:    os.Getenv("TMUX") != "",
	}
}

// TmuxActive returns true if tmux integration should be used.
func (r Runtime) TmuxActive() bool {
	return r.HasTmux && !r.NoSwitch && r.InTmux
}
