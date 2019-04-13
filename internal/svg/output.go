package svg

import (
	"errors"
	"fmt"
	"math"

	"github.com/innermond/pak"
)

func aproximateHeightText(numchar int, w float64) float64 {
	wchar := w / float64(numchar+2)
	return math.Floor(1.5*wchar*100.0) / 100
}

func Out(blocks []*pak.Box, topleftmargin float64, plain bool, showDim bool) (string, error) {
	if len(blocks) == 0 {
		return "", errors.New("no blocks")
	}

	gb := GroupStart("id=\"blocks\"")
	if !plain {
		gb = GroupStart("id=\"blocks\"", "inkscape:label=\"blocks\"", "inkscape:groupmode=\"layer\"")
	}
	// first block
	blk := blocks[0]
	gb += Rect(blk.X,
		blk.Y,
		blk.W,
		blk.H,
		"fill:magenta;stroke:none",
	)

	for _, blk := range blocks[1:] {
		if blk != nil {
			// blocks on the top edge must be shortened on height by a expand = half cutwidth
			if blk.Y == topleftmargin {
				gb += Rect(blk.X,
					blk.Y,
					blk.W,
					blk.H,
					"fill:red;stroke:none",
				)
				continue
			}
			// blocks on the left edge must be shortened on width by a expand = half cutwidth
			if blk.X == topleftmargin {
				gb += Rect(blk.X,
					blk.Y,
					blk.W,
					blk.H,
					"fill:green;stroke:none",
				)
				continue
			}
			// blocks that do not touch any big box edges keeps their expanded dimensions
			gb += Rect(blk.X,
				blk.Y,
				blk.W,
				blk.H,
				"fill:#eee;stroke:none",
			)
		} else {
			return "", errors.New("unexpected unfit block")
		}
	}
	gb = GroupEnd(gb)

	gt := ""
	if showDim {
		gt = GroupStart("id=\"dimensions\"")
		if !plain {
			gt = GroupStart("id=\"dimensions\"", "inkscape:label=\"dimensions\"", "inkscape:groupmode=\"layer\"")
		}
		for _, blk := range blocks {
			if blk != nil {
				x := fmt.Sprintf("%.2fx%.2f", blk.W, blk.H)
				if blk.Rotated {
					x += "xR"
				}
				y := aproximateHeightText(len(x), blk.W)
				gt += Text(blk.X+blk.W/2, blk.Y+blk.H/2+y/3, // y/3 is totally empirical
					x, "text-anchor:middle;font-size:"+fmt.Sprintf("%.2f", y)+";fill:#000")
			} else {
				return "", errors.New("unexpected unfit block")
			}
		}
		gt = GroupEnd(gt)
	}
	return gb + gt, nil
}