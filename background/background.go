package background

import (
	"context"
	"runtime/debug"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
)

var _ transport.Server = (*Background)(nil)

// 非线程安全的后台任务管理器
// 多次调用 Launch/Close 行为是未定义的
type BackgroundFunc func(ctx context.Context)

type Background struct {
	tasks []BackgroundFunc
	wg    sync.WaitGroup
	log   *log.Helper
	// 会在退出时 close 这个 channel
	closed chan struct{}
}

func New(logger log.Logger) *Background {
	return &Background{
		log:    log.NewHelper(log.With(logger, "module", "background")),
		closed: make(chan struct{}),
	}
}

// Start implements transport.Server.
func (bg *Background) Start(ctx context.Context) error {
	bg.Launch(ctx)
	return nil
}

// Stop implements transport.Server.
func (bg *Background) Stop(ctx context.Context) error {
	bg.log.Warnf("[Background] stopping")
	return bg.Close(ctx)
}

func (bg *Background) Add(f BackgroundFunc) {
	bg.tasks = append(bg.tasks, f)
}

// nopanic 捕获来自 f 的 panic 并返回它和堆栈信息
func nopanic(ctx context.Context, f BackgroundFunc) (p any, stk string) {
	defer func() {
		p = recover()
		if p != nil {
			stk = string(debug.Stack())
		}

	}()
	f(ctx)
	return
}

// forever 永远运行，直到 ctx 被取消，
// 如果 f 发生了 panic 会打印堆栈信息到 logger
func (bg *Background) forever(ctx context.Context, f BackgroundFunc) error {
	for ctx.Err() == nil {
		p, stk := nopanic(ctx, f)

		if p != nil {
			bg.log.Warnw("recovered panic %v\n%s", p, stk)
			continue
		}

		return nil
	}
	return ctx.Err()
}

// Launch 非阻塞的启动所有已添加的后台任务
func (bg *Background) Launch(ctx context.Context) {
	for _, task := range bg.tasks {
		bg.wg.Add(1)
		go func(ctx context.Context, task BackgroundFunc) {
			defer bg.wg.Done()
			ctx = NewContextWithClosed(ctx, bg.closed)
			bg.forever(ctx, task)
		}(ctx, task)
	}
}

// Close 等待所有后台任务退出并返回
func (bg *Background) Close(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	// 通知所有任务退出
	close(bg.closed)

	done := make(chan struct{})
	go func() {
		bg.wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

func (bg *Background) SetLogger(logger log.Logger) {
	bg.log = log.NewHelper(log.With(logger, "module", "background"))
}

var Default *Background = New(log.DefaultLogger)

type closedKey struct{}

func NewContextWithClosed(ctx context.Context, closed chan struct{}) context.Context {
	return context.WithValue(ctx, closedKey{}, closed)
}

func ClosedFromContext(ctx context.Context) (closed chan struct{}, ok bool) {
	closed, ok = ctx.Value(closedKey{}).(chan struct{})
	return
}
