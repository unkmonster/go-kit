package signature

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// MessageToSortedQueryStr
// 将 pb Message 转换为排序过的未编码的 url query  字符串
//
// eg. a=b&c=d
func MessageToSortedQueryStr(msg proto.Message, opts *protojson.MarshalOptions, excludeKeys ...string) (string, error) {
	data, err := opts.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("marshal to json: %w", err)
	}

	var buffer map[string]any
	if err := json.Unmarshal(data, &buffer); err != nil {
		return "", fmt.Errorf("unmarshal to map: %w", err)
	}

	keys := []string{}
	for k := range buffer {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// build excludeMap
	excludeMap := map[string]struct{}{}
	for _, key := range excludeKeys {
		excludeMap[key] = struct{}{}
	}

	parts := []string{}
	for _, key := range keys {
		if _, ok := excludeMap[key]; ok {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%v", key, buffer[key]))
	}
	text := strings.Join(parts, "&")

	return text, nil
}

func MapToSortedQueryStr(m map[string]any, excludeKeys ...string) string {
	keys := []string{}
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// build excludeMap
	excludeMap := map[string]struct{}{}
	for _, key := range excludeKeys {
		excludeMap[key] = struct{}{}
	}

	parts := []string{}
	for _, key := range keys {
		if _, ok := excludeMap[key]; ok {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%v", key, m[key]))
	}
	text := strings.Join(parts, "&")

	return text
}
