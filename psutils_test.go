package pqt

import (
	"testing"
)

func TestPsUtils(t *testing.T) {
	node := MakePostgresNode("master")

	t.Run("postmaster.pid", func(t *testing.T) {
		if node.Pid() != 0 {
			t.Fail()
		}
		node.Init()
		node.Start()
		if node.Pid() <= 0 {
			t.Fail()
		}
	})

	t.Run("children", func(t *testing.T) {
		process := node.GetProcess()
		if process.CmdLine == "" || process.Pid != node.Pid() {
			t.Fail()
		}
		children := process.Children()
		if len(children) == 0 {
			t.Fail()
		}
		for _, child := range children {
			if child.Pid == 0 || child.CmdLine == "" {
				t.Fail()
			}
			if child.ParentPid != process.Pid {
				t.Fail()
			}
		}
	})

	node.Stop()
}
