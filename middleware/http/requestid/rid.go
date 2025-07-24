package requestid

import (
	"context"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport"
)

func Server() middleware.Middleware {
	return func(h middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			if tr, ok := transport.FromServerContext(ctx); ok {
				if tid, ok := tracing.TraceID()(ctx).(string); ok {
					tr.ReplyHeader().Set("X-Request-ID", tid)
				}
			}
			return h(ctx, req)
		}
	}
}
