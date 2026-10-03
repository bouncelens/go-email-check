// Command emailcheck-api is a self-hosted HTTP API for the BounceLens email checks
// (typos, disposable domains, dead domains, role addresses, duplicates). No API key.
// Same request and response format as the API at https://bouncelens.com/.
//
//	GET  /api/check?email=jane@gmial.com
//	POST /api/check   {"emails": ["a@b.com", ...]}   (max 500)
//	GET  /healthz
//
// Environment: PORT (default 8080), MAX_EMAILS (default 500).
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	emailcheck "github.com/bouncelens/go-email-check"
)

const version = "1.1.0"

func main() {
	port := getenv("PORT", "8080")
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           newHandler(atoi(getenv("MAX_EMAILS", "500"), 500), nil),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("emailcheck-api %s listening on :%s (https://bouncelens.com/)", version, port)
	log.Fatal(srv.ListenAndServe())
}

// newHandler builds the API. lookup is nil in production (DNS over HTTPS); the tests replace it.
func newHandler(maxEmails int, lookup func(context.Context, string) emailcheck.DomainInfo) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": version})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Not found. Use /api/check"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{
			"name":    "BounceLens email check API (self-hosted)",
			"version": version,
			"usage":   `GET /api/check?email=jane@example.com or POST /api/check {"emails": [...]}`,
			"docs":    "https://github.com/bouncelens/go-email-check",
			"website": "https://bouncelens.com/",
		})
	})
	mux.HandleFunc("/api/check", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("access-control-allow-origin", "*")
		w.Header().Set("access-control-allow-headers", "content-type")
		var emails []string
		switch r.Method {
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
			return
		case http.MethodGet:
			if e := r.URL.Query().Get("email"); e != "" {
				emails = []string{e}
			}
		case http.MethodPost:
			var body struct {
				Emails []string `json:"emails"`
			}
			if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": `Body must be JSON: {"emails": [...]}`})
				return
			}
			emails = body.Emails
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": `Use GET ?email= or POST {"emails": [...]}`})
			return
		}
		if len(emails) == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "No emails given"})
			return
		}
		if len(emails) > maxEmails {
			writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "Max " + strconv.Itoa(maxEmails) + " emails per call"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()
		results := emailcheck.CheckWith(ctx, emails, emailcheck.Options{Lookup: lookup, Concurrency: 6})
		writeJSON(w, http.StatusOK, map[string]any{"summary": emailcheck.Summarize(results), "results": results})
	})
	return mux
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func atoi(s string, def int) int {
	if n, err := strconv.Atoi(s); err == nil && n > 0 {
		return n
	}
	return def
}
