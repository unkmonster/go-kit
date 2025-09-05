package realip

import (
	"net"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseProxy(t *testing.T) {
	anyContain := func(ns []*net.IPNet, ip net.IP) bool {
		for _, n := range ns {
			if n.Contains(ip) {
				return true
			}
		}
		return false
	}

	tests := []struct {
		proxy   string
		contain string
	}{
		{
			proxy: "localhost",
		},
		{
			proxy:   "0.0.0.0/0",
			contain: "127.42.24.1",
		},
		{
			proxy:   "110.110.110.110",
			contain: "110.110.110.110",
		},
		{
			proxy:   "127.0.0.1",
			contain: "127.0.0.1",
		},
	}

	t.Run("localhost", func(t *testing.T) {
		test := tests[0]
		result, err := parseProxy(test.proxy)
		require.NoError(t, err)
		require.NotEmpty(t, result)

		require.True(
			t,
			anyContain(result, net.ParseIP("127.0.0.1")) ||
				anyContain(result, net.ParseIP("::1")),
		)
	})

	for i, test := range tests[1:] {
		t.Run(strconv.FormatInt(int64(i), 10), func(t *testing.T) {
			result, err := parseProxy(test.proxy)
			require.NoError(t, err)
			require.NotEmpty(t, result)

			ip := net.ParseIP(test.contain)
			if ip == nil {
				panic("input contain is invalid ip")
			}
			require.True(t, anyContain(result, ip))
		})
	}
}

func TestGetClientIp(t *testing.T) {
	tests := []struct {
		trusted []string
		value   string
		result  string
	}{
		{
			trusted: []string{"127.0.0.1"},
			value:   "127.0.0.1",
			result:  "127.0.0.1",
		},
		{
			trusted: []string{"127.0.0.1"},
			value:   "1.1.1.1,127.0.0.1",
			result:  "1.1.1.1",
		},
		{
			trusted: []string{}, // 全部可信
			value:   "5.5.5.5,1.1.1.1,127.0.0.1",
			result:  "5.5.5.5",
		},
	}

	for i, test := range tests {
		t.Run(strconv.FormatInt(int64(i), 10), func(t *testing.T) {
			opt := &options{}
			WithTrustedProxies(test.trusted)(opt)

			result := getClientIp(opt, test.value)
			require.Equal(t, test.result, result)
		})
	}
}
