package slowerdaddy

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

var (
	ErrLimitChangedInflight = errors.New("limit changed while inflight")
)

type Allocator struct {
	mu      sync.Mutex
	global  *rate.Limiter
	local   *rate.Limiter
	updates chan UpdateQuotaRequest
	done    chan struct{}
	limit   int
}

type UpdateQuotaRequest int64

type QuotaRequest struct {
	allowCh chan int
	ConnID  string
	Value   int
}

func NewAllocator(global *rate.Limiter, limit int) *Allocator {
	return &Allocator{
		local:   rate.NewLimiter(rate.Limit(limit), limit),
		global:  global,
		limit:   limit,
		updates: make(chan UpdateQuotaRequest, 1),
	}
}

func (a *Allocator) TryAlloc(ctx context.Context, amount int) (int, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	quota := amount
	if amount > a.limit {
		quota = a.limit
	}

	var okGlobal, okLocal bool
	for !okGlobal {
		if amount > a.limit {
			quota = a.limit
		} else {
			quota = amount
		}
		okGlobal = a.tryAllocLocal(ctx, a.global, quota)
	}

	for !okLocal {
		if amount > a.limit {
			quota = a.limit
		} else {
			quota = amount
		}
		okLocal = a.tryAllocLocal(ctx, a.local, quota)
	}

	return quota, nil
}

func (a *Allocator) tryAllocLocal(ctx context.Context, limiter *rate.Limiter, quota int) bool {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	allowedLocal := make(chan bool, 1)

	go func() {
		if err := limiter.WaitN(ctx, quota); err != nil {
			if errors.Is(err, context.Canceled) {
				log.Println("allocator: context canceled")
			}
			allowedLocal <- false
		}
		allowedLocal <- true
	}()

	select {
	case <-ctx.Done():
		return false
	case allowed := <-allowedLocal:
		if !allowed {
			return false
		}
		return true
	case newLimit := <-a.updates:
		cancel()
		log.Println("new limit:", newLimit)
		return false
	case <-a.done:
		cancel()
		return false
	}
}

func (a *Allocator) SetLimit(limit int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.limit = limit
	a.local.SetLimit(rate.Limit(limit))
	a.local.SetBurst(limit)
	a.local.AllowN(time.Now(), limit)
	select {
	//there is a leftover update from the previous limit, we can ignore it
	case <-a.updates:
		a.updates <- UpdateQuotaRequest(limit)
	default:
		a.updates <- UpdateQuotaRequest(limit)
	}

	log.Println("allocator: new limit:", a.limit)
}

func (a *Allocator) Close() {
	a.done <- struct{}{}
}
