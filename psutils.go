package pqt

type ProcessType byte

const (
	POSTMASTER ProcessType = iota
)

type Process struct {
	ProcessType byte
	CmdLine     string
	Pid         int
	ParentPid   int
}

func (process *Process) Children() []*Process {
	return nil
}

func getProcessByPid(pid int) *Process {
	return nil
}
