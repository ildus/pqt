package pqt

import (
	"debug/dwarf"
	"debug/elf"
	"log"
)

var (
	dwarfData *dwarf.Data = nil
)

func getFunctionAddr(funcName string) uint64 {
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
			if name == funcName {
				if loc, ok := entry.Val(dwarf.AttrLocation).(uint64); ok {
					return loc
				}
			}
		}
	}
	return 0
}

func setupDebugInformation() {
	f, err := elf.Open(getBinPath("postgres"))
	if err != nil {
		log.Panic("can't open 'postgres' binary")
	}

	data, err := f.DWARF()
	if err != nil {
		log.Panic("can't get dwarf information from postgres binary")
	}
	dwarfData = data
}
