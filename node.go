package pqt

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/hpcloud/tail"
	_ "github.com/lib/pq"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
)

const (
	INITIAL int = iota
	STARTED int = iota
	STOPPED int = iota
)

type PostgresNode struct {
	name     string
	host     string
	port     int
	user     string
	database string

	baseDirectory string
	dataDirectory string
	pgLogFile     string
	status        int

	defaultConnection *sql.DB
}

func tailLog(node_name string, filename string) {
	t, err := tail.TailFile(filename, tail.Config{Follow: true})
	if err != nil {
		log.Print("can't tail file: ", filename)
	}
	for line := range t.Lines {
		log.Printf("%s: %s", node_name, line.Text)
	}
}

func (node *PostgresNode) connect() *sql.DB {
	conninfo := fmt.Sprintf("postgres://%s@%s:%d/%s?sslmode=disable",
		node.user, node.host, node.port, node.database)

	db, err := sql.Open("postgres", conninfo)
	if err != nil {
		log.Panic("Can't connect to database: ", err)
	}
	return db
}

func (node *PostgresNode) fetch(sql string, params ...interface{}) *sql.Rows {
	var err error

	if node.defaultConnection == nil {
		node.defaultConnection = node.connect()
	}

	rows, err := node.defaultConnection.Query(sql, params...)
	if err != nil {
		log.Panic(err)
	}

	return rows
}

func (node *PostgresNode) execute(sql string, params ...interface{}) {
	node.fetch(sql, params...).Close()
}

func (node *PostgresNode) start(params ...string) (string, error) {
	if node.status == STARTED {
		return "", errors.New("node has been started already")
	}

	if node.status == INITIAL {
		return "", errors.New("node has not been initialized")
	}

	if node.pgLogFile == "" {
		dir := filepath.Join(node.baseDirectory, "logs")
		os.Mkdir(dir, os.ModePerm)
		node.pgLogFile = filepath.Join(dir, "postgresql.log")
	}

	args := []string{
		"-D", node.dataDirectory,
		"-l", node.pgLogFile,
		"-w", // wait
		"start",
	}
	args = append(args, params...)

	res := execUtility("pg_ctl", args...)
	node.status = STARTED
	go tailLog(node.name, node.pgLogFile)

	return res, nil
}

func (node *PostgresNode) stop(params ...string) (string, error) {
	if node.status != STARTED {
		return "", errors.New("node has not been started")
	}

	args := []string{
		"-D", node.dataDirectory,
		"-l", node.pgLogFile,
		"-w", // wait
		"stop",
	}
	args = append(args, params...)

	res := execUtility("pg_ctl", args...)
	node.status = STOPPED

	return res, nil
}

func (node *PostgresNode) init(params ...string) (string, error) {
	if node.status != INITIAL {
		return "", errors.New("node has been initialized already")
	}

	if node.baseDirectory == "" {
		var err error
		node.baseDirectory, err = ioutil.TempDir("", "pqt_")
		if err != nil {
			log.Panic("can' create temporary directory")
		}
	}

	if node.dataDirectory == "" {
		dir := filepath.Join(node.baseDirectory, "data")
		os.Mkdir(dir, os.ModePerm)
		node.dataDirectory = dir
	}

	args := []string{
		"-D", node.dataDirectory,
		"-N",
	}
	args = append(args, params...)

	res := execUtility("initdb", args...)
	node.initDefaultConf()
	node.status = STOPPED
	return res, nil
}

func (node *PostgresNode) initDefaultConf() {
	lines := `
log_statement = 'all'
fsync = off
listen_addresses = '%s'
port = %d
`

	lines = fmt.Sprintf(lines, node.host, node.port)
	confFile := filepath.Join(node.dataDirectory, "postgresql.conf")
	err := ioutil.WriteFile(confFile, []byte(lines), os.ModePerm)

	if err != nil {
		log.Panic("can't write default configuration: ", err)
	}
}

func makePostgresNode(name string) *PostgresNode {
	user, err := user.Current()
	if err != nil {
		log.Panic("can't get current user's username")
	}

	return &PostgresNode{
		name:              name,
		host:              "127.0.0.1",
		port:              getAvailablePort(),
		defaultConnection: nil,
		status:            INITIAL,
		user:              user.Username,
		database:          "postgres",
	}
}
