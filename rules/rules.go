package rules

import (
	"github.com/Matir/httpwatch/httpsource"
	"log"
	"os"
)

type Rule struct {
	Name      string
	Operator  string
	Rules     []Rule
	Field     string
	Value     string
	evaluator Evaluator
}

var logger = log.New(os.Stderr, "rules: ", log.Lshortfile|log.Ltime)

func (r *Rule) Eval(pair *httpsource.RequestResponsePair) bool {
	if r.evaluator == nil {
		var err error
		r.evaluator, err = BuildEvaluator(r)
		if err != nil {
			panic(err)
		}
	}
	return r.evaluator.Eval(pair)
}

// SetLogger sets the logger for this package
func SetLogger(l *log.Logger) {
	logger = l
}
