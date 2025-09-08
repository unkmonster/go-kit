package reqmeta

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseMessage(t *testing.T) {
	msg := &ExampleRequest{
		Id: 5,
	}

	res, err := parseMessage(msg)
	require.NoError(t, err)

	require.Equal(t, "order", res.ResourceType)
	require.Equal(t, msg.Id, res.ResourceId)
	require.Equal(t, "REMOVE", res.Action)
	require.True(t, res.IsSelfHold)
	require.False(t, res.IsCollection)
}

func TestMissingIdField(t *testing.T) {
	msg := &MissingIdField{}
	res, err := parseMessage(msg)
	require.NoError(t, err)

	require.Empty(t, res.ResourceId)
}
