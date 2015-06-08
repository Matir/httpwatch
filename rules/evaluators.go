package rules

import (
	"fmt"
	"github.com/Matir/httpwatch/httpsource"
	"regexp"
	"strings"
)

type Evaluator interface {
	Eval(*httpsource.RequestResponsePair) bool
}

type abstractEvaluator struct {
	rule   *Rule
	getter *FieldGetter
}

func BuildEvaluator(r *Rule) (Evaluator, error) {
	// Operators that don't require value
	switch r.Operator {
	case "&&", "and":
		return &AndEvaluator{rule: r}, nil
	case "||", "or":
		return &OrEvaluator{rule: r}, nil
	}

	// These do require a value from the request/response
	getter, err := buildGetter(r.Field)
	if err != nil {
		return nil, err
	}
	switch r.Operator {
	case "==":
		return &EqualsEvaluator{r, &getter}, nil
	case "!=":
		return &NotEqualsEvaluator{EqualsEvaluator{r, &getter}}, nil
	case "~=":
		re, err := regexp.Compile(r.Value)
		if err != nil {
			return nil, err
		}
		return &RegexEvaluator{r, &getter, re}, nil
	}
	return nil, fmt.Errorf("Invalid operator: %s", r.Operator)
}

// Possible operations
type AndEvaluator abstractEvaluator
type OrEvaluator abstractEvaluator
type EqualsEvaluator abstractEvaluator
type ContainsEvaluator abstractEvaluator
type NotEqualsEvaluator struct {
	EqualsEvaluator
}
type RegexEvaluator struct {
	rule   *Rule
	getter *FieldGetter
	re     *regexp.Regexp
}

func (e *AndEvaluator) Eval(pair *httpsource.RequestResponsePair) bool {
	for _, r := range e.rule.Rules {
		if !r.Eval(pair) {
			return false
		}
	}
	return true
}

func (e *OrEvaluator) Eval(pair *httpsource.RequestResponsePair) bool {
	for _, r := range e.rule.Rules {
		if r.Eval(pair) {
			return true
		}
	}
	return false
}

func (e *EqualsEvaluator) Eval(pair *httpsource.RequestResponsePair) bool {
	val, err := (*e.getter)(pair)
	if err != nil {
		// TODO: log this error
		return false
	}
	return val == e.rule.Value
}

func (e *NotEqualsEvaluator) Eval(pair *httpsource.RequestResponsePair) bool {
	return !e.EqualsEvaluator.Eval(pair)
}

func (e *ContainsEvaluator) Eval(pair *httpsource.RequestResponsePair) bool {
	val, err := (*e.getter)(pair)
	if err != nil {
		// TODO: log this error
		return false
	}
	return strings.Contains(val, e.rule.Value)
}

func (e *RegexEvaluator) Eval(pair *httpsource.RequestResponsePair) bool {
	val, err := (*e.getter)(pair)
	if err != nil {
		// TODO: log this error
		return false
	}
	return e.re.MatchString(val)
}
