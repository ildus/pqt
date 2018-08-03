package main

import (
	"github.com/ildus/pqt"
	"sync"
)

var (
	wg sync.WaitGroup
)

const (
	threadsCount int = 5
	queriesCount int = 10000
)

func make_queries(node *pqt.PostgresNode) {
	for i := 0; i < queriesCount; i += 1 {
		conn := pqt.MakePostgresConn(node, "dattest1")
		conn.Execute("begin")
		conn.Execute("update t set a = a + 1")
		conn.Execute("commit")
		conn.Close()
	}
	wg.Done()
}

func main() {
	node := pqt.MakePostgresNode("main")
	node.Init()
	node.AppendConf("postgresql.conf", "lc_monetary=C")
	node.Start()
	node.Execute("postgres", "create database dattest1")
	node.Execute("dattest1", "create table t(a int)")

	for i := 0; i < threadsCount; i += 1 {
		wg.Add(1)
		go make_queries(node)
	}
	wg.Wait()

	node.Stop()
}
