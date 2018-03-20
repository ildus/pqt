package pqt

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"strconv"
	"strings"
)

type ProcessType byte

const (
	UNKNOWN    ProcessType = iota
	POSTMASTER ProcessType = iota
)

type Process struct {
	Type      ProcessType
	CmdLine   string
	Pid       int
	ParentPid int
}

func (process *Process) Children() (result []*Process) {
	var out bytes.Buffer
	cmd := exec.Command("pgrep", "-P", strconv.Itoa(process.Pid))
	cmd.Stdout = &out
	err := cmd.Run()

	if err != nil {
		log.Panic("pgrep launch error: ", err)
	}

	pids := strings.Split(out.String(), "\n")
	for _, spid := range pids {
		if spid == "" {
			continue
		}

		pid, err := strconv.Atoi(spid)
		if err != nil {
			log.Panicf("can't convert pgrep line ('%s') to int", spid)
		}
		child := getProcessByPid(pid)
		if child != nil {
			result = append(result, child)
		}
	}
	return result
}

func getProcessType(pid int) (result ProcessType) {
	result = UNKNOWN
	return result
}

func getProcessByPid(pid int) (result *Process) {
	if pid <= 0 {
		return nil
	}

	procDir := fmt.Sprintf("/proc/%d/", pid)
	cmdline, err := ioutil.ReadFile(procDir + "cmdline")
	if err != nil {
		log.Panicf("can't read /proc/%d/cmdline", pid)
	}

	statline, err := ioutil.ReadFile(procDir + "stat")
	if err != nil {
		log.Panicf("can't read /proc/%d/stat", pid)
	}
	ppid, err := strconv.Atoi(strings.Split(string(statline), " ")[3])
	if err != nil {
		log.Panicf("can't read parent pid")
	}

	result = &Process{
		Pid:       pid,
		CmdLine:   string(cmdline),
		ParentPid: ppid,
		Type:      getProcessType(pid),
	}
	return result
}
