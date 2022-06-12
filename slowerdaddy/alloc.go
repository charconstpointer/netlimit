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
	ErrAllocatorClosed       = errors.New("allocator closed")
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

// NewAllocator creates a new allocator with the given global and local limits.
// Allocator controls requested bandwith allocations and ensures that they not exceed requested limits.
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
func (a *Allocator) Alloc(ctx context.Context, requestedQuota int) (int, error) {
	grantedQuota, reservation := a.reserveGlobal(ctx, requestedQuota)
	if !reservation.OK() {
		return 0, ErrCouldNotReserveGlobal
	}

	availableAt := time.NewTimer(reservation.DelayFrom(time.Now()))
	err := a.allocLocal(ctx, grantedQuota)
	if err != nil {
		return 0, err
	}

	<-availableAt.C
	return grantedQuota, nil
}

func (a *Allocator) reserveGlobal(ctx context.Context, quota int) (int, *rate.Reservation) {
	if quota > int(a.local.Limit()) {
		quota = int(a.local.Limit())
	}

	return quota, a.global.ReserveN(time.Now(), quota)
}

func (a *Allocator) allocLocal(ctx context.Context, quota int) error {
	err := a.tryAllocLocal(ctx, a.local, quota)
	if err != nil {
		if errors.Is(err, ErrLimitChangedInflight) {
			err = a.tryAllocLocal(ctx, a.local, quota)
		}
		if errors.Is(err, ErrAllocatorClosed) {
			return err
		}
		if errors.Is(err, context.Canceled) {
			return err
		}
	}
	return nil
}

func (a *Allocator) tryAllocLocal(ctx context.Context, limiter *rate.Limiter, quota int) error {
	type allocResult struct {
		err     error
		success bool
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	allowedLocal := make(chan allocResult, 1)

	go func() {
		if err := limiter.WaitN(ctx, quota); err != nil {
			allowedLocal <- allocResult{err, false}
		}
		allowedLocal <- allocResult{nil, true}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case allowed := <-allowedLocal:
		if !allowed.success {
			return allowed.err
		}
		return nil
	case <-a.updates:
		cancel()
		return ErrLimitChangedInflight
	case <-a.done:
		cancel()
		return ErrAllocatorClosed
	}
}

// SetLimit sets the limit of the local limiter.
// setting new limit will attempt to cancel inflight allocations.
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
