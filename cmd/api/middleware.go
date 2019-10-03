package main

import (
	"net/http"
	"time"
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
