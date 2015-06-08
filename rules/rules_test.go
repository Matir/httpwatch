package rules

import (
	"github.com/Matir/httpwatch/httpsource"
	"net/http"
	"strings"
	"testing"
)

// This tests the whole stack
func TestRules(t *testing.T) {
	data := "This is some dummy data."
	buf := strings.NewReader(data)
	url := "https://github.com/Matir"
	req, _ := http.NewRequest("GET", url, buf)
	resp := http.Response{}
	pair := httpsource.RequestResponsePair{
		Request: req, RequestBody: []byte(data), Response: &resp, ResponseBody: []byte(data)}

	rule := Rule{
		Operator: "==",
		Field:    "request.url",
		Value:    url,
	}
	if !rule.Eval(&pair) {
		t.Error("Expected request.url == url.\n")
	}
}
