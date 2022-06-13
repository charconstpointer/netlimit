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

// Allocator is responsible for controlling requested allocations and ensuring that they not exceed requested limits.
// Allocator controls single connection
type Allocator struct {
	mu sync.Mutex
	// global is the global limiter responsible for maintaining the global bandwidth in the requested range
	global *rate.Limiter

	// local is the local limiter responsible for maintaining the local bandwidth in the requested range
	local *rate.Limiter

	// limitUpdates is a channel used to signal that the local limit has changed
	limitUpdates chan struct{}
}

// NewAllocator creates a new allocator with the given global and local limits.
// Allocator controls requested bandwidth allocations and ensures that they not exceed requested limits.
func NewAllocator(global *rate.Limiter, limit int) *Allocator {
	return &Allocator{
		local:        rate.NewLimiter(rate.Limit(limit), limit),
		global:       global,
		limitUpdates: make(chan struct{}, 1),
	}
}

// Alloc blocks until it is allowed to allocate requested quota.
func (a *Allocator) Alloc(ctx context.Context, requestedQuota int) (int, error) {
	grantedQuota, err := a.TryAlloc(ctx, requestedQuota)
	for err == ErrLimitChangedInflight {
		grantedQuota, err = a.TryAlloc(ctx, requestedQuota)
	}

	return grantedQuota, err
}

// TryAlloc reserves quota in a global limiter and then blocks until it is allowed to allocate the quota in the local limiter.
// Once the local limiter allows allocation, TryAlloc waits for the readiness or the global reservation
func (a *Allocator) TryAlloc(ctx context.Context, quota int) (int, error) {
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
	case <-a.limitUpdates:
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
	case <-a.limitUpdates:
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
	case <-a.limitUpdates:
		// there is a leftover update from the previous limit not consumed by allocator, discard it
		a.limitUpdates <- struct{}{}
	default:
		a.limitUpdates <- struct{}{}
	}

	return nil
}
