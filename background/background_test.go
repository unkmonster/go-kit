package background

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/require"
)

func TestInjectClosed(t *testing.T) {
	bg := New(log.DefaultLogger)
	bg.Add(func(ctx context.Context) {
		closed, ok := ClosedFromContext(ctx)
		require.True(t, ok)
		require.IsType(t, make(chan struct{}), closed)
	})
}

func TestQuitNotice(t *testing.T) {
	bg := New(log.DefaultLogger)

	var counter int64
	workers := 32

	for range workers {
		bg.Add(func(ctx context.Context) {
			closed, ok := ClosedFromContext(ctx)
			require.True(t, ok)

			<-closed
			atomic.AddInt64(&counter, 1)
		})
	}

	ctx := context.Background()
	bg.Start(ctx)

	err := bg.Close(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(workers), counter)
}
