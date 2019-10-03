package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {

	var port int
	flag.IntVar(&port, "p", 2222, "set port number '-p <port number>'")

	var concurencyPeak int
	flag.IntVar(&concurencyPeak, "c", 10, "set connections concurency maximum limit '-c 20'")

	var timePeak int
	flag.IntVar(&timePeak, "t", 0, "set a time limiter in milliseconds; no more than a request in that time '-t 200'")

	flag.Parse()

	param()

	var (
		fn          http.HandlerFunc
		limiterInfo string
	)
	if timePeak > 0 {
		fn = limiterByTime(http.HandlerFunc(fitboxes), timePeak)
		limiterInfo = fmt.Sprintf("concurency peak one request in time %d\n", timePeak)
	} else {
		fn = limiter(http.HandlerFunc(fitboxes), concurencyPeak)
		limiterInfo = fmt.Sprintf("concurency peak max requests %d\n", concurencyPeak)
	}

	var addr = fmt.Sprintf(":%d", port)

	s := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  100 * time.Second,
		Addr:         addr,
		Handler:      fn,
	}
	log.Println(
		"\nmain: starting server\n" +
			fmt.Sprintf("address %s\n", addr) +
			limiterInfo,
	)
	err := s.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
