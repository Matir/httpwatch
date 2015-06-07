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
}
