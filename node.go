package pqt

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"github.com/hpcloud/tail"
	_ "github.com/lib/pq"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	INITIAL int = iota
	STARTED int = iota
	STOPPED int = iota
)

var (
	pqtLogSetUp bool = false
	pqtLogFile       = flag.String("pqt-log", "", "Collect logs to one place")
)

// postmaster node
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
	logChan       chan bool

	defaultConnection *sql.DB
}

// Reads new lines from postgres logs.
func tailLog(node *PostgresNode, filename string) {
	t, err := tail.TailFile(filename, tail.Config{Follow: true})
	if err != nil {
		log.Print("can't tail file: ", filename)
	}

	for line := range t.Lines {
		flags := log.Flags()
		log.SetFlags(0)
		log.Printf("%s: %s", node.name, line.Text)
		log.SetFlags(flags)

		select {
		case <-node.logChan:
			log.Println("node has stopped")
			break
		default:
			// pass
		}
	}
}

// Creates a new connection to node.
func (node *PostgresNode) Connect() *sql.DB {
	conninfo := fmt.Sprintf("postgres://%s@%s:%d/%s?sslmode=disable",
		node.user, node.host, node.port, node.database)

	db, err := sql.Open("postgres", conninfo)
	if err != nil {
		log.Panic("Can't connect to database: ", err)
	}
	return db
}

// Execute query and fetch resulting rows from node.
// Uses the default connection.
func (node *PostgresNode) Fetch(sql string, params ...interface{}) *sql.Rows {
	var err error

	if node.defaultConnection == nil {
		node.defaultConnection = node.Connect()
	}

	rows, err := node.defaultConnection.Query(sql, params...)
	if err != nil {
		log.Panic(err)
	}

	return rows
}

// Executes query without returning any data.
// Uses the default connection.
func (node *PostgresNode) Execute(sql string, params ...interface{}) {
	node.Fetch(sql, params...).Close()
}

// Starts a postgres node.
// The node should be initialized.
func (node *PostgresNode) Start(params ...string) (string, error) {
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
	go tailLog(node, node.pgLogFile)

	return res, nil
}

// Stops a postgres node.
func (node *PostgresNode) Stop(params ...string) (string, error) {
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

	node.logChan <- true

	return res, nil
}

// Initializes a new postgres node.
// Creates directories for logs and data, and writes
// a default configuration.
func (node *PostgresNode) Init(params ...string) (string, error) {
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

// Returns postmaster pid.
func (node *PostgresNode) Pid() int {
	if node.status != STARTED {
		return 0
	}

	pidFile := filepath.Join(node.dataDirectory, "postmaster.pid")
	data, err := ioutil.ReadFile(pidFile)
	if err != nil {
		log.Panic("can't read pid file")
	}
	pid, err := strconv.Atoi(strings.Split(string(data), "\n")[0])
	if err != nil {
		log.Panic("can't convert to pid content of ", pidFile)
	}
	return pid
}

// Returns Process instance for postmaster.
func (node *PostgresNode) GetProcess() (result *Process) {
	result = getProcessByPid(node.Pid())
	result.Type = Postmaster
	return result
}

// Makes a new postgres node using specified name.
func MakePostgresNode(name string) *PostgresNode {
	if !pqtLogSetUp {
		pqtLogSetUp = true
		flag.Parse()

		if *pqtLogFile != "" {
			f, err := os.OpenFile(*pqtLogFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
				os.ModePerm)

			if err != nil {
				log.Panic("can't open file for logging")
			}
			log.SetOutput(f)
		}
	}

	curUser, err := user.Current()
	if err != nil {
		log.Panic("can't get current user's username")
	}

	return &PostgresNode{
		name:              name,
		host:              "127.0.0.1",
		port:              getAvailablePort(),
		defaultConnection: nil,
		status:            INITIAL,
		user:              curUser.Username,
		database:          "postgres",
		logChan:           make(chan bool, 1),
	}
}
