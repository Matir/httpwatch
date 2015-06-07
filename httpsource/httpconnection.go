// HTTPSource returns HTTP connections from a PacketSource
//
// We also have helpers for working with pcaps

package httpsource

import (
	"bufio"
	"bytes"
	"github.com/google/gopacket/tcpassembly/tcpreader"
	"io"
	"io/ioutil"
	"net/http"
)

// Matches pairs of requests & responses
type RequestResponsePair struct {
	Request  *http.Request
	Response *http.Response
}

// HTTPConnection represents the HTTP transactions within a single
// TCP session.  It may contain 1 or more RequestResponsePairs.
// Multiple pairs will be included in a keep-alive connection.
type HTTPConnection struct {
	Pairs    []*RequestResponsePair
	key      connKey
	a, b     *tcpreader.ReaderStream
	Finished func(*HTTPConnection)
	err      error
}

// Emulate a ReaderCloser
type bodyBuffer struct {
	*bytes.Reader
}

// Add a ReaderStream to this connection.
func (conn *HTTPConnection) AddStream(s *tcpreader.ReaderStream) {
	if conn.a == nil {
		conn.a = s
		return
	}
	if conn.b == nil {
		conn.b = s
		go conn.startReadConnection()
		return
	}
	panic("More than 2 Streams for one connection!")
}

// Read the connection data into Request/Response Pairs
func (conn *HTTPConnection) startReadConnection() {
	request, response, err := conn.sortStreams()
	if err != nil {
		logger.Printf("Error getting request/response: %v\n", err)
	} else {
		conn.readConnection(request, response)
	}
	conn.execCallback()
}

// Implementation of reading connection, should be more testable
func (conn *HTTPConnection) readConnection(request, response *bufio.Reader) {
	eof := false

	handleErr := func(err error) bool {
		if err == nil {
			return false
		}
		eof = eof || err == io.EOF
		if err == io.EOF {
			return false
		}
		logger.Printf("Error reading from HTTP Connection: %v\n", err)
		conn.err = err
		return true
	}

	for {
		req, err := http.ReadRequest(request)
		if handleErr(err) {
			return
		}
		// Replace the body
		buf, err := ioutil.ReadAll(req.Body)
		req.Body.Close()
		req.Body = &bodyBuffer{bytes.NewReader(buf)}
		if handleErr(err) {
			return
		}
		err = consumeWhitespace(request)
		handleErr(err)

		// Try to read a matching response
		resp, err := http.ReadResponse(response, req)
		if handleErr(err) {
			return
		}

		// Replace the body
		// TODO: figure out a lower memory version of this
		buf, err = ioutil.ReadAll(resp.Body)
		if handleErr(err) {
			return
		}
		resp.Body.Close()
		resp.Body = &bodyBuffer{bytes.NewReader(buf)}

		pair := &RequestResponsePair{Request: req, Response: resp}
		conn.Pairs = append(conn.Pairs, pair)

		err = consumeWhitespace(response)
		handleErr(err)

		if eof {
			break
		}
	}
}

func (conn *HTTPConnection) Success() bool {
	return len(conn.Pairs) > 0
}

// Who is the request & response?
func (conn *HTTPConnection) sortStreams() (*bufio.Reader, *bufio.Reader, error) {
	a, b := bufio.NewReader(conn.a), bufio.NewReader(conn.b)
	peek, err := a.Peek(5)
	if err != nil {
		return nil, nil, err
	}
	if string(peek) == "HTTP/" {
		// a is a response
		return b, a, nil
	}
	return a, b, nil
}

// Execute the finished callback
func (conn *HTTPConnection) execCallback() {
	conn.a = nil
	conn.b = nil
	conn.Finished(conn)
}

// Consume leftover whitespace
func consumeWhitespace(r *bufio.Reader) error {
	for {
		c, err := r.ReadByte()
		if err != nil {
			return err
		}
		if c != '\r' && c != '\n' {
			r.UnreadByte()
			return nil
		}
	}
}

func (b *bodyBuffer) Close() error { return nil }
