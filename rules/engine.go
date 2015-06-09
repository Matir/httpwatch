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
}

// NewRuleEngine creates a RuleEngine for the given set of Rules and the PairMux
// given.
func NewRuleEngine(rules []Rule, mux httpsource.PairMux) *RuleEngine {
	r := &RuleEngine{mux: mux}
	r.rawMatches = make(chan *httpsource.RequestResponsePair, 100)
	r.Matches = makeDeduplicatingChannel(r.rawMatches)
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
		for {
			item, ok := <-rule.input
			if !ok || !r.Running() {
				return
			}
			if rule.Eval(item) {
				r.rawMatches <- item
			}
		}
	}()
}

func makeDeduplicatingChannel(input <-chan *httpsource.RequestResponsePair) <-chan *httpsource.RequestResponsePair {
	output := make(chan *httpsource.RequestResponsePair, cap(input))
	go func() {
		seen := make(map[string]bool)
		for {
			pair, ok := <-input
			if !ok {
				close(output)
				return
			}
			fp := pair.Fingerprint()
			if _, ok := seen[fp]; !ok {
				seen[fp] = true
				output <- pair
			}
		}
	}()
	return output
}
