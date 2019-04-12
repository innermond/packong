package main

import (
	"sort"
	"strconv"
	"strings"

	"github.com/innermond/pak"
)

func boxesFromString(dimensions []string, extra float64) (boxes []*pak.Box) {
	for _, dd := range dimensions {
		d := strings.Split(dd, "x")
		if len(d) == 2 {
			d = append(d, "1", "1") // repeat 1 time
		} else if len(d) == 3 {
			d = append(d, "1") // can rotate
		}

		w, err := strconv.ParseFloat(d[0], 64)
		if err != nil {
			panic(err)
		}

		h, err := strconv.ParseFloat(d[1], 64)
		if err != nil {
			panic(err)
		}

		n, err := strconv.Atoi(d[2])
		if err != nil {
			panic(err)
		}

		r, err := strconv.ParseBool(d[3])
		if err != nil {
			panic(err)
		}

		for n != 0 {
			boxes = append(boxes, &pak.Box{W: w + extra, H: h + extra, CanRotate: r})
			n--
		}
		// sort descending by area
		sort.Slice(boxes, func(i, j int) bool {
			return boxes[i].W*boxes[i].H > boxes[j].W*boxes[j].H
		})
	}
	return
}
