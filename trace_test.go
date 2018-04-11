package pqt

import (
	"fmt"
	"testing"
)

func TestTrace(t *testing.T) {
	var debugger *Debugger

	node := MakePostgresNode("master")
	node.Init()
	node.Start()

	catched := 0
	process := node.GetProcess()
	children := process.Children()
	for _, child := range children {
		if child.Type == AutovacuumLauncher {
			debugger = MakeDebugger(node, child)
			debugger.CreateBreakpoint("pg_usleep", func() error {
				catched += 1
				return nil
			})
		}
	}

	node.Execute("select pg_sleep(1)")
	node.Execute("select pg_sleep(1)")
	fmt.Println("catched:", catched)
	node.Stop()
}
