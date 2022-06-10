package slowerdaddy

import (
	"context"
	"log"

	"golang.org/x/time/rate"
)

type Allocator struct {
	global       Limiter
	scoped       Limiter
	quotaReqs    chan *QuotaRequest
	updateLimits chan int
	limit        int
}

type QuotaRequest struct {
	allowCh chan int
	ConnID  string
	Value   int
}

func NewAllocator(global Limiter, limit int) *Allocator {
	return &Allocator{
		quotaReqs:    make(chan *QuotaRequest),
		scoped:       rate.NewLimiter(rate.Limit(limit), limit),
		global:       global,
		limit:        limit,
		updateLimits: make(chan int, 1),
	}
}

func (a *Allocator) Resolve(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case newLimit := <-a.updateLimits:
			log.Println("updating limit to", newLimit)
			a.scoped.SetLimit(rate.Limit(newLimit))
			a.scoped.SetBurst(newLimit)
			a.limit = newLimit
		default:
			if err := a.resolveQuotas(ctx); err != nil {
				log.Println("resolve failed:", err)
				continue
			}
		}
	}
}

func (a *Allocator) resolveQuotas(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return nil
	case req := <-a.quotaReqs:
		if req.Value > a.limit {
			req.Value = a.limit
		}
		if err := a.global.WaitN(ctx, req.Value); err != nil {
			log.Println("global wait failed:", err)
			return err
		}
		if err := a.scoped.WaitN(ctx, req.Value); err != nil {
			log.Println("scoped wait failed:", err)
			return err
		}
		req.allowCh <- req.Value
		log.Printf("grated %d quota for %s", req.Value, req.ConnID)
	}
	return nil
}
