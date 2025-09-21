package utils

import "runtime"

func callerName(skip int) string {
	pc, _, _, ok := runtime.Caller(skip)
	if !ok {
		return "unknown"
	}
	return runtime.FuncForPC(pc).Name()
}

// CallerName 返回当前函数的直接调用者
func CallerName() string {
	// 1 跳过这个函数自己
	return callerName(1)
}
