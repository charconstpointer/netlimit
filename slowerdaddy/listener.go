package slowerdaddy

import (
	"context"
	"errors"
	"log"
	"net"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
)

var (
	// ErrLimitGreaterThanTotal is returned when the limit is greater than the total limit of the listener.
	ErrLimitGreaterThanTotal = errors.New("limit per conn cannot be greater than total limit")
)

var _ net.Listener = (*Listener)(nil)

// Listener is a net.Listener that allows to control the bandwidth of the net.Conn connections and the limiter itself.
type Listener struct {
	mu sync.Mutex
	net.Listener
	limiter     *rate.Limiter
	conns       []*Allocator
	limitConn   int
	limitGlobal int
}

// Listen returns a Listener that will be bound to addr with the specified limits.
// Listen uses WithLimit to create the Listener.
func Listen(network, addr string, limitTotal, limitConn int) (*Listener, error) {
	ln, err := net.Listen(network, addr)
	if err != nil {
		return nil, err
	}

	limiter := rate.NewLimiter(rate.Limit(limitTotal), limitTotal)

	return &Listener{
		Listener:    ln,
		limitConn:   limitConn,
		limitGlobal: limitTotal,
		limiter:     limiter,
	}, nil
}

// Accept waits for and returns the next connection to the listener.
func (l *Listener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	alloc := NewAllocator(l.limiter, l.limitConn)
	newConn := &Conn{
		Conn: conn,
		a:    alloc,
	}

	l.mu.Lock()
	l.conns = append(l.conns, alloc)
	l.mu.Unlock()

	return newConn, nil
}

// SetTotalLimit sets the limit of the bandwidth of all net.Conn connections currently active combined.
func (l *Listener) SetTotalLimit(limit int) error {
	log.Println("allocator global: new limit:", limit)
	l.mu.Lock()
	l.limiter.AllowN(time.Now(), l.limitGlobal)
	l.limiter.SetLimit(rate.Limit(limit))
	l.limiter.SetBurst(limit)
	l.limitGlobal = limit
	l.mu.Unlock()
	return nil
}

func (l *Listener) SetLocalLimit(ctx context.Context, newLocalLimit int) error {
	log.Println("allocator local: new limit:", newLocalLimit)
	if newLocalLimit > l.limitGlobal {
		return ErrLimitGreaterThanTotal
	}
	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(len(l.conns))

	l.mu.Lock()
	defer l.mu.Unlock()
	for _, alloc := range l.conns {
		alloc := alloc
		eg.TryGo(func() error {
			return alloc.SetLimit(ctx, newLocalLimit)
		})
	}
	err := eg.Wait()

	if err != nil {
		log.Println("error setting local limit:", err)
		return err
	}
	l.limitConn = newLocalLimit
	return nil
}
