package pqt

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPsUtils(t *testing.T) {
	node := MakePostgresNode("master")

	t.Run("postmaster.pid", func(t *testing.T) {
		assert.Equal(t, node.Pid(), 0)
		node.Init()
		node.Start()
		assert.NotEqual(t, node.Pid(), 0)
	})

	t.Run("children", func(t *testing.T) {
		process := node.GetProcess()
		assert.NotEqual(t, process.CmdLine, "")
		assert.Equal(t, process.Pid, node.Pid())
		assert.Equal(t, process.Type, Postmaster)

		children := process.Children()
		assert.NotEqual(t, len(children), 0)

		for _, child := range children {
			assert.NotEqual(t, child.Pid, 0)
			assert.NotEqual(t, child.CmdLine, "")
			assert.Equal(t, child.ParentPid, process.Pid)
			assert.NotEqual(t, child.Type, UnknownProcess)
		}
	})

	node.Stop()
}
