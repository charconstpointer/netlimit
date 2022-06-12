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
	ErrLimitChangedInflight  = errors.New("limit changed while inflight")
	ErrCouldNotReserveGlobal = errors.New("could not reserve quota in a global limiter")
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

// Alloc reserves quota in a global limiter and then blocks until it is allowed
// to allocate in the local limiter.
// Once the local limiter allows allocation, Alloc waits for the readiness or the global reservation
func (a *Allocator) Alloc(ctx context.Context, amount int) (int, error) {
	quota, reservation := a.reserveGlobal(ctx, amount)
	if !reservation.OK() {
		return 0, ErrCouldNotReserveGlobal
	}

	availableAt := time.NewTimer(time.Duration(reservation.DelayFrom(time.Now())))

	var okLocal bool
	for !okLocal {
		if amount > a.limit {
			quota = a.limit
		} else {
			quota = amount
		}
		okLocal = a.tryAllocLocal(ctx, a.local, quota)
		select {
		case <-ctx.Done():
			reservation.Cancel()
			return 0, ctx.Err()
		default:
		}
	}

	<-availableAt.C
	return quota, nil
}

func (a *Allocator) reserveGlobal(ctx context.Context, amount int) (int, *rate.Reservation) {
	if amount > int(a.global.Limit()) {
		amount = int(a.global.Limit())
	}

	return amount, a.global.ReserveN(time.Now(), amount)
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

func (a *Allocator) tryAllocLocal2(ctx context.Context, quota int) bool {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	allowedLocal := make(chan bool, 1)

	go func() {
		if err := a.local.WaitN(ctx, quota); err != nil {
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
