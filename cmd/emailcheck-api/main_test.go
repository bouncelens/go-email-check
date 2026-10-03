package main

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	emailcheck "github.com/bouncelens/go-email-check"
)

// Fake DNS: gmail.com has MX, everything else does not exist.
func fakeLookup(_ context.Context, d string) emailcheck.DomainInfo {
	if d == "gmail.com" {
		return emailcheck.DomainInfo{Exists: true, MX: []string{"gmail-smtp-in.l.google.com"}, Provider: "Google"}
	}
	return emailcheck.DomainInfo{}
}

type response struct {
	Summary emailcheck.Summary  `json:"summary"`
	Results []emailcheck.Result `json:"results"`
	Error   string              `json:"error"`
}

func call(t *testing.T, method, url, body string) (int, response) {
	t.Helper()
	req := httptest.NewRequest(method, url, strings.NewReader(body))
	rec := httptest.NewRecorder()
	newHandler(3, fakeLookup).ServeHTTP(rec, req)
	var r response
	_ = json.Unmarshal(rec.Body.Bytes(), &r)
	return rec.Code, r
}

func TestGetOne(t *testing.T) {
	code, r := call(t, "GET", "/api/check?email=jane@gmial.com", "")
	if code != 200 || len(r.Results) != 1 || r.Results[0].Status != emailcheck.StatusInvalid || *r.Results[0].DidYouMean != "jane@gmail.com" {
		t.Fatalf("got %d %+v", code, r)
	}
}

func TestPostList(t *testing.T) {
	code, r := call(t, "POST", "/api/check", `{"emails":["jane@gmail.com","x@nope-bl.com","bad@"]}`)
	if code != 200 || r.Summary.Unconfirmed != 1 || r.Summary.Invalid != 2 {
		t.Fatalf("got %d %+v", code, r.Summary)
	}
}

func TestErrors(t *testing.T) {
	for _, c := range []struct {
		method, url, body string
		want              int
	}{
		{"GET", "/api/check", "", 400},
		{"POST", "/api/check", "not json", 400},
		{"POST", "/api/check", `{"emails":["a@b.co","c@d.co","e@f.co","g@h.co"]}`, 413},
		{"DELETE", "/api/check", "", 405},
		{"GET", "/nope", "", 404},
	} {
		if code, _ := call(t, c.method, c.url, c.body); code != c.want {
			t.Errorf("%s %s: got %d want %d", c.method, c.url, code, c.want)
		}
	}
}

func TestHealthAndCORS(t *testing.T) {
	rec := httptest.NewRecorder()
	newHandler(3, fakeLookup).ServeHTTP(rec, httptest.NewRequest("OPTIONS", "/api/check", nil))
	if rec.Code != 204 || rec.Header().Get("access-control-allow-origin") != "*" {
		t.Fatalf("preflight: %d %v", rec.Code, rec.Header())
	}
	if code, _ := call(t, "GET", "/healthz", ""); code != 200 {
		t.Fatalf("healthz %d", code)
	}
}
