package pqt

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
)

type ReplicaNode struct {
	PostgresNode
	master *PostgresNode
}

func makeReplicaNode(name string, master *PostgresNode) *ReplicaNode {
	node := &ReplicaNode{*makePostgresNode(name), master}
	return node
}

func (node *ReplicaNode) writeRecoveryConf() {
	lines := `
primary_conninfo = 'application_name=%s port=%d user=%s hostaddr=127.0.0.1'
standby_mode = on
`

	lines = fmt.Sprintf(lines, node.name, node.port, node.user)
	confFile := filepath.Join(node.dataDirectory, "recovery.conf")
	err := ioutil.WriteFile(confFile, []byte(lines), os.ModePerm)

	if err != nil {
		log.Panic("can't write recovery configuration: ", err)
	}
}

func (node *ReplicaNode) init(params ...string) (string, error) {
	var err error

	if node.master.status != STARTED {
		log.Panic("master node should be started")
	}

	node.baseDirectory, err = ioutil.TempDir("", "pqt_backup_")
	if err != nil {
		log.Panic("cannot create backup base directory")
	}
	node.dataDirectory = filepath.Join(node.baseDirectory, "data")
	os.Mkdir(node.dataDirectory, 0700)

	args := []string{
		"-p", strconv.Itoa(node.master.port),
		"-h", node.host,
		"-U", node.user,
		"-D", node.dataDirectory,
	}
	args = append(args, params...)
	res := execUtility("pg_basebackup", args...)
	node.initDefaultConf()
	node.writeRecoveryConf()
	node.status = STOPPED
	return res, nil
}

func (node *ReplicaNode) catchup() {
	var lsn string

	poll_lsn := "select pg_current_wal_lsn()::text"
	wait_lsn := "select pg_last_wal_replay_lsn() >= '%s'::pg_lsn"

	rows := node.master.fetch(poll_lsn)
	rows.Next()

	err := rows.Scan(&lsn)
	if err != nil {
		log.Panic("failed to poll current lsn from master")
	}
	rows.Close()

	wait_query := fmt.Sprintf(wait_lsn, lsn)
	for {
		var reached bool

		rows = node.fetch(wait_query)
		rows.Next()
		err = rows.Scan(&reached)
		rows.Close()
		if err != nil {
			log.Panic("failed to get replay lsn from replica")
		}

		if reached {
			break
		}
	}
}
