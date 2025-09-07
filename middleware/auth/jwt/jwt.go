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

type Claims = jwt.Claims

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

type KeyFunc func(ctx context.Context, token *jwt.Token) (any, error)

type options struct {
	// claims
	claims func() jwt.Claims
	// 用于验证 Token
	keyFunc  jwt.Keyfunc
	prevent  bool
	keyFunc2 KeyFunc
}

type Option func(*options)

func WithClaims(f func() jwt.Claims) Option {
	return func(o *options) {
		o.claims = f
	}
}

// WithKeyFunc 如果未指定此选项表示不验证 token 签名，
// 请确保 token 来自可信来源，例如网关
func WithKeyFunc(kf jwt.Keyfunc) Option {
	return func(o *options) {
		o.keyFunc = kf
	}
}

func WithKeyFunc2(kf KeyFunc) Option {
	return func(o *options) {
		o.keyFunc2 = kf
	}
}

// WithPreventReq 指示当缺少 Token / 签名验证失败时是否阻止请求
func WithPreventReq(prevent bool) Option {
	return func(o *options) {
		o.prevent = prevent
	}
}

func Server(opts ...Option) middleware.Middleware {
	claims := jwt.RegisteredClaims{}
	o := &options{
		claims:  func() jwt.Claims { return claims },
		prevent: true,
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
					if !o.prevent {
						return handler(ctx, req)
					}
					return nil, ErrMissingJwtToken
				}

				// 通过 metadata 传播 authorization header
				ctx = metadata.AppendToClientContext(ctx, authorizationKey, authorizationValue)

				// 判断是否为 Bearer token
				auths := strings.SplitN(authorizationValue, " ", 2)
				if len(auths) != 2 || !strings.EqualFold(auths[0], bearerWord) {
					if !o.prevent {
						return handler(ctx, req)
					}
					return nil, ErrMissingJwtToken
				}

				// 解析 token string
				tokenString := auths[1]

				var tokenInfo *jwt.Token
				var err error

				if mustVerify {
					var keyFunc jwt.Keyfunc
					if o.keyFunc2 != nil {
						keyFunc = func(t *jwt.Token) (interface{}, error) {
							return o.keyFunc2(ctx, tokenInfo)
						}
					} else if o.keyFunc != nil {
						keyFunc = o.keyFunc
					}
					tokenInfo, err = jwt.ParseWithClaims(tokenString, o.claims(), keyFunc)
				} else {
					tokenInfo, _, err = jwt.NewParser().ParseUnverified(tokenString, o.claims())
				}
				if err != nil {
					if !o.prevent {
						return handler(ctx, req)
					}
					return nil, errors.Unauthorized(reason, fmt.Sprintf("parse token: %s", err))
				}

				if mustVerify && !tokenInfo.Valid {
					if !o.prevent {
						return handler(ctx, req)
					}
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
