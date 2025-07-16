package log

import "github.com/go-kratos/kratos/v2/log"

var _ log.Logger = (*NullLogger)(nil)

// NullLogger 顾名思义它什么都不做
type NullLogger struct{}

// Log implements log.Logger.
func (n *NullLogger) Log(level log.Level, keyvals ...any) error {
	return nil
}
