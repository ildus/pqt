pqt
=========================

`pqt` makes easier to manage postgres nodes for testing and other purposes.

Now it can:

* Create, initialize a database, start, stop, restart nodes with
default or custom configuration.
* Trace the nodes (with debug breakpoints).
* Easy 

Installation
-------------

```
go get github.com/ildus/pqt
```

Usage
------

Create a node and its replica.

```
node := MakePostgresNode("master")
node_replica := MakeReplicaNode("replica", node)
```

Start the nodes and make replica to catchup to master.

```
node.Init()
node.Start()

node_replica.Init()
node_replica.Start()
node_replica.Catchup()
```

Get some data from the node.

```
var pid int
rows := node.Fetch("select pg_backend_pid()")
for rows.Next() {
	rows.Scan(&pid)
	break
}
rows.Close()
```

Or make a query without returned data.

```
node.Execute("create table one(a text)")
```

Or make a connection and reuse it for queries.

```
conn := MakePostgresConn(node, "postgres")
conn.Execute("discard all");
```

Get node processes and put some breakpoint on some of them.
This is mostly useful for extension testing.

```
process := node.GetProcess()
children := process.Children()
for _, child := range children {
	if child.Pid == 4567 {
		debugger = MakeDebugger(child)
		breakpoint = debugger.CreateBreakpoint("pg_backend_pid", func() error {
			catched += 1
			return nil
		})
	}
}
```
