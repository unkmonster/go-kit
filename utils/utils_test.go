package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCallerName(t *testing.T) {
	caller := CallerName()
	require.Contains(t, caller, "TestCallerName")
}

func TestShortCallerName(t *testing.T) {
	name := ShortCallerName()
	require.Equal(t, "utils.TestShortCallerName", name)
}
