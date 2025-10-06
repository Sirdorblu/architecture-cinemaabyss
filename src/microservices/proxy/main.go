package main

import (
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func parseTarget(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		log.Fatalf("bad target %q: %v", raw, err)
	}
	return u
}
func proxyTo(target *url.URL, w http.ResponseWriter, r *http.Request) {
	rp := httputil.NewSingleHostReverseProxy(target)
	orig := rp.Director
	rp.Director = func(req *http.Request) {
		orig(req)
		req.URL.Scheme, req.URL.Host, req.Host = target.Scheme, target.Host, target.Host
		req.Header.Set("X-Forwarded-Host", r.Host)
		req.Header.Set("X-Forwarded-Proto", "http")
	}
	rp.ServeHTTP(w, r)
}

func main() {
	rand.Seed(time.Now().UnixNano())

	port := env("PORT", "8000")
	monolithURL := env("MONOLITH_URL", "http://monolith:8080")
	moviesURL := env("MOVIES_SERVICE_URL", "http://movies-service:8081")
	eventsURL := env("EVENTS_SERVICE_URL", "http://events-service:8082")
	gradual := strings.EqualFold(env("GRADUAL_MIGRATION", "true"), "true")
	percent, _ := strconv.Atoi(env("MOVIES_MIGRATION_PERCENT", "0"))
	if percent < 0 || percent > 100 {
		percent = 0
	}

	monolith := parseTarget(monolithURL)
	movies := parseTarget(moviesURL)
	events := parseTarget(eventsURL)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("Strangler Fig Proxy is healthy"))
	})

	mux.HandleFunc("/api/movies", func(w http.ResponseWriter, r *http.Request) {
		target := monolith
		if gradual && rand.Intn(100) < percent {
			target = movies
		}
		proxyTo(target, w, r)
	})
	mux.HandleFunc("/api/movies/", func(w http.ResponseWriter, r *http.Request) {
		target := monolith
		if gradual && rand.Intn(100) < percent {
			target = movies
		}
		proxyTo(target, w, r)
	})

	mux.HandleFunc("/api/events", func(w http.ResponseWriter, r *http.Request) {
		proxyTo(events, w, r)
	})
	mux.HandleFunc("/api/events/", func(w http.ResponseWriter, r *http.Request) {
		proxyTo(events, w, r)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		proxyTo(monolith, w, r)
	})

	log.Printf("Proxy :%s | monolith=%s | movies=%s | events=%s | gradual=%v percent=%d",
		port, monolithURL, moviesURL, eventsURL, gradual, percent)

	srv := &http.Server{Addr: ":" + port, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	log.Fatal(srv.ListenAndServe())
}
