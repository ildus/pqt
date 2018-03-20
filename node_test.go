package pqt

import (
	"testing"
)

func TestNode(t *testing.T) {
	node := makePostgresNode("master")
	node1 := makePostgresNode("master1")
	node_replica := makeReplicaNode("replica", node)

	t.Run("init", func(t *testing.T) {
		node.init()
	})
	t.Run("start", func(t *testing.T) {
		node.start()
	})
	t.Run("second instance", func(t *testing.T) {
		node1.init()
		node1.start()
	})
	t.Run("replica", func(t *testing.T) {
		node_replica.init()
		node_replica.start()
		node_replica.catchup()
	})
	t.Run("stop", func(t *testing.T) {
		node.stop()
		node1.stop()
		node_replica.stop()
	})
}
