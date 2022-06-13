package slowerdaddy

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

var (
	ErrLimitChangedInflight  = errors.New("limit changed while inflight")
	ErrCouldNotReserveGlobal = errors.New("could not reserve quota in a global limiter")
)

type Allocator struct {
	mu             sync.Mutex
	global         *rate.Limiter
	local          *rate.Limiter
	localUpdatesCh chan UpdateQuotaRequest
}

type UpdateQuotaRequest int

// NewAllocator creates a new allocator with the given global and local limits.
// Allocator controls requested bandwidth allocations and ensures that they not exceed requested limits.
func NewAllocator(global *rate.Limiter, limit int) *Allocator {
	return &Allocator{
		local:          rate.NewLimiter(rate.Limit(limit), limit),
		global:         global,
		localUpdatesCh: make(chan UpdateQuotaRequest, 1),
	}
}

// Alloc reserves quota in a global limiter and then blocks until it is allowed to allocate in the local limiter.
// Once the local limiter allows allocation, Alloc waits for the readiness or the global reservation
func (a *Allocator) Alloc(ctx context.Context, requestedQuota int) (int, error) {
	grantedQuota, err := a.tryAlloc(ctx, requestedQuota)
	for err == ErrLimitChangedInflight {
		grantedQuota, err = a.tryAlloc(ctx, requestedQuota)
	}

	return grantedQuota, err
}

func (a *Allocator) tryAlloc(ctx context.Context, quota int) (int, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	grantedQuota, reservation := a.reserveGlobal(quota)
	if !reservation.OK() {
		return 0, ErrCouldNotReserveGlobal
	}

	availableAt := time.NewTimer(reservation.DelayFrom(time.Now()))
	err := a.tryAllocLocal(ctx, grantedQuota)
	if err != nil {
		reservation.Cancel()
		return 0, err
	}

	select {
	case <-a.localUpdatesCh:
		reservation.Cancel()
		return 0, ErrLimitChangedInflight
	default:
	}

	<-availableAt.C
	return grantedQuota, nil
}

func (a *Allocator) reserveGlobal(quota int) (int, *rate.Reservation) {
	if quota > int(a.local.Limit()) {
		quota = int(a.local.Limit())
	}

	return quota, a.global.ReserveN(time.Now(), quota)
}

func (a *Allocator) tryAllocLocal(ctx context.Context, quota int) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	allowedLocal := make(chan bool, 1)

	go func() {
		if err := a.local.WaitN(ctx, quota); err != nil {
			allowedLocal <- false
		}
		allowedLocal <- true
	}()

	select {
	case allowed := <-allowedLocal:
		if !allowed {
			return fmt.Errorf("could not allocate quota in local limiter")
		}
		return nil
	case <-a.localUpdatesCh:
		return ErrLimitChangedInflight
	case <-ctx.Done():
		return ctx.Err()
	}
}

// SetLimit sets the limit of the local limiter.
// setting new limit will attempt to cancel inflight allocations.
func (a *Allocator) SetLimit(limit int) error {
	if limit > int(a.global.Limit()) {
		return fmt.Errorf("local limit cannot be higher than global limit")
	}

	a.mu.Lock()
	a.local.SetLimit(rate.Limit(limit))
	a.local.SetBurst(limit)
	a.mu.Unlock()

	select {
	case <-a.localUpdatesCh:
		// there is a leftover update from the previous limit not consumed by allocator, discard it
		a.localUpdatesCh <- UpdateQuotaRequest(limit)
	default:
		a.localUpdatesCh <- UpdateQuotaRequest(limit)
	}

	return nil
}
