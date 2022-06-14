package netlimit

import (
	"context"
	"errors"
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

// Listener is a net.Listener that allows to control the bandwidth of the net.Conn connections it accepts.
type Listener struct {
	mu sync.Mutex
	net.Listener

	// limiter is the global limiter that is the upper bound of all net.Conn connections combined
	// all connections combined cannot exceed limits enforced by this limiter.
	limiter *rate.Limiter

	// conns is a list of currently "active" Conn connections.
	// conns are updated just after accepting a new Conn connection.
	// There is a gc like goroutine started just after Listener is init'd
	// to cleanup dangling Conn connections, the default time.Duration interval for
	// this process is time.Second, of course we could make it configurable.
	conns []*Conn

	// localLimit determines maximum bytes per second limit of bandwidth allowed per single active Conn connection
	// localLimit cannot be greater than globalLimit
	localLimit int

	// globalLimit determines maximum bytes per second limit of bandwidth allowed for all active Conn connections combined
	// globalLimit cannot be lower than localLimit
	globalLimit int

	// gcInterval is the interval between "gc" cycles
	gcInterval time.Duration
}

// Listen returns a *Listener that will be bound to addr with the specified limits.
// Listen starts gc like process in separate goroutine that attempts to clean up
// dangling Conn connections
// net.ListenConfig.Listen is
func Listen(network, addr string, limitTotal, limitConn int) (*Listener, error) {
	return ListenCtx(context.Background(), network, addr, limitTotal, limitConn)
}

// ListenCtx does the same as Listen but also takes a context.Context.
// The context argument is not used for anything after Listen returns.
// It's there to permit an early return for a DNS lookup,
// and because functions like internetSocket take a context argument
// even though it won't be used for the particular case of Listen
func ListenCtx(ctx context.Context, network, addr string, limitTotal, limitConn int) (*Listener, error) {
	if limitTotal < limitConn {
		return nil, ErrLimitGreaterThanTotal
	}

	cfg := net.ListenConfig{}
	ln, err := cfg.Listen(ctx, network, addr)
	if err != nil {
		return nil, err
	}
	limiter := rate.NewLimiter(rate.Limit(limitTotal), limitTotal)

	limitedLn := &Listener{
		Listener:    ln,
		localLimit:  limitConn,
		globalLimit: limitTotal,
		limiter:     limiter,
		gcInterval:  time.Second,
	}

	go limitedLn.gc()
	return limitedLn, nil
}

// Accept waits for and returns the next connection to the listener.
func (l *Listener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	alloc := NewDefaultAllocator(l.limiter, l.localLimit)
	newConn := NewConn(conn, alloc)

	l.mu.Lock()
	l.conns = append(l.conns, newConn)
	l.mu.Unlock()

	return newConn, nil
}

// SetGlobalLimit sets the limit of the bandwidth of all net.Conn connections currently active combined.
func (l *Listener) SetGlobalLimit(limit int) error {
	l.mu.Lock()
	l.limiter.SetLimit(rate.Limit(limit))
	l.limiter.SetBurst(limit)
	l.globalLimit = limit
	l.mu.Unlock()
	return nil
}

// SetLocalLimit sets the limit of the bandwidth of all net.Conn active and future connections accepted by the listener.
func (l *Listener) SetLocalLimit(newLocalLimit int) error {
	if newLocalLimit > l.globalLimit {
		return ErrLimitGreaterThanTotal
	}
	eg := errgroup.Group{}
	eg.SetLimit(len(l.conns))

	l.mu.Lock()
	defer l.mu.Unlock()
	for _, alloc := range l.conns {
		alloc := alloc
		eg.TryGo(func() error {
			return alloc.SetLimit(newLocalLimit)
		})
	}

	err := eg.Wait()
	if err != nil {
		return err
	}

	l.localLimit = newLocalLimit
	return nil
}

func (l *Listener) gc() {
	for {
		l.mu.Lock()
		for _, conn := range l.conns {
			conn := conn
			select {
			case <-conn.done:
				l.conns = remove(l.conns, conn)
			default:
			}
		}
		l.mu.Unlock()
		time.Sleep(l.gcInterval)
	}
}

func remove(slice []*Conn, elem *Conn) []*Conn {
	for i, v := range slice {
		if v == elem {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

// TODO: implement Close()
