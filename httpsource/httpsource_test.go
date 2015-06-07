package httpsource

import (
	"path/filepath"
	"testing"
)

func TestEndToEnd(t *testing.T) {
	fname := filepath.Join("testdata", "e2e.pcap")
	src := NewHTTPSource()
	src.AddPCAPFile(fname)
	src.WaitUntilFinished()
	t.Logf("Read %d connections.\n", len(src.Connections))
	src.ConvertConnectionsToPairs()
	t.Logf("Read %d request/response pairs.\n", len(src.Pairs))
	for pair := range src.Pairs {
		t.Logf("Request line: %s %s %s\n", pair.Request.Method, pair.Request.URL.String(), pair.Request.Proto)
	}
}

func TestAddPCAPFile(t *testing.T) {
	src := NewHTTPSource()
	err := src.AddPCAPFile("nonexistent")
	if err == nil {
		t.Fatal("Expected failure, got nil error.\n")
	}
}
