package pqt

import (
	"testing"
)

func TestNode(t *testing.T) {
	node := MakePostgresNode("master")
	node1 := MakePostgresNode("master1")
	node_replica := MakeReplicaNode("replica", node)

	t.Run("init", func(t *testing.T) {
		node.Init()
	})
	t.Run("start", func(t *testing.T) {
		node.Start()
	})
	t.Run("second instance", func(t *testing.T) {
		node1.Init()
		node1.Start()
	})
	t.Run("replica", func(t *testing.T) {
		node_replica.Init()
		node_replica.Start()
		node_replica.Catchup()
	})
	t.Run("stop", func(t *testing.T) {
		node.Stop()
		node1.Stop()
		node_replica.Stop()
	})
}
