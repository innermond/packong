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
	"github.com/pkg/errors"
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
	if werr(w, err, 500, "can't get width") {
		return
	}

	str = r.FormValue("height")
	height, err = strconv.ParseFloat(str, 64)
	if werr(w, err, 500, "can't get height") {
		return
	}

	str = r.FormValue("cutwidth")
	cutwidth, err = strconv.ParseFloat(str, 64)
	if werr(w, err, 500, "can't get cutwidth") {
		return
	}

	str = r.FormValue("mu")
	mu, err = strconv.ParseFloat(str, 64)
	if werr(w, err, 500, "can't get mu") {
		return
	}

	str = r.FormValue("ml")
	ml, err = strconv.ParseFloat(str, 64)
	if werr(w, err, 500, "can't get ml") {
		return
	}

	str = r.FormValue("pp")
	pp, err = strconv.ParseFloat(str, 64)
	if werr(w, err, 500, "can't get pp") {
		return
	}

	str = r.FormValue("pd")
	pd, err = strconv.ParseFloat(str, 64)
	if werr(w, err, 500, "can't get pd") {
		return
	}

	str = r.FormValue("topleftmargin")
	topleftmargin, err = strconv.ParseFloat(str, 64)
	if werr(w, err, 500, "can't get topleftmargin") {
		return
	}

	str = r.FormValue("tight")
	tight, err = strconv.ParseBool(str)
	if werr(w, err, 500, "can't get topleftmargin") {
		return
	}

	str = r.FormValue("plain")
	plain, err = strconv.ParseBool(str)
	if werr(w, err, 500, "can't get plain") {
		return
	}

	str = r.FormValue("showdim")
	showDim, err = strconv.ParseBool(str)
	if werr(w, err, 500, "can't get showdim") {
		return
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
		werr(w, err, 500, "packing error")
	}

	b, err := json.Marshal(rep)
	if err != nil {
		werr(w, err, 500, "json error")
	}
	io.Copy(w, bytes.NewReader(b))
}

func werr(w http.ResponseWriter, err error, code int, msg string) bool {
	err = errors.Cause(err)
	log.Printf("%v", err)
	http.Error(w, msg, code)
	return err != nil
}
