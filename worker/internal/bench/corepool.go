package bench

import "context"

type CorePool struct {
	tokens chan struct{}
}

func NewCorePool(size int) *CorePool {
	if size < 1 {
		size = 1
	}
	return &CorePool{tokens: make(chan struct{}, size)}
}

func (pool *CorePool) Size() int {
	if pool == nil {
		return 0
	}
	return cap(pool.tokens)
}

func (pool *CorePool) Busy() int {
	if pool == nil {
		return 0
	}
	return len(pool.tokens)
}

func (pool *CorePool) run(ctx context.Context, work func()) {
	if pool == nil {
		if ctx.Err() == nil {
			work()
		}
		return
	}
	select {
	case pool.tokens <- struct{}{}:
	case <-ctx.Done():
		return
	}
	defer func() { <-pool.tokens }()
	if ctx.Err() != nil {
		return
	}
	work()
}
