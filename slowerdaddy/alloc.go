package slowerdaddy

import (
	"context"
	"log"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type Allocator struct {
	global *rate.Limiter
	local  *rate.Limiter
	limit  int
	mu     sync.Mutex
}

type QuotaRequest struct {
	allowCh chan int
	ConnID  string
	Value   int
}

func NewAllocator(global *rate.Limiter, limit int) *Allocator {
	return &Allocator{
		local:  rate.NewLimiter(rate.Limit(limit), limit),
		global: global,
		limit:  limit,
	}
}

func (a *Allocator) TryAlloc(amount int) (int, bool) {
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

func (a *Allocator) SetLimit(limit int) {
	a.mu.Lock()
	a.limit = limit
	a.local.SetLimit(rate.Limit(limit))
	a.mu.Unlock()
}
