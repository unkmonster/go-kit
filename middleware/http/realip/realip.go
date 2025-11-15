package realip

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
)

var (
	ErrInvalidRemoteAddr = errors.BadRequest("INVALID_REMOTE_ADDR", "invalid remote address")
)

type realIpKey struct{}

type options struct {
	// 受信任的代理，仅受信任的代理的 xff 头是可信的
	TrustedProxies []*net.IPNet
	// 直接配置受信任的 header, 优先级高于可信代理
	TrustedHeader string
	// xff headers
	IpHeaders []string
}

type Option func(opts *options)

// parseProxy parse a hostname/ipAddr/cidr to *net.IPNet
func parseProxy(proxy string) ([]*net.IPNet, error) {
	cidrStrList := []string{}

	// 解析 IP/hostname 为 CIDR
	if !strings.Contains(proxy, "/") {
		ips := []net.IP{}
		ip := net.ParseIP(proxy)
		if ip == nil {
			// 尝试处理主机名
			result, err := net.LookupIP(proxy)
			if err != nil || len(result) == 0 {
				return nil, fmt.Errorf("invalid hostname: %v, %s", err, proxy)
			}
			ips = append(ips, result...)
		} else {
			ips = append(ips, ip)
		}

		for _, ip := range ips {
			str := ip.String()
			if ip.To4() != nil {
				str = str + "/32"
			} else {
				str = str + "/128"
			}
			cidrStrList = append(cidrStrList, str)
		}
	} else {
		cidrStrList = append(cidrStrList, proxy)
	}

	result := []*net.IPNet{}
	for _, str := range cidrStrList {
		_, cidr, err := net.ParseCIDR(str)
		if err != nil {
			return nil, err
		}
		result = append(result, cidr)
	}

	return result, nil
}

// isTrustedProxy check is ip a trusted proxy
func isTrustedProxy(options *options, ip net.IP) bool {
	for _, proxy := range options.TrustedProxies {
		if proxy.Contains(ip) {
			return true
		}
	}
	return false
}

// getClientIp 反向顺序验证 ip 头
// 返回 ip header 中从右往左第一个不信任的 IP 作为 client IP
func getClientIp(options *options, value string) string {
	if value == "" {
		return ""
	}

	ips := strings.Split(value, ",")
	for i := len(ips) - 1; i >= 0; i-- {
		ipStr := strings.TrimSpace(ips[i])
		if !isTrustedProxy(options, net.ParseIP(ipStr)) {
			return ipStr
		}
	}
	return ""
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
				panic(fmt.Sprintf("parse proxy %q: %v", proxy, err))
			}
			results = append(results, cidr...)
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
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return nil, fmt.Errorf("missing Transpoter in context")
			}
			htr, ok := tr.(http.Transporter)
			if !ok {
				return nil, fmt.Errorf("'Transpoter' in context must be 'http.Transpoter', not be %T", tr)
			}
			request := htr.Request()

			// 如果制定了 TrustedHeader, 尝试从中获取客户端 IP
			if options.TrustedHeader != "" {
				value := request.Header.Get(options.TrustedHeader)
				if net.ParseIP(value) != nil {
					ctx = context.WithValue(ctx, realIpKey{}, value)
					return h(ctx, req)
				}
			}

			remoteIpStr, _, err := net.SplitHostPort(request.RemoteAddr)
			if err != nil {
				return nil, ErrInvalidRemoteAddr.WithCause(err).WithMetadata(map[string]string{
					"remote_addr": request.RemoteAddr,
				})
			}
			remoteIp := net.ParseIP(remoteIpStr)
			if remoteIp == nil {
				return nil, ErrInvalidRemoteAddr.WithMetadata(map[string]string{
					"remote_addr": request.RemoteAddr,
				})
			}

			// 如果下游属于可信代理，尝试从 xff 头获取 client ip
			if isTrustedProxy(options, remoteIp) {
				for _, headerName := range options.IpHeaders {
					value := getClientIp(options, request.Header.Get(headerName))
					if value != "" {
						ctx = context.WithValue(ctx, realIpKey{}, value)
						return h(ctx, req)
					}
				}
			}

			// 否则使用 remote IP 作为 RealIP
			ctx = context.WithValue(ctx, realIpKey{}, remoteIpStr)
			return h(ctx, req)
		}
	}
}

func FromContext(ctx context.Context) (val string, ok bool) {
	val, ok = ctx.Value(realIpKey{}).(string)
	return
}
