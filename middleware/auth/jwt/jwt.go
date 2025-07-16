package jwt

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/metadata"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/golang-jwt/jwt/v5"
)

const (

	// bearerWord the bearer key word for authorization
	bearerWord string = "Bearer"

	// authorizationKey holds the key used to store the JWT Token in the request tokenHeader.
	authorizationKey string = "Authorization"

	// reason holds the error reason.
	reason string = "UNAUTHORIZED"
)

var (
	ErrMissingJwtToken        = errors.Unauthorized(reason, "JWT token is missing")
	ErrMissingKeyFunc         = errors.Unauthorized(reason, "keyFunc is missing")
	ErrTokenInvalid           = errors.Unauthorized(reason, "Token is invalid")
	ErrTokenExpired           = errors.Unauthorized(reason, "JWT token has expired")
	ErrTokenParseFail         = errors.Unauthorized(reason, "Fail to parse JWT token ")
	ErrUnSupportSigningMethod = errors.Unauthorized(reason, "Wrong signing method")
	ErrWrongContext           = errors.Unauthorized(reason, "Wrong context for middleware")
	ErrNeedTokenProvider      = errors.Unauthorized(reason, "Token provider is missing")
	ErrSignToken              = errors.Unauthorized(reason, "Can not sign token.Is the key correct?")
	ErrGetKey                 = errors.Unauthorized(reason, "Can not get key while signing token")
)

type authKey struct{}

type options struct {
	// claims
	claims func() jwt.Claims
	// 用于验证 Token
	keyFunc jwt.Keyfunc
}

type Option func(*options)

func WithClaims(f func() jwt.Claims) Option {
	return func(o *options) {
		o.claims = f
	}
}

func WithKeyFunc(kf jwt.Keyfunc) Option {
	return func(o *options) {
		o.keyFunc = kf
	}
}

// Server 服务侧中间件
// 如果提供了 KeyFunc, 则强制验证签名
// 1. 解析 token 并保存到上下文。如果未指定 keyFunc, 则不验证 token, 请确保 token 来自可信来源，例如网关
// 2. 通过 metadata 传播 authorization header
// 无论是否验证 token, 缺少 header 总会返回 ErrMissingToken
func Server(opts ...Option) middleware.Middleware {
	claims := jwt.RegisteredClaims{}
	o := &options{
		claims: func() jwt.Claims { return claims },
	}
	for _, opt := range opts {
		opt(o)
	}

	mustVerify := o.keyFunc != nil

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			if tr, ok := transport.FromServerContext(ctx); ok {
				authorizationValue := tr.RequestHeader().Get(authorizationKey)

				if authorizationValue == "" {
					return nil, ErrMissingJwtToken
				}

				// 通过 metadata 传播 authorization header
				ctx = metadata.AppendToClientContext(ctx, authorizationKey, authorizationValue)

				// 判断是否为 Bearer token
				auths := strings.SplitN(authorizationValue, " ", 2)
				if len(auths) != 2 || !strings.EqualFold(auths[0], bearerWord) {
					return nil, ErrMissingJwtToken
				}

				// 解析 token string
				tokenString := auths[1]

				var tokenInfo *jwt.Token
				var err error

				if mustVerify {
					tokenInfo, err = jwt.ParseWithClaims(tokenString, o.claims(), o.keyFunc)
				} else {
					tokenInfo, _, err = jwt.NewParser().ParseUnverified(tokenString, o.claims())
				}
				if err != nil {
					return nil, errors.Unauthorized(reason, fmt.Sprintf("parse token: %s", err))
				}

				if mustVerify && !tokenInfo.Valid {
					return nil, ErrTokenInvalid
				}

				// 存入上下文
				ctx = NewContext(ctx, tokenInfo.Claims)
				return handler(ctx, req)
			}
			return nil, ErrWrongContext
		}
	}
}

// NewContext put auth info into context
func NewContext(ctx context.Context, info jwt.Claims) context.Context {
	return context.WithValue(ctx, authKey{}, info)
}

// FromContext extract auth info from context
func FromContext(ctx context.Context) (token jwt.Claims, ok bool) {
	token, ok = ctx.Value(authKey{}).(jwt.Claims)
	return
}
