package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/innermond/packong"
	"github.com/innermond/pak"
	"github.com/pkg/errors"
)

func fitBoxes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	w.Header().Set("X-Content-Type-Options", "sniff")
	w.Header().Set("Content-Type", "application/json")

	// parameters
	var (
		err error

		dimensions []string

		unit          string
		width, height float64

		tight, plain, showDim bool

		cutwidth, topleftmargin float64

		mu, ml, pp, pd float64
	)

	// get input data
	var resp ResponseData
	{
		dec := json.NewDecoder(r.Body)
		err = dec.Decode(&resp)
		if werr(w, err, 500, "can't decode json") {
			return
		}
		defer r.Body.Close()
	}

	// unique name
	var outname string
	{
		rand.Seed(time.Now().UnixNano())
		chars := []byte("abcdefghijklmnopqrstuvwxyz")
		lenchars := len(chars)
		var b strings.Builder
		for i := 0; i < lenchars; i++ {
			b.WriteByte(chars[rand.Intn(lenchars)])
		}
		outname = b.String()
	}

	// assign input data
	width = resp.Width
	height = resp.Height
	tight = true
	topleftmargin = resp.Topleftmargin
	cutwidth = resp.Cutwidth
	plain = resp.Plain
	showDim = true
	mu = resp.Mu
	ml = resp.Ml
	pp = resp.Pp
	pd = resp.Pd
	unit = resp.Unit
	if unit == "" {
		unit = "mm"
	}

	dimensions = resp.Dimensions
	if len(dimensions) == 0 {
		werr(w, errors.New("dimensions required"), 400, "dimensions required")
		return
	}

	op := packong.NewOp(width, height, dimensions, unit).
		Outname(outname).
		Outweb(true).
		Tight(tight).
		Topleft(topleftmargin).
		Cutwidth(cutwidth).
		Appearance(plain, showDim, true).
		Price(mu, ml, pp, pd)
	boxes, err := op.BoxesFromString()
	if err != nil {
		werr(w, err, 500, "couldn't figure out dimensions; invalid dimensions")
		return
	}

	rep, outs, err := op.Fit([][]*pak.Box{boxes}, false)
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
			werr(w, errs[0], 500, "error preparing svg vizual")
			return
		}
	}
	repJson := packong.ReportJSON((*rep))
	out := struct {
		Rep  packong.ReportJSON `json:"rep,omitempty"`
		Svgs map[string]string  `json:"svgs,omitempty"`
	}{
		repJson,
		svgs,
	}
	b, err := json.Marshal(out)
	if err != nil {
		werr(w, err, 500, "json error")
		return
	}

	if verbose {
		log.Println(string(b))
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
