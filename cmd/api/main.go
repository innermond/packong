package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/innermond/packong"
)

func main() {
	http.HandleFunc("/", fitBoxes)
	err := http.ListenAndServe(":2222", nil)
	if err != nil {
		log.Fatal(err)
	}
}

func fitBoxes(w http.ResponseWriter, r *http.Request) {
	// parameters
	var (
		err error

		dimensions []string

		outname, unit string
		str           string
		width, height float64

		tight bool

		plain, showDim bool

		cutwidth, topleftmargin float64

		mu, ml, pp, pd float64
	)

	str = r.FormValue("width")
	width, err = strconv.ParseFloat(str, 64)
	if err != nil {
		panic("can't get width")
	}

	str = r.FormValue("height")
	height, err = strconv.ParseFloat(str, 64)
	if err != nil {
		panic("can't get height")
	}

	str = r.FormValue("cutwidth")
	cutwidth, err = strconv.ParseFloat(str, 64)
	if err != nil {
		panic("can't get cutwidth")
	}

	str = r.FormValue("mu")
	mu, err = strconv.ParseFloat(str, 64)
	if err != nil {
		panic("can't get mu")
	}

	str = r.FormValue("ml")
	ml, err = strconv.ParseFloat(str, 64)
	if err != nil {
		panic("can't get ml")
	}

	str = r.FormValue("pp")
	pp, err = strconv.ParseFloat(str, 64)
	if err != nil {
		panic("can't get pp")
	}

	str = r.FormValue("pd")
	pd, err = strconv.ParseFloat(str, 64)
	if err != nil {
		panic("can't get pd")
	}

	str = r.FormValue("topleftmargin")
	topleftmargin, err = strconv.ParseFloat(str, 64)
	if err != nil {
		panic("can't get topleftmargin")
	}

	str = r.FormValue("tight")
	tight, err = strconv.ParseBool(str)
	if err != nil {
		panic("can't get topleftmargin")
	}

	str = r.FormValue("plain")
	plain, err = strconv.ParseBool(str)
	if err != nil {
		panic("can't get plain")
	}

	str = r.FormValue("showdim")
	showDim, err = strconv.ParseBool(str)
	if err != nil {
		panic("can't get showdim")
	}

	outname = r.FormValue("outname")
	unit = r.FormValue("unit")
	dimensions = strings.Fields(r.FormValue("dimensions"))
	// second parameters outs will give svg
	rep, _, err := packong.NewOp(width, height, dimensions, unit).
		Outname(outname).
		Tight(tight).
		Topleft(topleftmargin).
		Cutwidth(cutwidth).
		Appearance(plain, showDim).
		Price(mu, ml, pp, pd).
		Fit()

	if err != nil {
		panic(err)
	}

	b, err := json.Marshal(rep)
	if err != nil {
		panic(err)
	}
	io.Copy(w, bytes.NewReader(b))
}
