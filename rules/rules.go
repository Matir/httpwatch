package rules

import (
	"github.com/Matir/httpwatch/httpsource"
)

type Rule struct {
	Name      string
	Operator  string
	Rules     []Rule
	Field     string
	Value     string
	evaluator Evaluator
}

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
