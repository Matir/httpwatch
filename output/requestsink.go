package output

import (
	"fmt"
	"github.com/Matir/httpwatch/httpsource"
	"os"
)

type requestSink struct {
	fp *os.File
}

// TODO: support alternate files
func makeRequestSink(_ map[string]string) OutputSink {
	return &requestSink{os.Stdout}
}

func (s *requestSink) Write(pair *httpsource.RequestResponsePair) {
	fmt.Fprintf(s.fp, "%s %s\n", pair.Request.Method, pair.Request.URL.String())
}

func init() {
	outputSinkRegistry["request"] = makeRequestSink
}
