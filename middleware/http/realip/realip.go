package realip

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport/http"
)

var (
	ErrWrongContext      = errors.InternalServer("INTERNAL", "wrong context")
	ErrInvalidRemoteAddr = errors.BadRequest("INVALID_REMOTE_ADDR", "invalid remote address")
)

type realIpKey struct{}

type options struct {
	TrustedProxies []*net.IPNet
	TrustedHeader  string
	IpHeaders      []string
}

type Option func(opts *options)

func parseProxy(proxy string) (*net.IPNet, error) {
	if !strings.Contains(proxy, "/") {
		ip := net.ParseIP(proxy)
		if ip == nil {
			return nil, fmt.Errorf("invalid ip addr: %s", proxy)
		}

		if ip.To4() != nil {
			proxy = proxy + "/32"
		} else {
			proxy = proxy + "/128"
		}
	}
	_, cidr, err := net.ParseCIDR(proxy)
	if err != nil {
		return nil, err
	}
	return cidr, nil
}

func isTrustedProxy(options *options, ip net.IP) bool {
	if len(options.TrustedProxies) == 0 {
		return true
	}

	for _, proxy := range options.TrustedProxies {
		if proxy.Contains(ip) {
			return true
		}
	}
	return false
}

// validateIpHeader 反向顺序验证 ip 头，
// 返回从右往左第一个不信任的 IP, 或从右到左第一个无效 IP 的右一个 IP 作为 client ip
func validateIpHeader(options *options, value string) string {
	if value == "" {
		return ""
	}

	var result string
	ips := strings.Split(value, ",")
	for i := len(ips) - 1; i >= 0; i-- {
		ipStr := strings.TrimSpace(ips[i])
		ip := net.ParseIP(ipStr)
		if ip == nil {
			break
		}
		if i != 0 && !isTrustedProxy(options, ip) {
			break
		}
		result = ipStr
	}
	return result
}

func WithTrustedHeader(header string) Option {
	return func(opts *options) {
		opts.TrustedHeader = header
	}
}

// WithTrustedProxies 支持 IP, CIDR, 如果不指定此选项表示信任所有下游
func WithTrustedProxies(proxies []string) Option {
	return func(opts *options) {
		results := []*net.IPNet{}
		for _, proxy := range proxies {
			cidr, err := parseProxy(proxy)
			if err != nil {
				panic(fmt.Sprintf("parse proxy error: %s: %v", proxy, err))
			}
			results = append(results, cidr)
		}
		opts.TrustedProxies = results
	}
}

func WithIpHeaders(headers []string) Option {
	return func(opts *options) {
		opts.IpHeaders = headers
	}
}

// Server 服务侧中间件，从 http.request 中获取客户端 IP, 并存入上下文
// 解析顺序：
// 1. 首先尝试从 TrustedHeader 获取
// 2. 如果下游属于 TrustedProxies, 尝试从 IpHeaders 中获取
// 3. 否则使用 RemoteAddr 作为 client IP
func Server(logger log.Logger, opts ...Option) middleware.Middleware {
	options := &options{
		TrustedProxies: make([]*net.IPNet, 0),
		IpHeaders: []string{
			"X-Real-IP",
			"X-Forwarded-For",
		},
	}
	for _, opt := range opts {
		opt(options)
	}

	return func(h middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			if httpreq, ok := http.RequestFromServerContext(ctx); ok {
				if options.TrustedHeader != "" {
					value := httpreq.Header.Get(options.TrustedHeader)
					if net.ParseIP(value) != nil {
						ctx = context.WithValue(ctx, realIpKey{}, value)
						return h(ctx, req)
					}
				}

				remoteIpStr, _, err := net.SplitHostPort(httpreq.RemoteAddr)
				if err != nil {
					log.NewHelper(logger).Warnf("failed to split remote addr host:port: %w", err)
					return nil, ErrInvalidRemoteAddr
				}
				remoteIp := net.ParseIP(remoteIpStr)
				if remoteIp == nil {
					return nil, ErrInvalidRemoteAddr
				}

				if isTrustedProxy(options, remoteIp) {
					for _, headerName := range options.IpHeaders {
						value := validateIpHeader(options, httpreq.Header.Get(headerName))
						if value != "" {
							ctx = context.WithValue(ctx, realIpKey{}, value)
							return h(ctx, req)
						}
					}
				}

				ctx = context.WithValue(ctx, realIpKey{}, remoteIpStr)
				return h(ctx, req)
			}
			return nil, ErrWrongContext
		}
	}
}

func FromContext(ctx context.Context) (val string, ok bool) {
	val, ok = ctx.Value(realIpKey{}).(string)
	return
}
