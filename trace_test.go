// +build linux

package pqt

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTrace(t *testing.T) {
	var debugger *Debugger
	var breakpoint *Breakpoint

	node := MakePostgresNode("master")
	node.Init()
	node.Start()

	catched := 0
	process := node.GetProcess()

	var pid int
	rows := node.Fetch("select pg_backend_pid()")
	for rows.Next() {
		rows.Scan(&pid)
		break
	}
	rows.Close()

	children := process.Children()
	for _, child := range children {
		if child.Pid == pid {
			debugger = MakeDebugger(child, getBinPath("postgres"))
			breakpoint = debugger.CreateBreakpoint("pg_backend_pid", func() error {
				catched += 1
				return nil
			})
		}
	}

	node.Execute("select pg_backend_pid()")
	node.Execute("select pg_backend_pid()")
	assert.Equal(t, catched, 2)
	debugger.RemoveBreakpoint(breakpoint)
	node.Execute("select pg_backend_pid()")
	assert.Equal(t, catched, 2)
	node.Stop()
}
