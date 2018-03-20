package pqt

type Process struct {
	cmdline    string
	pid        int
	parent_pid int
}

func (process *Process) children() []*Process {
	return nil
}

func get_process_by_pid(pid int) *Process {
	return nil
}
