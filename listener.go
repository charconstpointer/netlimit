package netlimit

import (
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

// Listener is a net.Listener that allows to control the bandwidth of the net.Conn connections and the limiter itself.
type Listener struct {
	mu sync.Mutex
	net.Listener

	// limiter is the global limiter that is the upper bound of all net.Conn connections combined
	// all connections combined cannot exceed limits enforced by this limiter.
	limiter *rate.Limiter

	// conns is the list of currently "active" Conn connections.
	// conns are created when a new connection is accepted
	conns []*Conn

	// localLimit is the limit of the bandwidth allowed for a single net.Conn connection per second
	localLimit int

	// globalLimit is the limit of the bandwidth allowed for all net.Conn connections combined per second
	// cannot be lower than localLimit
	globalLimit int

	// gcInterval
	gcInterval time.Duration
}

// Listen returns a Listener that will be bound to addr with the specified limits.
// Listen uses WithLimit to create the Listener.
func Listen(network, addr string, limitTotal, limitConn int) (*Listener, error) {
	ln, err := net.Listen(network, addr)
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
