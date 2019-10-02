package svg

import (
	"errors"
	"fmt"
	"math"

	"github.com/innermond/pak"
)

func aproximateHeightText(numchar int, w float64) float64 {
	wchar := w / float64(numchar)
	return math.Floor(1.5*wchar*100.0) / 100
}

var (
	strokeStyle = "stroke: gray;stroke-width:2;fill:none"
)

func style(fill string, outline bool) string {
	if outline {
		return strokeStyle
	}
	return fill
}

func Out(blocks []*pak.Box, topleftmargin float64, widthSvg float64, plain bool, showDim bool, outline bool) (string, error) {
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
		style("fill:magenta;stroke:none", outline),
	)

	for _, blk := range blocks[1:] {
		if blk != nil {
			// blocks on the top edge must be shortened on height by a expand = half cutwidth
			if blk.Y == topleftmargin {
				gb += Rect(blk.X,
					blk.Y,
					blk.W,
					blk.H,
					style("fill:red;stroke:none", outline),
				)
				continue
			}
			// blocks on the left edge must be shortened on width by a expand = half cutwidth
			if blk.X == topleftmargin {
				gb += Rect(blk.X,
					blk.Y,
					blk.W,
					blk.H,
					style("fill:green;stroke:none", outline),
				)
				continue
			}
			// blocks that do not touch any big box edges keeps their expanded dimensions
			gb += Rect(blk.X,
				blk.Y,
				blk.W,
				blk.H,
				style("fill:#eee;stroke:none", outline),
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
				xt := blk.X + blk.W/2
				yt := blk.Y + blk.H/2
				rotation := ""
				lendim := blk.W
				lenx := float64(len(x))
				if blk.H > blk.W {
					rotation = fmt.Sprintf(" transform=\"rotate(90, %.2f,%.2f)\" ", xt, yt)
					lendim = blk.H
				}
				// assume height of a letter si 2 than its width
				// assume 16px represents 100%
				y := (2.0 * math.Floor(lendim/lenx) / 16)
				if y > 2.5 {
					y = 2.5 // assume height of 250% is universally readable
				}
				y *= 100
				gt += Text(xt, yt, rotation,
					x, "text-anchor:middle;font-size:"+fmt.Sprintf("%.2f%%", y)+";fill:#000")
			} else {
				return "", errors.New("unexpected unfit block")
			}
		}
		gt = GroupEnd(gt)
	}
	return gb + gt, nil
}
