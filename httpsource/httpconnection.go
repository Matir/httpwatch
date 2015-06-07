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

// RequestResponsePair is a container for an associated
// http.Request and http.Response, along with a copy of their bodies,
// to allow repeated inspection.
type RequestResponsePair struct {
	Request      *http.Request
	RequestBody  []byte
	Response     *http.Response
	ResponseBody []byte
}

// HTTPConnection represents the HTTP transactions within a single
// TCP session.  It may contain 1 or more RequestResponsePairs.
// Multiple pairs will be included in a keep-alive connection.
type HTTPConnection struct {
	Pairs    []*RequestResponsePair
	key      connKey
	data     [2][]byte
	cdata    int
	fin      chan bool
	Finished func(*HTTPConnection)
	err      error
}

// bodyBuffer implements ReaderCloser by wrapping a bytes.Reader.
type bodyBuffer struct {
	*bytes.Reader
}

// NewHTTPConnection reates an HTTPConnection for a given key with a callback.
func NewHTTPConnection(key connKey, finished func(*HTTPConnection)) *HTTPConnection {
	c := &HTTPConnection{Finished: finished, key: key}
	c.fin = make(chan bool, 2)
	return c
}

// AddStream adds a ReaderStream to the connection conn.
func (conn *HTTPConnection) AddStream(s *tcpreader.ReaderStream) {
	// launch a goroutine to read everything
	choice := conn.cdata
	conn.cdata++
	go func() {
		data, err := ioutil.ReadAll(s)
		conn.data[choice] = data
		if err != nil {
			logger.Printf("Unable to read all from connection: %v\n", err)
			conn.err = err
		}
		conn.fin <- true
	}()
	if conn.cdata == 2 {
		go conn.startReadConnection()
	}
}

// Read the connection data into Request/Response Pairs
func (conn *HTTPConnection) startReadConnection() {
	// Wait for 2 to be finished
	<-conn.fin
	<-conn.fin
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
		reqbuf, err := ioutil.ReadAll(req.Body)
		req.Body.Close()
		req.Body = &bodyBuffer{bytes.NewReader(reqbuf)}
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
		respbuf, err := ioutil.ReadAll(resp.Body)
		if handleErr(err) {
			return
		}
		resp.Body.Close()
		resp.Body = &bodyBuffer{bytes.NewReader(respbuf)}

		pair := &RequestResponsePair{Request: req,
			RequestBody: reqbuf, Response: resp, ResponseBody: respbuf}
		conn.Pairs = append(conn.Pairs, pair)

		err = consumeWhitespace(response)
		handleErr(err)

		if eof {
			break
		}
	}
}

// Success returns true if any connection data was read, false otherwise.
func (conn *HTTPConnection) Success() bool {
	return len(conn.Pairs) > 0
}

// Who is the request & response?
func (conn *HTTPConnection) sortStreams() (*bufio.Reader, *bufio.Reader, error) {
	a := bufio.NewReader(bytes.NewReader(conn.data[0]))
	b := bufio.NewReader(bytes.NewReader(conn.data[1]))
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
