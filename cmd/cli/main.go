package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/innermond/packong"
)

var (
	dimensions []string

	outname, unit, bigbox string
	wh                    []string
	width, height         float64

	tight bool

	plain, showDim bool

	cutwidth, topleftmargin float64

	mu, ml, pp, pd float64
)

func param() {
	var err error

	flag.StringVar(&outname, "o", "fit", "name of the maching project")
	flag.StringVar(&unit, "u", "mm", "unit of measurements")

	flag.StringVar(&bigbox, "bb", "0x0", "dimensions as \"wxh\" in units for bigest box / mother surface")

	flag.BoolVar(&tight, "tight", false, "when true only aria used tighten by height is taken into account")
	flag.BoolVar(&plain, "inkscape", true, "when false will save svg as inkscape svg")
	flag.BoolVar(&showDim, "showdim", false, "generate a layer with dimensions \"wxh\" regarding each box")
	flag.Float64Var(&mu, "mu", 15.0, "used material price per 1 square meter")
	flag.Float64Var(&ml, "ml", 5.0, "lost material price per 1 square meter")
	flag.Float64Var(&pp, "pp", 0.25, "perimeter price per 1 linear meter; used for evaluating cuts price")
	flag.Float64Var(&pd, "pd", 10, "travel price to location")
	flag.Float64Var(&cutwidth, "cutwidth", 0.0, "the with of material that is lost due to a cut")
	flag.Float64Var(&topleftmargin, "margin", 0.0, "offset from top left margin")

	flag.Parse()

	wh = strings.Split(bigbox, "x")
	switch len(wh) {
	case 1:
		wh = append(wh, wh[0])
	case 0:
		panicli("need to specify dimensions for big box")
	}
	width, err = strconv.ParseFloat(wh[0], 64)
	if err != nil {
		panicli("can't get width")
	}
	height, err = strconv.ParseFloat(wh[1], 64)
	if err != nil {
		panicli("can't get height")
	}
	dimensions = flag.Args()

	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "inkscape":
			plain = false
		case "tight":
			tight = true
		case "showdim":
			showDim = true
		}
	})
}

func main() {
	param()

	packong.NewOp(width, height, dimensions, unit).
		Outname(outname).
		Apearence(plain, showDim).
		Price(mu, ml, pp, pd).
		Fit()
}

func panicli(msg interface{}) {
	var code int

	switch msg.(type) {
	case string:
		code = 0
	case error:
		code = 1
	}
	fmt.Fprintln(os.Stdout, msg)
	os.Exit(code)
}
