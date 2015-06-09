package rules

import (
	"github.com/Matir/httpwatch/httpsource"
	"sync"
)

type ruleContainer struct {
	Rule
	input <-chan *httpsource.RequestResponsePair
}

// RuleEngine manages running RequestResponsePairs through the Rules defined
type RuleEngine struct {
	Matches    <-chan *httpsource.RequestResponsePair
	rules      []ruleContainer
	mux        httpsource.PairMux
	running    bool
	lock       sync.Mutex
	rawMatches chan *httpsource.RequestResponsePair
	finished   chan *ruleContainer
	allDone    chan bool
}

// NewRuleEngine creates a RuleEngine for the given set of Rules and the PairMux
// given.
func NewRuleEngine(rules []Rule, mux httpsource.PairMux) *RuleEngine {
	r := &RuleEngine{mux: mux}
	r.rawMatches = make(chan *httpsource.RequestResponsePair, 100)
	r.Matches = makeDeduplicatingChannel(r.rawMatches)
	r.finished = make(chan *ruleContainer, len(rules)*2)
	r.allDone = make(chan bool)
	for _, rule := range rules {
		r.AddRule(rule)
	}
	return r
}

// AddRule adds a Rule to the RuleEngine, and starts it if the RuleEngine is
// already running.
func (r *RuleEngine) AddRule(rule Rule) {
	r.lock.Lock()
	defer r.lock.Unlock()
	name := "rule:" + rule.Name
	rc := ruleContainer{rule, r.mux.AddOutput(name, 10)}
	r.rules = append(r.rules, rc)
	if r.running {
		r.startRule(rc)
	}
}

// Start starts each of the rules in a goroutine
// TODO: make it stop everything if input ceases
func (r *RuleEngine) Start() {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.running {
		return
	}
	r.running = true
	// Start the underlying mux, if it is not already
	r.mux.Start()
	// Start up the shutdown watcher
	go r.watchFinished()
	for _, rc := range r.rules {
		r.startRule(rc)
	}
}

// Running returns true if the RuleEngine is running
func (r *RuleEngine) Running() bool {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.running
}

func (r *RuleEngine) startRule(rule ruleContainer) {
	go func() {
		for item := range rule.input {
			if !r.Running() {
				break
			}
			if rule.Eval(item) {
				r.rawMatches <- item
			}
		}
		r.finished <- &rule
	}()
}

// Goroutine to watch for finished workers
func (r *RuleEngine) watchFinished() {
	for done := range r.finished {
		logger.Printf("Received finished signal.\n")
		if func() bool {
			r.lock.Lock()
			defer r.lock.Unlock()
			for i, rc := range r.rules {
				if rc.Name == done.Name {
					r.rules = append(r.rules[:i], r.rules[i+1:]...)
					break
				}
				logger.Printf("Got finished signal for %s, but not found in %s.\n", done.Name, rc.Name)
			}
			if len(r.rules) == 0 {
				r.allDone <- true
				close(r.rawMatches)
				return true
			}
			return false
		}() {
			return
		}
	}
}

func (r *RuleEngine) WaitUntilFinished() {
	<-r.allDone
}

func makeDeduplicatingChannel(input <-chan *httpsource.RequestResponsePair) <-chan *httpsource.RequestResponsePair {
	output := make(chan *httpsource.RequestResponsePair, cap(input))
	go func() {
		seen := make(map[string]bool)
		for pair := range input {
			fp := pair.Fingerprint()
			if _, ok := seen[fp]; !ok {
				seen[fp] = true
				output <- pair
			}
		}
		logger.Printf("Closing dedupe output...\n")
		close(output)
	}()
	return output
}
