package output

import (
	"github.com/Matir/httpwatch/httpsource"
	"sync"
	"time"
)

type output struct {
	name string
	dst  chan<- *httpsource.RequestResponsePair
}

// OutputMux reads from a single channel and distributes it to
// many child channels in parallel.
type OutputMux struct {
	outputs  []output
	lock     sync.Mutex
	src      <-chan *httpsource.RequestResponsePair
	blocking bool
	timeout  time.Duration
	writer   func(output, *httpsource.RequestResponsePair)
}

// NewBlockingOutputMux creates a new OutputMux that blocks on writes to full
// channels.
func NewBlockingOutputMux(src <-chan *httpsource.RequestResponsePair) OutputMux {
	m := OutputMux{src: src, blocking: true, writer: blockingOutputWriter}
	return m
}

// NewNonBlockingOutputMux creates new OutputMux that doesn't block on writes.
func NewNonBlockingOutputMux(src <-chan *httpsource.RequestResponsePair, timeout time.Duration) OutputMux {
	m := OutputMux{src: src, blocking: false, timeout: timeout}
	if timeout != 0 {
		m.writer = makeTimeoutOutputWriter(timeout)
	} else {
		m.writer = nonBlockingOutputWriter
	}
	return m
}

// AddOutput adds an output with name 'name' and channel buffer size 'buf'
func (m *OutputMux) AddOutput(name string, buf int) <-chan *httpsource.RequestResponsePair {
	c := make(chan *httpsource.RequestResponsePair, buf)
	m.lock.Lock()
	defer m.lock.Unlock()
	m.outputs = append(m.outputs, output{name, c})
	return c
}

// blockingOutputWriter writes out to a channel
func blockingOutputWriter(o output, item *httpsource.RequestResponsePair) {
	o.dst <- item
}

// timeoutOutputWriter writes out to a channel with a timeout in ms
func makeTimeoutOutputWriter(timeout time.Duration) func(output, *httpsource.RequestResponsePair) {
	return func(o output, item *httpsource.RequestResponsePair) {
		kill := make(chan bool)
		go func() {
			time.Sleep(timeout)
			kill <- true
		}()
		select {
		case o.dst <- item:
			// Working as intended
		case <-kill:
			// TODO: log timeout on channel
		}
	}
}

// nonBlockingOutputWriter doesn't block at all
func nonBlockingOutputWriter(o output, item *httpsource.RequestResponsePair) {
	select {
	case o.dst <- item:
		// Working as planned
	default:
		// TODO: log failure to write to channel
	}
}
