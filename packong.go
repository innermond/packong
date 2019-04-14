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

var strategies = map[string]*pak.Base{
	"BestAreaFit":      &pak.Base{&pak.BestAreaFit{}},
	"BestLongSide":     &pak.Base{&pak.BestLongSide{}},
	"BestShortSide":    &pak.Base{&pak.BestShortSide{}},
	"BottomLeft":       &pak.Base{&pak.BottomLeft{}},
	"BestSimilarRatio": &pak.Base{&pak.BestSimilarRatio{}},
}

type Op struct {
	dimensions []string

	outname, unit string
	width, height float64

	tight bool

	plain, showDim bool

	cutwidth, topleftmargin float64

	mu, ml, pp, pd float64
}

func NewOp(w, h float64, dd []string, u string) *Op {
	return &Op{
		width:      w,
		height:     h,
		dimensions: dd,
		unit:       u,

		tight: true,
		plain: true,
	}
}

func (op *Op) Topleft(tl float64) *Op {
	op.topleftmargin = tl
	return op
}

func (op *Op) Appearance(yesno ...bool) *Op {
	switch len(yesno) {
	case 0:
		op.plain = true
		op.showDim = false

	case 1:
		op.plain = yesno[0]

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

func (op *Op) Price(mu, ml, pp, pd float64) *Op {
	op.mu = mu
	op.ml = ml
	op.pp = pp
	op.pd = pd
	return op
}

func (op *Op) Fit() (*Report, []FitReader, error) {

	wins := map[string][]float64{}
	remnants := map[string][]*pak.Box{}
	outputFn := map[string][]FitReader{}
	mx := sync.Mutex{}

	var wg sync.WaitGroup
	wg.Add(len(strategies))

	for strategyName, strategy := range strategies {
		strategyName := strategyName
		strategy := strategy
		go func() {
			mx.Lock()
			wins[strategyName], remnants[strategyName], outputFn[strategyName] = op.matchboxes(strategyName, strategy)
			defer mx.Unlock()
			defer wg.Done()
		}()
	}
	wg.Wait()

	k := 1000.0
	switch op.unit {
	case "cm":
		k = 100.0
	case "m":
		k = 1.0
	}
	k2 := k * k

	smallestLostArea, prevSmallestLostArea := math.MaxFloat32, math.MaxFloat32
	winingStrategyName := ""
	for sn, st := range wins {
		smallestLostArea = st[0]/k2 - st[1]/k2
		fmt.Printf("%s lost area %.2f\n", sn, smallestLostArea)
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
	outFns, ok := outputFn[winingStrategyName]
	if !ok {
		return nil, nil, errors.New("outFns error")
	}
	usedArea, boxesArea, boxesPerim, numSheetsUsed := best[0], best[1], best[2], best[3]
	lostArea := usedArea - boxesArea
	procentArea := boxesArea * 100 / usedArea
	boxesArea = boxesArea / k2
	usedArea = usedArea / k2
	lostArea = lostArea / k2
	boxesPerim = boxesPerim / k
	price := boxesArea*op.mu + lostArea*op.ml + boxesPerim*op.pp + op.pd
	rep := &Report{
		WiningStrategyName: winingStrategyName,
		BoxesArea:          boxesArea,
		UsedArea:           usedArea,
		LostArea:           lostArea,
		ProcentArea:        procentArea,
		BoxesPerim:         boxesPerim,
		Price:              price,
		UnfitLen:           len(boxes),
		UnfitCode:          pak.BoxCode(boxes),
		NumSheetUsed:       numSheetsUsed,
	}

	return rep, outFns, nil
}

func (op *Op) boxesFromString(extra float64) (boxes []*pak.Box) {
	for _, dd := range op.dimensions {
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

type FitReader map[string]io.Reader

func (op *Op) matchboxes(strategyName string, strategy *pak.Base) ([]float64, []*pak.Box, []FitReader) {

	var (
		boxes     []*pak.Box
		lenboxes  int
		remaining []*pak.Box
	)
	inx, usedArea, boxesArea, boxesPerim := 0, 0.0, 0.0, 0.0
	fnOutput := []FitReader{}

	// if the cut can eat half of its width along cutline
	// we compensate expanding boxes with an entire cut width
	boxes = op.boxesFromString(op.cutwidth)
	lenboxes = len(boxes)

	for lenboxes > 0 {
		// shrink all aria
		op.width -= op.topleftmargin
		op.height -= op.topleftmargin
		bin := pak.NewBin(op.width, op.height, strategy)
		remaining = []*pak.Box{}
		maxx, maxy := 0.0, 0.0
		// pack boxes into bin
		for _, box := range boxes {
			if !bin.Insert(box) {
				remaining = append(remaining, box)
				// cannot insert skyp to next box
				continue
			}

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
		usedArea += (maxx * maxy)

		inx++

		if len(remaining) == lenboxes {
			break
		}
		lenboxes = len(remaining)
		boxes = remaining[:]

		if op.outname != "" {
			func(inx int, boxes []*pak.Box) {
				fn := fmt.Sprintf("%s.%d.%s.svg", op.outname, inx, strategyName)

				s := svg.Start(op.width, op.height, op.unit, op.plain)
				si, err := svg.Out(boxes, op.topleftmargin, op.plain, op.showDim)
				if err != nil {
					return
				}
				s += svg.End(si)
				fnOutput = append(fnOutput, FitReader{fn: strings.NewReader(s)})
			}(inx, bin.Boxes[:])
		}
	}
	return []float64{usedArea, boxesArea, boxesPerim, float64(inx)}, remaining, fnOutput
}

type Report struct {
	WiningStrategyName string
	BoxesArea          float64
	UsedArea           float64
	LostArea           float64
	ProcentArea        float64
	BoxesPerim         float64
	Price              float64
	UnfitLen           int
	UnfitCode          string
	NumSheetUsed       float64
}
