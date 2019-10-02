package main

import "flag"

// it shows every step
var verbose bool

func param() {
	flag.BoolVar(&verbose, "verbose", false, "tell me more about you")
	flag.Parse()
}
