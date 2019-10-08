package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/innermond/packong/cmd/api/requestid"
)

func main() {

	var concurencyPeakEnv, timePeakEnv int
	{
		var err error
		concurencyPeakEnv, err = strconv.Atoi(env("PACKONG_CONCURENCY", "10"))
		if err != nil {
			concurencyPeakEnv = 10
		}

		timePeakEnv, err = strconv.Atoi(env("PACKONG_TIME", "0"))
		if err != nil {
			timePeakEnv = 0
		}
	}

	var port string
	flag.StringVar(&port, "p", env("PACKONG_PORT", "2222"), "set port number '-p <port number>'")

	var concurencyPeak int
	flag.IntVar(&concurencyPeak, "c", concurencyPeakEnv, "set connections concurency maximum limit '-c 20'")

	var timePeak int
	flag.IntVar(&timePeak, "t", timePeakEnv, "set a time limiter in milliseconds; no more than a request in that time '-t 200'")

	flag.Parse()

	param()

	var (
		fn          http.HandlerFunc
		limiterInfo string
	)

	var reqid func(http.HandlerFunc) http.HandlerFunc = requestid.Handler
	fn = reqid(logRequest(fitboxes))

	if timePeak > 0 {
		fn = limiterByTime(http.HandlerFunc(fn), timePeak)
		limiterInfo = fmt.Sprintf("concurency peak one request in time %d\n", timePeak)
	} else {
		fn = limiter(http.HandlerFunc(fn), concurencyPeak)
		limiterInfo = fmt.Sprintf("concurency peak max requests %d\n", concurencyPeak)
	}

	var addr = fmt.Sprintf(":%s", port)

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

	// ctrl+c
	done := make(chan struct{}, 1)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGTRAP)

	go func() {
		// wait for closing signal
		<-quit

		log.Print("server shutting down...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		s.SetKeepAlivesEnabled(false)
		err := s.Shutdown(ctx)
		if err != nil {
			log.Fatalf("server cold brutal %v\n", err)
		}

		close(done)
	}()

	// blocks here doing serving
	err := s.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}

	<-done
	log.Println("server cold")
}

func env(key, alternative string) string {
	if value, found := os.LookupEnv(key); found {
		return value
	}
	return alternative
}
