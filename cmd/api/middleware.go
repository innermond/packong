package main

import (
	"log"
	"net/http"
	"time"

	"github.com/innermond/packong/cmd/api/requestid"
)

func limiter(f http.HandlerFunc, max int) http.HandlerFunc {
	// semaphore
	sem := make(chan struct{}, max)

	return func(w http.ResponseWriter, r *http.Request) {
		// blocks if semaphore is full
		sem <- struct{}{}
		// dequeue semaphore
		defer func() { <-sem }()

		// execute
		f(w, r)
	}
}

func limiterByTime(f http.HandlerFunc, max int) http.HandlerFunc {
	// semaphore
	sem := time.Tick(time.Duration(max) * time.Millisecond)

	return func(w http.ResponseWriter, r *http.Request) {
		// blocks waiting next tick
		<-sem
		// execute
		f(w, r)
	}
}

func getid(r *http.Request) string {
	id, ok := requestid.FromContext(r.Context())
	if ok == false {
		return "unexpected"
	}
	if id == "" {
		return "unknown"
	}
	return id
}

func logRequest(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		f(w, r)

		// log request by who(IP address)
		ip := r.RemoteAddr
		id := getid(r)

		log.Printf(
			"%s\t%s\t%s\t%s\t%v",
			id,
			r.Method,
			r.RequestURI,
			ip,
			time.Since(start),
		)
	}
}
