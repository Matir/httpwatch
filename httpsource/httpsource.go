// HTTPSource returns HTTP connections from a PacketSource
//
// We also have helpers for working with pcaps
// TODO: Make more generic interfaces for testing.

package httpsource

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/tcpassembly"
	"github.com/google/gopacket/tcpassembly/tcpreader"
	"log"
	"os"
	"sync"
	"time"
)

type connKey [2]gopacket.Flow

// Implements tcpassembly.StreamFactory
type HTTPSource struct {
	Connections chan *HTTPConnection
	Pairs       chan *RequestResponsePair
	pending     map[connKey]*HTTPConnection
	pool        *tcpassembly.StreamPool
	readers     int
	mu          sync.Mutex
	finished    chan bool
}

var logger = log.New(os.Stderr, "httpsource: ", log.Lshortfile|log.Ltime)

// Create a new empty source
func NewHTTPSource() *HTTPSource {
	src := &HTTPSource{}
	src.pending = make(map[connKey]*HTTPConnection)
	src.pool = tcpassembly.NewStreamPool(src)
	src.Connections = make(chan *HTTPConnection, 100)
	src.finished = make(chan bool)
	return src
}

// Create a new stream for a given flow
func (src *HTTPSource) New(netFlow, tcpFlow gopacket.Flow) tcpassembly.Stream {
	stream := tcpreader.NewReaderStream()
	// Add to mappings
	key := connKey{netFlow, tcpFlow}
	logger.Printf("Using key: %v\n", key)
	conn, ok := src.pending[key]
	if !ok {
		// Try other direction
		key = key.swap()
		conn, ok = src.pending[key]
	}
	if !ok {
		conn = NewHTTPConnection(key, src.connectionFinished)
		src.pending[key] = conn
	}
	conn.AddStream(&stream)
	return &stream
}

// Callback for each connection
func (src *HTTPSource) connectionFinished(conn *HTTPConnection) {
	src.mu.Lock()
	delete(src.pending, conn.key)
	src.mu.Unlock()
	if conn.Success() {
		logger.Printf("Success.\n")
		src.Connections <- conn
	}
	select {
	case src.finished <- true:
		return
	default:
		return
	}
}

// Provide Request/Response pairs instead of full connections
func (src *HTTPSource) ConvertConnectionsToPairs() {
	if src.Pairs != nil {
		panic("ConvertConnectionsToPairs called multiple times!")
	}
	src.Pairs = make(chan *RequestResponsePair, cap(src.Connections))
	go func() {
		for conn := range src.Connections {
			for _, pair := range conn.Pairs {
				src.Pairs <- pair
			}
		}
		close(src.Pairs)
	}()
}

// Add a new packet source
func (src *HTTPSource) AddSource(pktsrc *gopacket.PacketSource) {
	assembler := tcpassembly.NewAssembler(src.pool)
	// Increment the counter
	src.mu.Lock()
	src.readers++
	src.mu.Unlock()
	// Run the actual assembly in a goroutine
	go src.readPacketsFromSource(pktsrc, assembler)
}

// Helper for pcap files
func (src *HTTPSource) AddPCAPFile(fname string) error {
	if handle, err := pcap.OpenOffline(fname); err != nil {
		return err
	} else {
		return src.addPCAPSource(handle)
	}
}

// Helper for live cap.
// Assumes a lot of things.  If you want more control, build your own source
// and call AddSource
func (src *HTTPSource) AddPCAPIface(iface string) error {
	if handle, err := pcap.OpenLive(iface, 0xffff, false, 100*time.Millisecond); err != nil {
		return err
	} else {
		return src.addPCAPSource(handle)
	}
}

// Common pcap code
func (src *HTTPSource) addPCAPSource(handle *pcap.Handle) error {
	if err := handle.SetBPFFilter("tcp and port 80"); err != nil {
		handle.Close()
		return err
	}
	src.AddSource(gopacket.NewPacketSource(handle, handle.LinkType()))
	return nil
}

// Used as a goroutine to continually read packets and assemble them
// Currently enforcing a 1:1 assembler/source relationship
func (src *HTTPSource) readPacketsFromSource(pktsrc *gopacket.PacketSource,
	assembler *tcpassembly.Assembler) {
	defer func() {
		assembler.FlushAll()
		src.readerFinished()
		logger.Println("Packet source finished.")
	}()
	for packet := range pktsrc.Packets() {
		netFlow := packet.NetworkLayer().NetworkFlow()
		if tcp := packet.Layer(layers.LayerTypeTCP); tcp != nil {
			tcp, _ := tcp.(*layers.TCP)
			// If we have capture metadata, use it to provide the timestamp
			if metadata := packet.Metadata(); metadata != nil {
				assembler.AssembleWithTimestamp(netFlow, tcp, metadata.Timestamp)
			} else {
				assembler.Assemble(netFlow, tcp)
			}
		}
	}
}

// A reader has finished
func (src *HTTPSource) readerFinished() {
	src.mu.Lock()
	defer src.mu.Unlock()
	src.readers--
	if src.readers == 0 {
		src.finished <- true
	}
}

// Check if all readers have finished
func (src *HTTPSource) Finished() bool {
	src.mu.Lock()
	defer src.mu.Unlock()
	if src.readers == 0 && len(src.pending) == 0 {
		close(src.Connections)
		return true
	}
	return false
}

// Wait until all readers are finished and their streams
// have been processed.  Note that this may block if
// enough connections exist to fill src.Connections.
func (src *HTTPSource) WaitUntilFinished() {
	for {
		if func() bool { // Block to scope the defer
			<-src.finished
			src.mu.Lock()
			defer src.mu.Unlock()
			if src.readers == 0 && len(src.pending) == 0 {
				close(src.Connections)
				return true
			}
			return false
		}() {
			break
		}
	}
}

// Swap a connkey
func (key connKey) swap() connKey {
	net, tcp := key[0], key[1]
	return connKey{net.Reverse(), tcp.Reverse()}
}
