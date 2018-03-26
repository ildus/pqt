package pqt

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTrace(t *testing.T) {
	setupDebugInformation("test")

	addr, _ := getFunctionAddr("some_func")
	assert.Equal(t, addr, uint64(0x5fa))
}
