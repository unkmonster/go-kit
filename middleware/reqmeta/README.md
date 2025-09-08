# reqmeta

是一个服务测中间件，功能是从 proto message 中提取预定义的元数据，详见单元测试和 example.proto

- x-resource-type: str 资源类型
- x-resource-id-field: 持有资源 ID 的字段名称
- x-action: 对资源的操作，如果未指定，使用 HTTP ACTION
- x-self-hold: 指示当前资源是否属于自己，比如 GET /users/me
- x-resource-collection: 目标是否为资源集合
