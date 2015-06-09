package output

import (
	"github.com/Matir/httpwatch/httpsource"
)

type OutputSink interface {
	Write(*httpsource.RequestResponsePair)
}

type OutputSinkBuilder func(options map[string]string) OutputSink

var outputSinkRegistry = make(map[string]OutputSinkBuilder)

func GetOutputSink(name string, options map[string]string) OutputSink {
	if builder, ok := outputSinkRegistry[name]; ok {
		return builder(options)
	}
	return nil
}
