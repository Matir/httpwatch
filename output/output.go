package output

import (
	"fmt"
	"github.com/Matir/httpwatch/httpsource"
	"log"
	"os"
	"sync"
)

var logger = log.New(os.Stderr, "output: ", log.Lshortfile|log.Ltime)

type OutputSink interface {
	Write(*httpsource.RequestResponsePair)
}

type OutputEngine struct {
	mux      httpsource.PairMux
	finished chan bool
	allDone  chan bool
	lock     sync.Mutex
	active   int
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
	e := OutputEngine{mux: httpsource.NewBlockingPairMux(input)}
	e.finished = make(chan bool)
	e.allDone = make(chan bool, 1)
	return e
}

func (e *OutputEngine) AddOutput(name string, options map[string]string) error {
	e.lock.Lock()
	defer e.lock.Unlock()
	o := GetOutputSink(name, options)
	if o == nil {
		return fmt.Errorf("Invalid output type %s", name)
	}
	c := e.mux.AddOutput("output:"+name, 20)
	e.active++
	go func() {
		for pair := range c {
			o.Write(pair)
		}
		e.finished <- true
	}()
	return nil
}

func (e *OutputEngine) Start() {
	e.mux.Start()
	go func() {
		for _ = range e.finished {
			logger.Printf("Output finished, %d active...\n", e.active)
			if func() bool {
				e.lock.Lock()
				defer e.lock.Unlock()
				e.active--
				if e.active == 0 {
					e.allDone <- true
					return true
				}
				return false
			}() {
				return
			}
		}
	}()
}

func (e *OutputEngine) WaitUntilFinished() {
	<-e.allDone
}

// SetLogger sets the logger for this package
func SetLogger(l *log.Logger) {
	logger = l
}
