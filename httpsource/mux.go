package httpsource

import (
	"sync"
	"time"
)

type output struct {
	name string
	dst  chan<- *RequestResponsePair
}

// PairMux reads from a single channel and distributes it to
// many child channels in parallel.
type PairMux struct {
	outputs  []output
	lock     sync.Mutex
	src      <-chan *RequestResponsePair
	blocking bool
	timeout  time.Duration
	writer   func(output, *RequestResponsePair)
	started  bool
}

// NewBlockingPairMux creates a new PairMux that blocks on writes to full
// channels.
func NewBlockingPairMux(src <-chan *RequestResponsePair) PairMux {
	m := PairMux{src: src, blocking: true, writer: blockingOutputWriter}
	return m
}

// NewNonBlockingPairMux creates new PairMux that doesn't block on writes.
func NewNonBlockingPairMux(src <-chan *RequestResponsePair, timeout time.Duration) PairMux {
	m := PairMux{src: src, blocking: false, timeout: timeout}
	if timeout != 0 {
		m.writer = makeTimeoutOutputWriter(timeout)
	} else {
		m.writer = nonBlockingOutputWriter
	}
	return m
}

// AddOutput adds an output with name 'name' and channel buffer size 'buf'
func (m *PairMux) AddOutput(name string, buf int) <-chan *RequestResponsePair {
	c := make(chan *RequestResponsePair, buf)
	m.lock.Lock()
	defer m.lock.Unlock()
	m.outputs = append(m.outputs, output{name, c})
	return c
}

// Start stats the goroutine that will perform the copying.
func (m *PairMux) Start() {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.started {
		return
	}
	go func() {
		for {
			m.RunStep()
		}
	}()
	m.started = true
}

// RunStep handles a single item through the mux
func (m *PairMux) RunStep() {
	item := <-m.src
	m.lock.Lock()
	outputs := m.outputs
	m.lock.Unlock()

	for _, output := range outputs {
		m.writer(output, item)
	}
}

// blockingOutputWriter writes out to a channel
func blockingOutputWriter(o output, item *RequestResponsePair) {
	o.dst <- item
}

// timeoutOutputWriter writes out to a channel with a timeout in ms
func makeTimeoutOutputWriter(timeout time.Duration) func(output, *RequestResponsePair) {
	return func(o output, item *RequestResponsePair) {
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
func nonBlockingOutputWriter(o output, item *RequestResponsePair) {
	select {
	case o.dst <- item:
		// Working as planned
	default:
		// TODO: log failure to write to channel
	}
}
