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
	for c := range src.Connections {
		for _, pair := range c.Pairs {
			t.Logf("Request line: %s %s %s\n", pair.Request.Method, pair.Request.URL.String(), pair.Request.Proto)
		}
	}
}
