package packong

import (
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/innermond/packong/internal/svg"
	"github.com/innermond/pak"
)

const (
	CONTINUE_FIT_OP = iota
	ABORT_FIT_OP
)

// strategies used for packing boxes on mother box
var strategies = map[string]*pak.Base{
	"BestAreaFit":      &pak.Base{&pak.BestAreaFit{}},
	"BestLongSide":     &pak.Base{&pak.BestLongSide{}},
	"BestShortSide":    &pak.Base{&pak.BestShortSide{}},
	"BottomLeft":       &pak.Base{&pak.BottomLeft{}},
	"BestSimilarRatio": &pak.Base{&pak.BestSimilarRatio{}},
}

// Op describe a boxes packing operation
type Op struct {
	// dimensions's boxes
	dimensions []string
	// filename of a graphic file (svg) with boxes packed
	outname string
	// measurement unit: mm, cm
	unit string
	// mother box dimensions
	width, height float64
	// when true boxes area is surround exactly area boxes swarm
	tight bool
	// plain FALSE indicates svg output is as-inkscape
	plain bool
	// will rendered "wxh" dimensions pair on every box
	showDim bool
	// amount of expanding area's box in order to accomodate to loosing material
	// when a physical cut (that has real width which eats from box area) occurs
	cutwidth float64
	// point from where boxes are lay down
	topleftmargin float64

	// prices:
	// mu - material used, a price that reflects man's work
	// ml - material lost, a price regarding raw material - that's it it doesn't contains man's work
	// pp - perimeter price, a price connected with number of cuts needed for breaking big sheet to needed pieces
	// pd - price of moving to location
	mu, ml, pp, pd float64

	// it considers lost material as valuable as used material
	greedy bool
	// vendors are selling lengths of sheets measured by natural numbers
	vendorsellint bool

	// scale factors: k is for lenghts, k2 is for areas
	k, k2 float64

	// output for web
	outweb bool

	// outline, no fill
	outline bool
}

func NewOp(w, h float64, dd []string, u string) *Op {
	op := &Op{
		width:      w,
		height:     h,
		dimensions: dd,
		unit:       u,

		tight: true,
		plain: true,

		greedy:        false,
		vendorsellint: true,
	}

	op.k, op.k2 = op.kk()

	return op
}

func (op *Op) Topleft(tl float64) *Op {
	op.topleftmargin = tl
	return op
}

func (op *Op) WidthHeight(w, h float64) *Op {
	op.width = w
	op.height = h
	return op
}
func (op *Op) Dimensions(dd []string) *Op {
	op.dimensions = dd
	return op
}
func (op *Op) Tight(t bool) *Op {
	op.tight = t
	return op
}

func (op *Op) Appearance(yesno ...bool) *Op {
	switch len(yesno) {
	case 0:
		op.plain = true
		op.showDim = false

	case 1:
		op.plain = yesno[0]

	case 3:
		op.plain = yesno[0]
		op.showDim = yesno[1]
		op.outline = yesno[2]

	default:
		op.plain = yesno[0]
		op.showDim = yesno[1]
	}

	return op
}

func (op *Op) Cutwidth(cw float64) *Op {
	op.cutwidth = cw
	return op
}

func (op *Op) Outname(name string) *Op {
	op.outname = name
	return op
}

func (op *Op) Outweb(val bool) *Op {
	op.outweb = val
	return op
}

func (op *Op) Price(mu, ml, pp, pd float64) *Op {
	op.mu = mu
	op.ml = ml
	op.pp = pp
	op.pd = pd
	return op
}

func (op *Op) Greedy(mood bool) *Op {
	op.greedy = mood
	return op
}

func (op *Op) VendorSellInt(sell bool) *Op {
	op.vendorsellint = sell
	return op
}

func (op *Op) kk() (float64, float64) {
	k := 1000.0
	switch op.unit {
	case "cm":
		k = 100.0
	case "m":
		k = 1.0
	}
	k2 := k * k

	return k, k2
}

func (op *Op) NumStrategy() int {
	return len(strategies)
}

func (op *Op) Fit(pp [][]*pak.Box, deep bool) (*Report, []FitReader, error) {

	wins := map[string][]float64{}
	done := map[string][]*pak.Box{}
	remnants := map[string][]*pak.Box{}
	outputFn := map[string][]FitReader{}
	mx := sync.Mutex{}

	var wg sync.WaitGroup
	peek := 100
	//numAll:= len(pp) * len(strategies)
	var piece [][]*pak.Box
	lenpp := len(pp)
	i := 0
	j := peek
	if j > lenpp {
		j = lenpp
	}
pieced:
	for {
		piece = pp[i:j]
		wg.Add((j - i) * len(strategies))
		for pix, permutated := range piece {
			for strategyName, strategy := range strategies {
				sn := strategyName + ".perm." + strconv.Itoa(i+pix)
				s := strategy
				// unsorted
				go func() {
					bb := []*pak.Box{}
					for _, box := range permutated {
						bb = append(bb, &pak.Box{W: box.W, H: box.H, CanRotate: box.CanRotate})
					}
					mx.Lock()
					wins[sn], done[sn], remnants[sn], outputFn[sn] = op.matchboxes(sn, s, bb)
					defer mx.Unlock()
					defer wg.Done()
				}()
			}
		}
		wg.Wait()
		if j == lenpp {
			break pieced
		}
		i = j
		j += peek
		if j > lenpp {
			j = lenpp
		}
	}

	smallestLostArea, prevSmallestLostArea := math.MaxFloat32, math.MaxFloat32
	winingStrategyName := ""
	for sn, st := range wins {
		smallestLostArea = st[0]/op.k2 - st[2]/op.k2
		if smallestLostArea <= prevSmallestLostArea {
			prevSmallestLostArea = smallestLostArea
			winingStrategyName = sn
		}
	}

	best, ok := wins[winingStrategyName]
	if !ok {
		return nil, nil, errors.New("no wining strategy")
	}
	boxes, ok := remnants[winingStrategyName]
	if !ok {
		return nil, nil, errors.New("remnants error")
	}
	fitboxes, ok := done[winingStrategyName]
	if !ok {
		return nil, nil, errors.New("fitboxes error")
	}
	outFns, ok := outputFn[winingStrategyName]
	if !ok {
		return nil, nil, errors.New("outFns error")
	}
	usedArea, vendoredArea, vendoredLength, boxesArea, boxesPerim, numSheetsUsed := best[0], best[1], best[2], best[3], best[4], best[5]
	lostArea := usedArea - boxesArea
	if op.vendorsellint {
		lostArea = vendoredArea - boxesArea
	}
	procentArea := 0.0
	if usedArea > 0 {
		procentArea = boxesArea * 100 / usedArea
	}
	boxesArea = boxesArea / op.k2
	usedArea = usedArea / op.k2
	vendoredArea = vendoredArea / op.k2
	vendoredLength = vendoredLength / op.k
	lostArea = lostArea / op.k2
	boxesPerim = boxesPerim / op.k
	price := boxesArea*op.mu + lostArea*op.ml + boxesPerim*op.pp + op.pd
	if op.greedy {
		price = boxesArea*op.mu + lostArea*op.mu + boxesPerim*op.pp + op.pd
	}
	rep := &Report{
		WiningStrategyName: winingStrategyName,
		BoxesArea:          boxesArea,
		UsedArea:           usedArea,
		VendoredArea:       vendoredArea,
		VendoredLength:     vendoredLength,
		VendoredWidth:      op.width / op.k,
		LostArea:           lostArea,
		ProcentArea:        procentArea,
		BoxesPerim:         boxesPerim,
		Price:              price,
		UnfitLen:           len(boxes),
		UnfitCode:          pak.BoxCode(boxes),
		FitCode:            pak.BoxCode(fitboxes),
		NumSheetUsed:       numSheetsUsed,
	}

	return rep, outFns, nil
}

func (op *Op) BoxesFromString() (boxes []*pak.Box, err error) {
	for _, dd := range op.dimensions {
		d := strings.Split(dd, "x")
		if len(d) == 2 {
			d = append(d, "1", "1") // repeat 1 time
		} else if len(d) == 3 {
			d = append(d, "1") // can rotate
		}

		w, err := strconv.ParseFloat(d[0], 64)
		if err != nil {
			return nil, err
		} else if w <= 0 {
			err = fmt.Errorf("greater than zero condition; received %f", w)
			return nil, err
		}

		h, err := strconv.ParseFloat(d[1], 64)
		if err != nil {
			return nil, err
		} else if h <= 0 {
			err = fmt.Errorf("greater than zero condition; received %f", h)
			return nil, err
		}

		n, err := strconv.Atoi(d[2])
		if err != nil {
			return nil, err
		} else if n < 1 {
			err = fmt.Errorf("greater than zero condition; received %q", n)
			return nil, err
		} else if n > 50 {
			err = fmt.Errorf("lesser than peak condition; received %f", w)
			return nil, err
		}

		r, err := strconv.ParseBool(d[3])
		if err != nil {
			return nil, err
		}

		for n != 0 {
			var val = &pak.Box{W: w + op.cutwidth, H: h + op.cutwidth, CanRotate: r}
			boxes = append(boxes, val)
			n--
		}

		sort.Slice(boxes, func(i, j int) bool {
			return boxes[i].W*boxes[i].H > boxes[j].W*boxes[j].H
		})
	}
	return
}

type FitReader map[string]io.Reader

func (op *Op) matchboxes(strategyName string, strategy *pak.Base, boxes []*pak.Box) ([]float64, []*pak.Box, []*pak.Box, []FitReader) {

	var (
		lenboxes  int
		remaining []*pak.Box
		done      []*pak.Box
	)
	inx, usedArea, vendoredArea, vendoredLength, boxesArea, boxesPerim := 0, 0.0, 0.0, 0.0, 0.0, 0.0
	fnOutput := []FitReader{}

	lenboxes = len(boxes)

	for lenboxes > 0 {
		// shrink all aria
		op.width -= op.topleftmargin
		op.height -= op.topleftmargin
		bin := pak.NewBin(op.width, op.height, strategy)
		remaining = []*pak.Box{}
		maxx, maxy := 0.0, 0.0
		// partials metrics per cycle
		vendoredAreaForInx, vendoredLengthForInx := 0.0, 0.0
		// pack boxes into bin
		for _, box := range boxes {
			// cutwidth acts like a padding enlarging boxes
			if op.topleftmargin == 0.0 {
				// all boxes touching top or left edges will need a half expand
				if box.X == 0.0 && box.Y == 0.0 { // top left box
					box.W -= op.cutwidth / 2
					box.H -= op.cutwidth / 2
				} else if box.X == 0.0 && box.Y != 0.0 { // leftmost column
					box.W -= op.cutwidth / 2
					box.Y -= op.cutwidth / 2
				} else if box.Y == 0.0 && box.X != 0.0 { // topmost row
					box.H -= op.cutwidth / 2
					box.X -= op.cutwidth / 2
				} else if box.X*box.Y != 0.0 { // the other boxes
					box.X -= op.cutwidth / 2
					box.Y -= op.cutwidth / 2
				}
			} else {
				// no need to adjust W or H but X and Y
				box.X += op.topleftmargin
				box.Y += op.topleftmargin
			}
			if !bin.Insert(box) {
				remaining = append(remaining, box)
				// cannot insert skyp to next box
				continue
			}
			done = append(done, box)

			boxesArea += (box.W * box.H)
			boxesPerim += 2 * (box.W + box.H)

			if box.Y+box.H-op.topleftmargin > maxy {
				maxy = box.Y + box.H - op.topleftmargin
			}
			if box.X+box.W-op.topleftmargin > maxx {
				maxx = box.X + box.W - op.topleftmargin
			}
		}
		// enlarge aria back
		op.width += op.topleftmargin
		op.height += op.topleftmargin

		if op.tight {
			maxx = op.width
		} else {
			maxx = op.width
			maxy = op.height
		}

		vendoredAreaForInx = (maxx * maxy)
		usedArea += vendoredAreaForInx
		if op.vendorsellint {
			// vendors sells integers so convert the maxy into meters, find closest integer and then convert to mm
			vendoredAreaForInx = math.Ceil(maxy/op.k) * op.k * maxx
			vendoredArea += vendoredAreaForInx
		} else {
			vendoredArea = usedArea
		}
		vendoredLengthForInx = vendoredAreaForInx / op.width
		vendoredLength = vendoredArea / op.width

		inx++

		if len(remaining) == lenboxes {
			break
		}
		lenboxes = len(remaining)
		boxes = remaining[:]

		if op.outname != "" {
			// vendoredLength is a fraction associated with inx cycle from cummulative vendoredLength
			func(inx int, boxes []*pak.Box, vendoredLengthForInx float64) {
				fn := fmt.Sprintf("%s.%d.%s.svg", op.outname, inx, strategyName)

				var s string
				if op.outweb {
					s = svg.StartWeb(op.width, vendoredLengthForInx+op.topleftmargin, op.plain)
				} else {
					s = svg.Start(op.width, vendoredLengthForInx+op.topleftmargin, op.unit, op.plain)
				}
				si, err := svg.Out(boxes, op.cutwidth, op.topleftmargin, op.width, op.unit, op.plain, op.showDim, op.outline)
				if err != nil {
					return
				}
				s += svg.End(si)
				fnOutput = append(fnOutput, FitReader{fn: strings.NewReader(s)})
			}(inx, bin.Boxes[:], vendoredLengthForInx)
		}
	}
	return []float64{usedArea, vendoredArea, vendoredLength, boxesArea, boxesPerim, float64(inx)}, done, remaining, fnOutput
}

//go:generate json_snake_case -type=Report
type Report struct {
	WiningStrategyName string
	BoxesArea          float64
	UsedArea           float64
	VendoredArea       float64
	VendoredLength     float64
	VendoredWidth      float64
	LostArea           float64
	ProcentArea        float64
	BoxesPerim         float64
	Price              float64
	UnfitLen           int
	UnfitCode          string
	FitCode            string
	NumSheetUsed       float64
}
