package slowerdaddy

import (
	"context"
	"log"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type Allocator struct {
	mu      sync.Mutex
	global  *rate.Limiter
	local   *rate.Limiter
	updates chan UpdateQuotaRequest
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

func (a *Allocator) Alloc(amount int) (int, bool) {
	quota := amount
	if amount > a.limit {
		quota = a.limit
	}
	log.Println("checking possible alloc of", quota, "bytes in global limiter")
	if ok := a.global.AllowN(time.Now(), quota); !ok {
		log.Println("global quota exceeded")
		return 0, false
	}
	log.Println("trying to alloc", quota, "bytes in local limiter")
	if err := a.global.WaitN(context.Background(), quota); err != nil {
		log.Println("global wait failed:", err)
		return 0, false
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	if ok := a.local.AllowN(time.Now(), quota); !ok {
		log.Println("local quota exceeded")
		return 0, false
	}
	if err := a.local.WaitN(context.Background(), quota); err != nil {
		log.Println("local wait failed:", err)
		return 0, false
	}

	log.Printf("grated %d quota for", quota)
	return quota, true
}

type RequestQuotaResponse struct {
	Quota int
	OK    bool
}

func (a *Allocator) TryAlloc(ctx context.Context, amount int) (int, error) {
	quota := amount
	if amount > a.limit {
		quota = a.limit
	}

	var okGlobal, okLocal bool
	for !okGlobal {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		okGlobal = a.tryAllocLocal(ctx, a.global, quota)
	}

	for !okLocal {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		okLocal = a.tryAllocLocal(ctx, a.local, quota)
	}
	return quota, nil
}

func (a *Allocator) SetLimit(limit int) {
	a.mu.Lock()
	a.limit = limit
	a.local.SetLimit(rate.Limit(limit))
	a.local.AllowN(time.Now(), limit)
	a.mu.Unlock()
}

func (a *Allocator) SetLocalLimit(limit int) {
	a.mu.Lock()
	a.limit = limit
	a.local.SetLimit(rate.Limit(limit))
	a.local.SetBurst(limit)
	a.local.AllowN(time.Now(), limit)
	a.mu.Unlock()
	a.updates <- UpdateQuotaRequest(limit)
	log.Println("allocator: new limit:", limit)
}

func (a *Allocator) tryAllocLocal(ctx context.Context, limiter *rate.Limiter, quota int) bool {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	allowedLocal := make(chan bool, 1)
	go func() {
		if err := limiter.WaitN(ctx, quota); err != nil {
			allowedLocal <- false
		}
		allowedLocal <- true
	}()

	for {
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
		}
	}
}
