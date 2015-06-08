package rules

import (
	"github.com/Matir/httpwatch/httpsource"
	"testing"
)

type StubEvaluator struct {
	val bool
}

func (e *StubEvaluator) Eval(_ *httpsource.RequestResponsePair) bool {
	return e.val
}

func DummyGetter(_ *httpsource.RequestResponsePair) (string, error) {
	return "Dummy value", nil
}

func TestAndEvaluator(t *testing.T) {
	var err error
	tRule := Rule{evaluator: &StubEvaluator{true}}
	fRule := Rule{evaluator: &StubEvaluator{false}}
	r := Rule{
		Operator: "&&",
		Rules:    []Rule{tRule, fRule},
	}
	r.evaluator, err = BuildEvaluator(&r)
	if err != nil {
		t.Fatalf("Unable to build evaluator: %v\n", err)
	}
	if r.evaluator.Eval(nil) {
		t.Errorf("tRule & fRule == true!?")
	}
	r = Rule{
		Operator: "&&",
		Rules:    []Rule{tRule, tRule},
	}
	r.evaluator, err = BuildEvaluator(&r)
	if err != nil {
		t.Fatalf("Unable to build evaluator: %v\n", err)
	}
	if !r.evaluator.Eval(nil) {
		t.Errorf("tRule & tRule == false!?")
	}
}

func TestOrEvaluator(t *testing.T) {
	var err error
	tRule := Rule{evaluator: &StubEvaluator{true}}
	fRule := Rule{evaluator: &StubEvaluator{false}}
	r := Rule{
		Operator: "||",
		Rules:    []Rule{tRule, fRule},
	}
	r.evaluator, err = BuildEvaluator(&r)
	if err != nil {
		t.Fatalf("Unable to build evaluator: %v\n", err)
	}
	if !r.evaluator.Eval(nil) {
		t.Errorf("tRule | fRule == false!?")
	}
	r = Rule{
		Operator: "||",
		Rules:    []Rule{fRule, fRule},
	}
	r.evaluator, err = BuildEvaluator(&r)
	if err != nil {
		t.Fatalf("Unable to build evaluator: %v\n", err)
	}
	if r.evaluator.Eval(nil) {
		t.Errorf("fRule | fRule == true!?")
	}
}

func TestEqualsEvaluator(t *testing.T) {
	var err error
	r := Rule{
		Operator: "==",
		Value:    "Dummy value",
		Field:    "request.url",
	}
	r.evaluator, err = BuildEvaluator(&r)
	if err != nil {
		t.Fatalf("Unable to build evaluator: %v\n", err)
	}
	fg := FieldGetter(DummyGetter)
	r.evaluator.(*EqualsEvaluator).getter = &fg
	res := r.evaluator.Eval(nil)
	if !res {
		t.Errorf("Got false, expected true.")
	}
}

func TestInvalidOperator(t *testing.T) {
	r := Rule{
		Operator: "wtf",
		Field:    "request.url",
	}
	e, err := BuildEvaluator(&r)
	if err == nil || e != nil {
		t.Fatalf("Expected err, nil evaluator, got: %v, %v\n", e, err)
	}
}

func TestRegexEvaluator(t *testing.T) {
	var err error
	r := Rule{
		Operator: "~=",
		Field:    "request.url",
		Value:    "ummy",
	}
	r.evaluator, err = BuildEvaluator(&r)
	if err != nil {
		t.Fatalf("Unable to build evaluator: %v\n", err)
	}
	fg := FieldGetter(DummyGetter)
	r.evaluator.(*RegexEvaluator).getter = &fg
	res := r.evaluator.Eval(nil)
	if !res {
		t.Errorf("Expected val ~= ummy.\n")
	}
}
