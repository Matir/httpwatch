package rules

import (
	"github.com/Matir/httpwatch/httpsource"
	"net/http"
	"strings"
	"testing"
)

func TestBuildGetter(t *testing.T) {
	errcnds := []string{"", "non-existent", "foo.bar", "request.foo.bar", "request.foo"}
	for _, s := range errcnds {
		if g, err := buildGetter(s); g != nil || err == nil {
			t.Errorf("Expected an error, got %v %v\n", g, err)
		}
	}
}

func TestRequestBodyGetter(t *testing.T) {
	target := "targetstring"
	pair := httpsource.RequestResponsePair{RequestBody: []byte(target)}
	if v, err := requestBodyGetter(&pair); v != target || err != nil {
		t.Errorf("Expected %s, got %s.\n", target, v)
	}
}

func TestHeaderGetterFull(t *testing.T) {
	ct := "text/plain"
	req := http.Request{Header: make(http.Header)}
	req.Header["Content-Type"] = []string{ct}
	resp := http.Response{Header: make(http.Header)}
	resp.Header["Content-Type"] = []string{ct}
	pair := httpsource.RequestResponsePair{Request: &req, Response: &resp}

	expected := []struct {
		getter, value string
	}{
		{"request.header.Content-Type", ct},
		{"request.header.missing", ""},
		{"response.header.content-type", ct},
	}

	for _, test := range expected {
		g, err := buildGetter(test.getter)
		if err != nil {
			t.Fatalf("Error building getter: %v\n", err)
		}
		val, err := g(&pair)
		if err != nil {
			t.Errorf("Error parsing header: %v\n", err)
		}
		if val != test.value {
			t.Errorf("%v: Expected %s, got %s.\n", test.getter, test.value, val)
		}
	}
}

func TestURLPartGetters(t *testing.T) {
	uri := "https://github.com/Matir/httpwatch?params=1"
	req, err := requestFromURI(uri)
	if err != nil {
		t.Fatalf("Error from NewRequest: %v\n", err)
	}
	pair := httpsource.RequestResponsePair{Request: req}
	tests := []struct {
		field, value string
	}{
		{"scheme", "https"},
		{"host", "github.com"},
		{"path", "/Matir/httpwatch"},
		{"query", "params=1"},
		{"fragment", ""},
	}
	for _, test := range tests {
		field := "request.url." + test.field
		g, err := buildGetter(field)
		if err != nil {
			t.Errorf("Error building: %v\n", err)
		}
		val, err := g(&pair)
		if err != nil {
			t.Errorf("Error getting val: %v\n", err)
		}
		if val != test.value {
			t.Errorf("Got %v, expected %v.\n", val, test.value)
		}
	}
	g, err := buildGetter("request.url.foo")
	if err == nil {
		t.Errorf("Expected an error, got %v\n", g)
	}
}

func TestURLGetter(t *testing.T) {
	uri := "https://github.com/Matir/httpwatch?params=1"
	req, err := requestFromURI(uri)
	if err != nil {
		t.Fatalf("Error from NewRequest: %v\n", err)
	}
	pair := httpsource.RequestResponsePair{Request: req}
	g, err := buildGetter("request.url")
	if err != nil {
		t.Errorf("Expected no error, got %v\n", err)
	}
	val, err := g(&pair)
	if err != nil {
		t.Errorf("Expected no error, got %v\n", err)
	}
	if val != uri {
		t.Errorf("Expected %v, got %v\n", uri, val)
	}
}

func requestFromURI(uri string) (*http.Request, error) {
	body := strings.NewReader("<html>")
	return http.NewRequest("GET", uri, body)
}
