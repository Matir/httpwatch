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
	"time"
)

type connKey [2]gopacket.Flow

// Implements tcpassembly.StreamFactory
type HTTPSource struct {
	pending     map[connKey]HTTPConnection
	pool        *tcpassembly.StreamPool
	Connections chan *HTTPConnection
}

var logger = log.New(os.Stderr, "httpsource", log.Lshortfile|log.Ltime)

func NewHTTPSource() *HTTPSource {
	src := &HTTPSource{}
	src.pending = make(map[connKey]HTTPConnection)
	src.pool = tcpassembly.NewStreamPool(src)
	src.Connections = make(chan *HTTPConnection, 100)
	return src
}

// Create a new stream for a given flow
func (src *HTTPSource) New(netFlow, tcpFlow gopacket.Flow) tcpassembly.Stream {
	stream := tcpreader.NewReaderStream()
	// Add to mappings
	key := connKey{netFlow, tcpFlow}
	var conn HTTPConnection
	if conn, ok := src.pending[key]; !ok {
		conn = HTTPConnection{Finished: src.connectionFinished}
		src.pending[key] = conn
	}
	conn.AddStream(&stream)
	return &stream
}

// Callback for each connection
func (src *HTTPSource) connectionFinished(conn *HTTPConnection) {
	delete(src.pending, conn.key)
	if conn.Success() {
		src.Connections <- conn
	}
}

// Add a new packet source
func (src *HTTPSource) AddSource(pktsrc *gopacket.PacketSource) {
	assembler := tcpassembly.NewAssembler(src.pool)
	// Run the actual assembly in a goroutine
	go readPacketsFromSource(pktsrc, assembler)
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
func readPacketsFromSource(pktsrc *gopacket.PacketSource,
	assembler *tcpassembly.Assembler) {
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
	logger.Printf("Packet source %v finished.\n", pktsrc)
}
