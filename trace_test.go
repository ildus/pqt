package pqt

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTrace(t *testing.T) {
	setupDebugInformation()

	t.Run("lookup_function", func(t *testing.T) {
		assert.NotEqual(t, getFunctionAddr("LWLockAcquire"), 0)
	})
}
