package realip

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
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
		// 特殊用例
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
		{
			proxy:   "192.168.0.0/16",
			contain: "192.168.0.100",
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
				panic(fmt.Sprintf("input contain is invalid ip: %s", test.contain))
			}
			require.True(t, anyContain(result, ip))
		})
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		trusted []string
		value   string
		result  string
	}{
		{
			trusted: []string{"127.0.0.1"},
			value:   "1.1.1.1,2.2.2.2,127.0.0.1",
			result:  "2.2.2.2",
		},
		{
			trusted: []string{"127.0.0.1"},
			value:   "127.0.0.1",
			result:  "", // 由于全部为可信地址，结果为空
		},
		{
			trusted: []string{"0.0.0.0/0"}, // 全部可信
			value:   "5.5.5.5,1.1.1.1,127.0.0.1",
			result:  "",
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

var _ khttp.Transporter = (*transpoter)(nil)

type transpoter struct {
	request *khttp.Request
}

// Endpoint implements http.Transporter.
func (t *transpoter) Endpoint() string {
	panic("unimplemented")
}

// Kind implements http.Transporter.
func (t *transpoter) Kind() transport.Kind {
	panic("unimplemented")
}

// Operation implements http.Transporter.
func (t *transpoter) Operation() string {
	panic("unimplemented")
}

// PathTemplate implements http.Transporter.
func (t *transpoter) PathTemplate() string {
	panic("unimplemented")
}

// ReplyHeader implements http.Transporter.
func (t *transpoter) ReplyHeader() transport.Header {
	panic("unimplemented")
}

// Request implements http.Transporter.
func (t *transpoter) Request() *khttp.Request {
	return t.request
}

// RequestHeader implements http.Transporter.
func (t *transpoter) RequestHeader() transport.Header {
	panic("unimplemented")
}

func TestRealIP(t *testing.T) {
	tests := []struct {
		trustedProxies []string
		ipHeaders      []string
		trustedHeader  string

		remoteAddr string
		headers    http.Header
		expect     string
	}{
		// 下游属于可信代理，从 xff 提取 2.2.2.2 作为结果
		{
			trustedProxies: []string{
				"192.168.0.0/16",
			},
			ipHeaders: []string{
				"X-Forwarded-For",
			},
			trustedHeader: "",
			remoteAddr:    "192.168.0.1:5000",
			headers: map[string][]string{
				"X-Forwarded-For": {"1.1.1.1,2.2.2.2"},
			},
			expect: "2.2.2.2",
		},
		// 同上，但下游不属于可信代理，使用 remoteAddr 作为结果
		{
			trustedProxies: []string{
				"192.168.0.0/16",
			},
			ipHeaders: []string{
				"X-Forwarded-For",
			},
			trustedHeader: "",
			remoteAddr:    "10.0.0.0:5000",
			headers: map[string][]string{
				"X-Forwarded-For": {"1.1.1.1,2.2.2.2"},
			},
			expect: "10.0.0.0",
		},
		// 同上，但 trusted_header 被指定，优先从此头部获取结果
		{
			trustedProxies: []string{
				"192.168.0.0/16",
			},
			ipHeaders: []string{
				"X-Forwarded-For",
			},
			trustedHeader: "Cf-Connecting-IP",
			remoteAddr:    "192.168.0.1:5000",
			headers: map[string][]string{
				"X-Forwarded-For":  {"1.1.1.1,2.2.2.2"},
				"Cf-Connecting-Ip": {"8.8.8.8"},
			},
			expect: "8.8.8.8",
		},
		// 同第一个例子，但下游和 2.2.2.2 同时属于可信代理，使用 2.2.2.2 左边的 IP 作为结果
		{
			trustedProxies: []string{
				"192.168.0.0/16",
				"2.2.2.2",
			},
			ipHeaders: []string{
				"X-Forwarded-For",
			},
			trustedHeader: "",
			remoteAddr:    "192.168.0.1:5000",
			headers: map[string][]string{
				"X-Forwarded-For": {"1.1.1.1,2.2.2.2"},
			},
			expect: "1.1.1.1",
		},
		// 同首个例子，但制定了可信标头，可是由于请求中缺少可信标头，结果不变
		{
			trustedProxies: []string{
				"192.168.0.0/16",
			},
			ipHeaders: []string{
				"X-Forwarded-For",
			},
			trustedHeader: "Cf-Connecting-Ip",
			remoteAddr:    "192.168.0.1:5000",
			headers: map[string][]string{
				"X-Forwarded-For": {"1.1.1.1,2.2.2.2"},
			},
			expect: "2.2.2.2",
		},
	}

	for i, test := range tests {
		t.Run(strconv.FormatInt(int64(i), 10), func(t *testing.T) {
			m := Server(
				log.DefaultLogger,
				WithTrustedProxies(test.trustedProxies),
				WithIpHeaders(test.ipHeaders),
				WithTrustedHeader(test.trustedHeader),
			)

			var next middleware.Handler = func(ctx context.Context, req any) (any, error) {

				val, ok := FromContext(ctx)
				if test.expect == "" {
					require.False(t, ok)
					return nil, nil
				}

				require.Equal(t, test.expect, val)
				return nil, nil
			}

			ctx := context.Background()

			req := &http.Request{
				RemoteAddr: test.remoteAddr,
				Header:     test.headers,
			}
			ctx = transport.NewServerContext(ctx, &transpoter{request: req})
			_, err := m(next)(ctx, nil)
			require.NoError(t, err)
		})
	}

}
