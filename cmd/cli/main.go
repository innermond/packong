package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/innermond/2pak/internal/svg"
	"github.com/innermond/pak"
)

var (
	dimensions []string

	outname, unit, bigbox string
	wh                    []string
	width, height         float64

	modeReportAria    string
	tight, supertight bool

	report, output, plain, showDim bool

	cutwidth, topleftmargin float64
	expandtocutwidth        bool

	mu, ml, pp, pd float64
)

func param() {
	var err error

	flag.StringVar(&outname, "o", "fit", "name of the maching project")
	flag.StringVar(&unit, "u", "mm", "unit of measurements")

	flag.StringVar(&bigbox, "bb", "0x0", "dimensions as \"wxh\" in units for bigest box / mother surface")

	flag.BoolVar(&report, "r", true, "match report")
	flag.BoolVar(&output, "f", false, "outputing files representing matching")
	flag.BoolVar(&tight, "tight", false, "when true only aria used tighten by height is taken into account")
	flag.BoolVar(&supertight, "supertight", false, "when true only aria used tighten bu height and width is taken into account")
	flag.BoolVar(&plain, "inkscape", true, "when false will save svg as inkscape svg")
	flag.BoolVar(&showDim, "showdim", false, "generate a layer with dimensions \"wxh\" regarding each box")
	flag.Float64Var(&mu, "mu", 15.0, "used material price per 1 square meter")
	flag.Float64Var(&ml, "ml", 5.0, "lost material price per 1 square meter")
	flag.Float64Var(&pp, "pp", 0.25, "perimeter price per 1 linear meter; used for evaluating cuts price")
	flag.Float64Var(&pd, "pd", 10, "travel price to location")
	flag.Float64Var(&cutwidth, "cutwidth", 0.0, "the with of material that is lost due to a cut")
	flag.Float64Var(&topleftmargin, "margin", 0.0, "offset from top left margin")

	flag.Parse()

	wh = strings.Split(bigbox, "x")
	switch len(wh) {
	case 1:
		wh = append(wh, wh[0])
	case 0:
		panicli("need to specify dimensions for big box")
	}
	width, err = strconv.ParseFloat(wh[0], 64)
	if err != nil {
		panicli("can't get width")
	}
	height, err = strconv.ParseFloat(wh[1], 64)
	if err != nil {
		panicli("can't get height")
	}
	dimensions = flag.Args()

	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "inkscape":
			plain = false
		case "tight":
			tight = true
			modeReportAria = "tight"
		case "showdim":
			showDim = true
		case "supertight":
			supertight = true
			modeReportAria = "supertight"
		}
	})
}

func main() {
	param()

	strategies := map[string]*pak.Base{
		"BestAreaFit":      &pak.Base{&pak.BestAreaFit{}},
		"BestLongSide":     &pak.Base{&pak.BestLongSide{}},
		"BestShortSide":    &pak.Base{&pak.BestShortSide{}},
		"BottomLeft":       &pak.Base{&pak.BottomLeft{}},
		"BestSimilarRatio": &pak.Base{&pak.BestSimilarRatio{}},
	}
	wins := map[string][]float64{}
	remnants := map[string][]*pak.Box{}
	outputFn := map[string]func(){}
	mx := sync.Mutex{}

	var wg sync.WaitGroup
	wg.Add(len(strategies))
	for strategyName, strategy := range strategies {
		strategyName := strategyName
		strategy := strategy
		go func() {
			mx.Lock()
			wins[strategyName], remnants[strategyName], outputFn[strategyName] = fit(width, height, strategyName, strategy)
			defer mx.Unlock()
			defer wg.Done()
		}()
	}
	wg.Wait()

	k := 1000.0
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
		panicli("no wining strategy")
	}
	boxes, ok := remnants[winingStrategyName]
	if !ok {
		panicli("remnants error")
	}
	outFn, ok := outputFn[winingStrategyName]
	if !ok {
		panicli("outFn error")
	}
	outFn()
	usedArea, boxesArea, boxesPerim, numSheetsUsed := best[0], best[1], best[2], best[3]
	lostArea := usedArea - boxesArea
	procentArea := boxesArea * 100 / usedArea
	boxesArea = boxesArea / k2
	usedArea = usedArea / k2
	lostArea = lostArea / k2
	boxesPerim = boxesPerim / k
	price := boxesArea*mu + lostArea*ml + boxesPerim*pp + pd
	fmt.Printf("strategy %s boxes aria %.2f used aria %.2f lost aria %.2f procent %.2f%% perim %.2f price %.2f remaining boxes %d %s sheets used %.0f\n",
		winingStrategyName, boxesArea, usedArea, lostArea, procentArea, boxesPerim, price, len(boxes), pak.BoxCode(boxes), numSheetsUsed)
}

func panicli(msg interface{}) {
	var code int

	switch msg.(type) {
	case string:
		code = 0
	case error:
		code = 1
	}
	fmt.Fprintln(os.Stdout, msg)
	os.Exit(code)
}

func fit(width float64, height float64, strategyName string, strategy *pak.Base) ([]float64, []*pak.Box, func()) {

	var (
		boxes     []*pak.Box
		lenboxes  int
		remaining []*pak.Box
	)
	inx, usedArea, boxesArea, boxesPerim := 0, 0.0, 0.0, 0.0
	fnOutput := func() {}

	// if the cut can eat half of its width along cutline
	// we compensate expanding boxes with an entire cut width
	boxes = boxesFromString(dimensions, cutwidth)
	lenboxes = len(boxes)

	for lenboxes > 0 {
		bin := pak.NewBin(width, height, strategy)
		remaining = []*pak.Box{}
		maxx, maxy := 0.0, 0.0
		// shrink all aria
		width -= topleftmargin
		height -= topleftmargin
		// pack boxes into bin
		for _, box := range boxes {
			if !bin.Insert(box) {
				remaining = append(remaining, box)
				// cannot insert skyp to next box
				continue
			}

			if topleftmargin == 0.0 {
				// all boxes touching top or left edges will need a half expand
				if box.X == 0.0 && box.Y == 0.0 { // top left box
					box.W -= cutwidth / 2
					box.H -= cutwidth / 2
				} else if box.X == 0.0 && box.Y != 0.0 { // leftmost column
					box.W -= cutwidth / 2
					box.Y -= cutwidth / 2
				} else if box.Y == 0.0 && box.X != 0.0 { // topmost row
					box.H -= cutwidth / 2
					box.X -= cutwidth / 2
				} else if box.X*box.Y != 0.0 { // the other boxes
					box.X -= cutwidth / 2
					box.Y -= cutwidth / 2
				}
			} else {
				// no need to adjust W or H but X and Y
				box.X += topleftmargin
				box.Y += topleftmargin
			}

			boxesArea += (box.W * box.H)
			boxesPerim += 2 * (box.W + box.H)

			if box.Y+box.H-topleftmargin > maxy {
				maxy = box.Y + box.H - topleftmargin
			}
			if box.X+box.W-topleftmargin > maxx {
				maxx = box.X + box.W - topleftmargin
			}
		}
		// enlarge aria back
		width += topleftmargin
		height += topleftmargin

		if modeReportAria == "tight" {
			maxx = width
		} else if modeReportAria != "supertight" {
			maxx = width
			maxy = height
		}
		usedArea += (maxx * maxy)

		inx++

		if len(remaining) == lenboxes {
			break
		}
		lenboxes = len(remaining)
		boxes = remaining[:]

		if output {
			fnOutput = func() {
				fn := fmt.Sprintf("%s.%d.%s.svg", outname, inx, strategyName)

				f, err := os.Create(fn)
				if err != nil {
					panicli("cannot create file")
				}

				s := svg.Start(width, height, unit, plain)
				si, err := outsvg(bin.Boxes, topleftmargin, plain, showDim)
				if err != nil {
					f.Close()
					os.Remove(fn)
				} else {
					s += svg.End(si)

					_, err = f.WriteString(s)
					if err != nil {
						panicli(err)
					}
					f.Close()
				}
			}
		}
	}
	return []float64{usedArea, boxesArea, boxesPerim, float64(inx)}, remaining, fnOutput
}
