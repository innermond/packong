package main

import (
	"log"
	"net/http"
	"time"
)

func main() {

	param()
	s := &http.Server{
		ReadHeaderTimeout: 20 * time.Second,
		Addr:              ":2222",
		Handler:           http.HandlerFunc(fitBoxes),
	}
	log.Println("starting server...")
	err := s.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
