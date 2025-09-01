package realip

import (
	"net"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseProxy(t *testing.T) {
	tests := []struct {
		proxy   string
		contain string
	}{
		{
			proxy:   "localhost",
			contain: "127.0.0.1",
		},
		{
			proxy:   "0.0.0.0/0",
			contain: "127.42.24.1",
		},
		{
			proxy:   "110.110.110.110",
			contain: "110.110.110.110",
		},
	}

	for i, test := range tests {
		t.Run(strconv.FormatInt(int64(i), 10), func(t *testing.T) {
			result, err := parseProxy(test.proxy)
			require.NoError(t, err)
			require.NotEmpty(t, result)

			ip := net.ParseIP(test.contain)
			if ip == nil {
				panic("input contain is invalid ip")
			}
			require.True(t, result[0].Contains(ip))
		})
	}
}
