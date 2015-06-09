package output

import (
	"fmt"
	"github.com/Matir/httpwatch/httpsource"
)

type OutputSink interface {
	Write(*httpsource.RequestResponsePair)
}

type OutputEngine struct {
	mux httpsource.PairMux
}

type OutputSinkBuilder func(options map[string]string) OutputSink

var outputSinkRegistry = make(map[string]OutputSinkBuilder)

func GetOutputSink(name string, options map[string]string) OutputSink {
	if builder, ok := outputSinkRegistry[name]; ok {
		return builder(options)
	}
	return nil
}

func NewOutputEngine(input <-chan *httpsource.RequestResponsePair) OutputEngine {
	return OutputEngine{mux: httpsource.NewBlockingPairMux(input)}
}

func (e *OutputEngine) AddOutput(name string, options map[string]string) error {
	o := GetOutputSink(name, options)
	if o == nil {
		return fmt.Errorf("Invalid output type %s", name)
	}
	c := e.mux.AddOutput("output:"+name, 20)
	go func() {
		for pair := range c {
			o.Write(pair)
		}
	}()
	return nil
}
