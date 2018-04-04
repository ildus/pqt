package pqt

import (
	"debug/dwarf"
	"debug/elf"
	"fmt"
	dlv_api "github.com/derekparker/delve/service/api"
	dlv_debug "github.com/derekparker/delve/service/debugger"
	sys "golang.org/x/sys/unix"
	"log"
)

var (
	dwarfData *dwarf.Data = nil
)

type BreakpointCallback func() error
type Debugger struct {
	ApiDebugger *dlv_debug.Debugger
	Process     *Process
	breakpoints []*dlv_api.Breakpoint
}

func getFunctionAddr(funcName string) (uint64, error) {
	if dwarfData == nil {
		log.Panic("debug information should be set up")
	}

	reader := dwarfData.Reader()
	for {
		entry, err := reader.Next()
		if err != nil {
			log.Panic("dwarf data reading error: ", err)
		}
		if entry == nil {
			break
		}

		if entry.Tag == dwarf.TagSubprogram {
			name := entry.Val(dwarf.AttrName).(string)
			if name != funcName {
				continue
			}

			addrAttr := entry.Val(dwarf.AttrLowpc)
			if addrAttr == nil {
				return 0, fmt.Errorf("symbol %q has no LowPC attribute", name)
			}
			addr, ok := addrAttr.(uint64)
			if !ok {
				return 0, fmt.Errorf("symbol %q has non-uint64 LowPC attribute", name)
			}
			return addr, nil
		}
	}
	return 0, fmt.Errorf("function is not found")
}

func setupDebugInformation(path string) {
	f, err := elf.Open(path)
	if err != nil {
		log.Panic("can't open binary: ", err)
	}

	data, err := f.DWARF()
	if err != nil {
		log.Panic("can't get dwarf information from binary: ", err)
	}
	dwarfData = data
}

func (d *Debugger) Stop() error {
	return sys.Kill(d.Process.Pid, sys.SIGSTOP)
}

func (d *Debugger) Continue() error {
	return sys.Kill(d.Process.Pid, sys.SIGCONT)
}

func MakeDebugger(node *PostgresNode, p *Process) *Debugger {
	if dwarfData == nil {
		setupDebugInformation(getBinPath("postgres"))
	}

	config := dlv_debug.Config{
		AttachPid:  p.Pid,
		WorkingDir: node.baseDirectory,
		Backend:    "lldb",
	}
	debugger, err := dlv_debug.New(&config, nil)
	if err != nil {
		log.Fatal("can't create debugger process: ", err)
	}

	d := &Debugger{
		ApiDebugger: debugger,
		Process:     p,
	}
	return d
}

func (debugger *Debugger) CreateBreakpoint(funcName string,
	callback BreakpointCallback) {

	addr, err := getFunctionAddr(funcName)
	if err != nil {
		log.Fatal("can't find function addr: ", err)
	}

	bp := &dlv_api.Breakpoint{
		Addr: addr,
	}

	_, err = debugger.ApiDebugger.CreateBreakpoint(bp)
	if err == nil {
		log.Printf("breakpoint set")
	}
}
