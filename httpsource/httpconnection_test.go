package httpsource

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fatalIfErr(t *testing.T, err error) {
	if err == nil {
		return
	}
	t.Fatal(err)
}

func TestConsumeWhitespace(t *testing.T) {
	r := strings.NewReader("\r\n\r\nFoo")
	br := bufio.NewReader(r)
	if err := consumeWhitespace(br); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 1024)
	if n, err := br.Read(buf); err != nil {
		t.Fatal(err)
	} else {
		s := string(buf[:n])
		if s != "Foo" {
			t.Fatalf("Expected \"Foo\", got \"%v\"\n", s)
		}
	}
}

func TestReadConnection(t *testing.T) {
	reqs, err := os.Open(filepath.Join("testdata", "requests.txt"))
	fatalIfErr(t, err)
	resps, err := os.Open(filepath.Join("testdata", "responses.txt"))
	fatalIfErr(t, err)
	conn := HTTPConnection{}
	conn.readConnection(bufio.NewReader(reqs), bufio.NewReader(resps))
	if len(conn.Pairs) != 2 {
		t.Fatalf("Expected 2 pairs, got %d.\n", len(conn.Pairs))
	}
	if conn.err != nil {
		t.Fatalf("Got an error: %v\n", conn.err)
	}
}
