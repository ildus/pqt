package pqt

import (
	"debug/dwarf"
	"debug/elf"
	"fmt"
	"log"
)

var (
	dwarfData *dwarf.Data = nil
)

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
