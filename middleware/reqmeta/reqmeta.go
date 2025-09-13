package reqmeta

import (
	"context"
	"strings"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
	openapi_v3 "github.com/google/gnostic-models/openapiv3"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	// 请求资源的类型
	resourceTypeKey = "x-resource-type"
	// 持有资源 ID 的字段名称
	resourceIdFieldKey = "x-resource-id-field"
	// 操作, 如果未指定使用 HTTP 标注方法填充
	actionKey = "x-resource-action"
	// 当前资源是否属于请求者
	selfHoldKey = "x-resource-self-hold"
	// 是否为资源集合
	resourceCollectionKey = "x-resource-collection"
	// 持有 owner ID 的字段名称
	ownerIdFieldKey = "x-resource-owner-id-field"
	// owner 类型：user/admin/...
	ownerTypeKey = "x-resource-owner-type"
)

type Resource struct {
	ResourceId   any
	ResourceType string
	Action       string
	IsCollection bool
	IsSelfHold   bool
	OwnerId      any
	OwnerType    string
}

type options struct {
	attributes map[string]string
}

type Option func(o *options)

// WithAttributes UNUSED
func WithAttributes(attributes map[string]string) Option {
	return func(o *options) {
		o.attributes = attributes
	}
}

// 从 proto message 提取请求元数据并注入到上下文
func Server(opts ...Option) middleware.Middleware {
	options := &options{
		attributes: nil,
	}
	for _, opt := range opts {
		opt(options)
	}

	return func(h middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			msg, ok := req.(proto.Message)
			if !ok {
				return h(ctx, req)
			}

			meta, err := parseMessage(msg)
			if err != nil {
				return nil, err
			}

			// 使用标准方法作为 action 的默认值
			if meta.Action == "" {
				if tr, ok := transport.FromServerContext(ctx); ok {
					if htr, ok := tr.(http.Transporter); ok {
						meta.Action = htr.Request().Method
					}
				}
			}
			ctx = NewContext(ctx, meta)
			return h(ctx, req)
		}
	}
}

func parseMessage(msg proto.Message) (Resource, error) {
	desc := msg.ProtoReflect().Descriptor()
	schema := proto.GetExtension(desc.Options(), openapi_v3.E_Schema).(*openapi_v3.Schema)

	var (
		resourceIdField string
		ownerIdField    string
	)

	result := Resource{}
	for _, ext := range schema.GetSpecificationExtension() {
		value := ext.Value.Yaml
		switch ext.Name {
		case resourceTypeKey:
			result.ResourceType = value
		case resourceIdFieldKey:
			resourceIdField = value
		case actionKey:
			result.Action = value
		case selfHoldKey:
			result.IsSelfHold = strings.EqualFold("true", value)
		case resourceCollectionKey:
			result.IsCollection = strings.EqualFold("true", value)
		case ownerIdFieldKey:
			ownerIdField = value
		case ownerTypeKey:
			result.OwnerType = value
		}
	}

	msgReflect := msg.ProtoReflect()
	field := msgReflect.Descriptor().Fields().ByName(protoreflect.Name(resourceIdField))
	if field != nil {
		result.ResourceId = msgReflect.Get(field).Interface()
	}
	field = msgReflect.Descriptor().Fields().ByName(protoreflect.Name(ownerIdField))
	if field != nil {
		result.OwnerId = msgReflect.Get(field).Interface()
	}
	return result, nil
}

type resKey struct{}

func NewContext(ctx context.Context, meta Resource) context.Context {
	return context.WithValue(ctx, resKey{}, meta)
}

func FromContext(ctx context.Context) (meta Resource, ok bool) {
	meta, ok = ctx.Value(resKey{}).(Resource)
	return
}
