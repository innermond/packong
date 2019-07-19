package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type bigboxes [][2]float64

func (bb *bigboxes) String() string {
	return fmt.Sprintf("%v", *bb)
}

func (bb *bigboxes) Set(value string) error {
	var wh []string
	for _, bbox := range strings.Split(value, ",") {
		wh = strings.Split(bbox, "x")

		switch len(wh) {
		case 0:
			return errors.New("need to specify dimensions for big box")
		case 1:
			wh = append(wh, wh[0])
		}

		w, err := strconv.ParseFloat(wh[0], 64)
		if err != nil {
			return errors.New("can't get width")
		}

		h, err := strconv.ParseFloat(wh[1], 64)
		if err != nil {
			return errors.New("can't get height")
		}

		*bb = append(*bb, [2]float64{w, h})
	}

	return nil
}

func (bb *bigboxes) Dim() (out string) {

	for _, bbox := range *bb {
		out += fmt.Sprintf(",%vx%v", bbox[0], bbox[1])
	}

	if len(out) > 0 {
		out = out[1:]
	}
	return
}

type price []float64

func (p *price) String() string {
	return fmt.Sprintf("%v", *p)
}

func (p *price) Set(value string) error {
	for _, e := range strings.Split(value, ",") {
		val, err := strconv.ParseFloat(e, 64)
		if err != nil {
			return errors.New("price value not a number")
		}

		*p = append(*p, val)
	}

	return nil
}
