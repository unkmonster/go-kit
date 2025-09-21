package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCallerName(t *testing.T) {
	caller := callerName(1)
	require.Contains(t, caller, "TestCallerName")
}
