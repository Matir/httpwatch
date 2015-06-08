package rules

import (
	"errors"
	"fmt"
	"github.com/Matir/httpwatch/httpsource"
	"net/http"
	"net/url"
	"strings"
)

type FieldGetter func(*httpsource.RequestResponsePair) (string, error)

// Possible getters
func buildGetter(value string) (FieldGetter, error) {
	if value == "" {
		return nil, errors.New("No field specified.")
	}
	rr, remains, err := splitFirst(value, ".")
	if err != nil {
		return nil, err
	}
	if rr != "request" && rr != "response" {
		return nil, fmt.Errorf("Unknown entity: %s", rr)
	}
	if strings.ContainsRune(remains, '.') {
		field, attribute, _ := splitFirst(value, ".")
		return buildTwoPartGetter(rr, field, attribute)
	}
	return buildOnePartGetter(rr, remains)
}

func buildTwoPartGetter(rr, field, attribute string) (FieldGetter, error) {
	switch field {
	case "header":
		return buildHeaderValueGetter(rr, attribute), nil
	}
	return nil, fmt.Errorf("Unknown field: %s", field)
}

func buildOnePartGetter(rr, field string) (FieldGetter, error) {
	switch rr {
	case "request":
		switch field {
		case "url":
			return buildURLGetter(), nil
		case "body":
			return requestBodyGetter, nil
		case "method":
			return requestMethodGetter, nil
		case "host":
			return requestHostGetter, nil
		}
	case "response":
		switch field {
		case "body":
			return responseBodyGetter, nil
		case "code":
			return responseCodeGetter, nil
		case "status":
			return responseStatusGetter, nil
		}
	}
	return nil, fmt.Errorf("Unknown field: %s", field)
}

// Build header matching code
func requestHeaderGetter(pair *httpsource.RequestResponsePair) http.Header {
	return pair.Request.Header
}

func responseHeaderGetter(pair *httpsource.RequestResponsePair) http.Header {
	return pair.Response.Header
}

func buildHeaderValueGetter(rr, name string) FieldGetter {
	getter := requestHeaderGetter
	if rr == "response" {
		getter = responseHeaderGetter
	}
	return func(pair *httpsource.RequestResponsePair) (string, error) {
		h := getter(pair)
		if val, ok := h[name]; ok {
			return strings.Join(val, ";"), nil
		}
		return "", nil
	}
}

// Build URL matching code
func buildURLGetter() FieldGetter {
	return func(pair *httpsource.RequestResponsePair) (string, error) {
		u := pair.Request.URL
		if u == nil {
			return "", errors.New("No URL in Request.")
		}
		return u.String(), nil
	}
}

// Get a single element from the URL
func buildURLFieldGetter(field string) (FieldGetter, error) {
	var getter func(u *url.URL) string
	switch field {
	case "scheme":
		getter = func(u *url.URL) string { return u.Scheme }
	case "host":
		getter = func(u *url.URL) string { return u.Host }
	case "path":
		getter = func(u *url.URL) string { return u.Path }
	case "query", "querystring":
		getter = func(u *url.URL) string { return u.RawQuery }
	case "fragment":
		getter = func(u *url.URL) string { return u.Fragment }
	default:
		return nil, fmt.Errorf("Unknown field: %s", field)
	}

	return func(pair *httpsource.RequestResponsePair) (string, error) {
		u := pair.Request.URL
		if u == nil {
			return "", errors.New("No URL in Request")
		}
		return getter(u), nil
	}, nil
}

// Literal getters
func requestBodyGetter(pair *httpsource.RequestResponsePair) (string, error) {
	return string(pair.RequestBody), nil
}

func responseBodyGetter(pair *httpsource.RequestResponsePair) (string, error) {
	return string(pair.ResponseBody), nil
}

func requestMethodGetter(pair *httpsource.RequestResponsePair) (string, error) {
	return pair.Request.Method, nil
}

func requestHostGetter(pair *httpsource.RequestResponsePair) (string, error) {
	return pair.Request.Host, nil
}

func responseCodeGetter(pair *httpsource.RequestResponsePair) (string, error) {
	return string(pair.Response.StatusCode), nil
}

func responseStatusGetter(pair *httpsource.RequestResponsePair) (string, error) {
	return pair.Response.Status, nil
}

// Utility functions
func splitFirst(s, sep string) (string, string, error) {
	items := strings.SplitN(s, sep, 2)
	if len(items) != 2 {
		return "", "", fmt.Errorf("Separator %s not in %v.", sep, s)
	}
	return items[0], items[1], nil
}
