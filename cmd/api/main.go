package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/innermond/packong"
	"github.com/pkg/errors"
)

func main() {
	s := &http.Server{
		ReadHeaderTimeout: 20 * time.Second,
		Addr:              ":2222",
		Handler:           fitBoxes,
	}
	err := s.ListenAndServe()
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

	err = r.ParseForm()
	if werr(w, err, 500, "can't parse form") {
		return
	}

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
	if str == "" {
		cutwidth = 0.0
	} else {
		cutwidth, err = strconv.ParseFloat(str, 64)
		if werr(w, err, 500, "can't get cutwidth") {
			return
		}
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
	if str == "" {
		topleftmargin = 0.0
	} else {
		topleftmargin, err = strconv.ParseFloat(str, 64)
		if werr(w, err, 500, "can't get topleftmargin") {
			return
		}
	}

	str = r.FormValue("tight")
	if str == "" {
		tight = true
	} else {
		tight, err = strconv.ParseBool(str)
		if werr(w, err, 500, "can't get tight") {
			return
		}
	}

	str = r.FormValue("plain")
	if str == "" {
		plain = true
	} else {
		plain, err = strconv.ParseBool(str)
		if werr(w, err, 500, "can't get plain") {
			return
		}
	}

	str = r.FormValue("showdim")
	if str == "" {
		plain = true
	} else {
		showDim, err = strconv.ParseBool(str)
		if werr(w, err, 500, "can't get showdim") {
			return
		}
	}

	outname = r.FormValue("outname")

	unit = r.FormValue("unit")
	if unit == "" {
		unit = "mm"
	}

	dimensions = strings.Fields(r.FormValue("dimensions"))
	if len(dimensions) == 0 {
		werr(w, errors.New("dimensions required"), 400, "dimensions required")
		return
	}

	// second parameters outs will give svg
	rep, outs, err := packong.NewOp(width, height, dimensions, unit).
		Outname(outname).
		Tight(tight).
		Topleft(topleftmargin).
		Cutwidth(cutwidth).
		Appearance(plain, showDim).
		Price(mu, ml, pp, pd).
		Fit()

	if err != nil {
		werr(w, err, 500, "packing error")
		return
	}

	var (
		svgs map[string]string
		errs []error
	)
	if len(outname) > 0 {
		svgs, errs = writeSvg(outs)
		if len(errs) > 0 {
			werr(w, errs[0], 500, "json error")
			return
		}
	}
	out := struct {
		Rep  *packong.Report
		Svgs map[string]string
	}{
		rep,
		svgs,
	}
	b, err := json.Marshal(out)
	if err != nil {
		werr(w, err, 500, "json error")
		return
	}
	io.Copy(w, bytes.NewReader(b))
}

func writeSvg(outs []packong.FitReader) (svgs map[string]string, errs []error) {
	var (
		b   []byte
		err error
	)

	svgs = map[string]string{}
	for _, out := range outs {
		for nm, r := range out {
			b, err = ioutil.ReadAll(r)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			svgs[nm] = string(b)
		}
	}
	return
}

func werr(w http.ResponseWriter, err error, code int, msg string) bool {
	if err == nil {
		return false
	}

	err = errors.Cause(err)
	log.Printf("%v", err)
	http.Error(w, msg, code)
	return true
}
