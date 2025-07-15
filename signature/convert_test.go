package signature

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMapToQuery(t *testing.T) {
	m := map[string]any{
		"num":  2,
		"a":    1,
		"str":  "hello",
		"bool": true,
	}

	queryStr := MapToSortedQueryStr(m, "a")

	require.Equal(t, "bool=true&num=2&str=hello", queryStr)
}
