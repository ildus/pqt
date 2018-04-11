package pqt

import (
	"debug/dwarf"
	"debug/elf"
	"fmt"
	sys "golang.org/x/sys/unix"
	"log"
	"syscall"
)

var (
	dwarfData *dwarf.Data = nil
)

type BreakpointCallback func() error

type breakpoint struct {
	addr        uint64
	original    []byte
	callback    BreakpointCallback
	description string
}

type Debugger struct {
	Process        *Process
	BreakpointChan chan *breakpoint
	Thread         Thread
	Breakpoints    map[uint64]*breakpoint
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

func setPC(pid int, pc uint64) {
	var regs syscall.PtraceRegs
	err := syscall.PtraceGetRegs(pid, &regs)
	if err != nil {
		log.Fatal(err)
	}
	regs.SetPC(pc)
	err = syscall.PtraceSetRegs(pid, &regs)
	if err != nil {
		log.Fatal(err)
	}
}

func getPC(pid int) uint64 {
	var regs syscall.PtraceRegs
	err := syscall.PtraceGetRegs(pid, &regs)
	if err != nil {
		log.Fatal(err)
	}
	return regs.PC()
}

func writeBreakpoint(pid int, breakpoint uintptr) []byte {
	original := make([]byte, 1)
	_, err := syscall.PtracePeekData(pid, breakpoint, original)
	if err != nil {
		log.Fatal("can't peek data for breakpoint: ", err)
	}
	_, err = syscall.PtracePokeData(pid, breakpoint, []byte{0xCC})
	if err != nil {
		log.Fatal("can't poke data for breakpoint: ", err)
	}
	return original
}

func clearBreakpoint(pid int, breakpoint uintptr, original []byte) {
	_, err := syscall.PtracePokeData(pid, breakpoint, original)
	if err != nil {
		log.Fatal("can't poke data that removes breakpoint: ", err)
	}
}

func MakeDebugger(node *PostgresNode, p *Process) *Debugger {
	path := getBinPath("postgres")
	log.Printf(path)
	if dwarfData == nil {
		setupDebugInformation(path)
	}

	breakpointChan := make(chan *breakpoint, 1)
	debugger := &Debugger{
		Process:        p,
		BreakpointChan: breakpointChan,
	}

	thread := makeThread(func() {
		var ws syscall.WaitStatus

		err := syscall.PtraceAttach(p.Pid)
		if err != nil {
			log.Fatal("can't attach: ", err)
		}

		// should stop after attach
		_, err = syscall.Wait4(p.Pid, &ws, syscall.WALL, nil)
		if !ws.Stopped() {
			log.Fatal("could not attach: ", err)
		}
		startingPC := getPC(p.Pid)
		syscall.PtraceCont(p.Pid, 0)

		for {
			var msg uint
			_, err := syscall.Wait4(p.Pid, &ws, syscall.WALL, nil)
			if err != nil {
				log.Fatal(err)
			}
			msg, err = syscall.PtraceGetEventMsg(p.Pid)
			log.Print(msg)

			if ws.StopSignal() == sys.SIGTRAP {
				curAddr := getPC(p.Pid)
				br, ok := debugger.Breakpoints[curAddr]
				if ok {
					clearBreakpoint(p.Pid, uintptr(curAddr), br.original)
					log.Println(br)
				}
			} else if ws.Signaled() || ws.Exited() {
				log.Println("exited")
				break
			} else if ws.Stopped() {
				select {
				case br := <-breakpointChan:
					resaddr := startingPC + br.addr
					br.original = writeBreakpoint(p.Pid, uintptr(resaddr))
					debugger.Breakpoints[resaddr] = br
				default:
					break
				}
				log.Println("stopped")
			}
			log.Println("continued")
			syscall.PtraceCont(p.Pid, 0)
		}
	})
	debugger.Thread = thread
	return debugger
}

func (debugger *Debugger) CreateBreakpoint(funcName string,
	callback BreakpointCallback) {

	addr, err := getFunctionAddr(funcName)
	log.Printf("addr for %s is %d", funcName, addr)
	if err != nil {
		log.Fatal("can't find function addr: ", err)
	}
	br := &breakpoint{
		addr:        addr,
		callback:    callback,
		description: funcName,
	}
	debugger.BreakpointChan <- br
	syscall.Kill(debugger.Process.Pid, syscall.SIGSTOP)
}

func (debugger *Debugger) Detach() {
	err := syscall.PtraceDetach(debugger.Process.Pid)
	if err != nil {
		log.Fatal("can't detach: ", err)
	}
}

func (debugger *Debugger) Stop() error {
	return sys.Kill(debugger.Process.Pid, sys.SIGSTOP)
}

func (debugger *Debugger) Continue() error {
	return sys.Kill(debugger.Process.Pid, sys.SIGCONT)
}
