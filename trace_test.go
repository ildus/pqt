package pqt

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTrace(t *testing.T) {
	var debugger *Debugger

	node := MakePostgresNode("master")
	node.Init()
	node.Start()

	process := node.GetProcess()
	children := process.Children()
	for _, child := range children {
		if child.Type == AutovacuumLauncher {
			debugger = MakeDebugger(node, child)
			assert.NotEqual(t, debugger.ApiDebugger, nil)
			debugger.CreateBreakpoint("LWLockAcquire", func() error {
				return nil
			})
		}
	}
}
