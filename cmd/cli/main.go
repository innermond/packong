package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/atotto/clipboard"
	"github.com/innermond/packong"
)

var (
	dimensions []string

	outname, unit, bigbox string
	wh                    []string
	width, height         float64

	tight bool

	plain, showDim, greedy, vendorsellint, deep bool

	cutwidth, topleftmargin float64

	mu, ml, pp, pd float64
	showOffer      bool
)

func param() {
	var err error

	flag.StringVar(&outname, "o", "", "name of the maching project")
	flag.StringVar(&unit, "u", "mm", "unit of measurements")

	flag.StringVar(&bigbox, "bb", "0x0", "dimensions as \"wxh\" in units for bigest box / mother surface")

	flag.BoolVar(&tight, "tight", false, "when true only aria used tighten by height is taken into account")
	flag.BoolVar(&plain, "inkscape", true, "when false will save svg as inkscape svg")
	flag.BoolVar(&showDim, "showdim", false, "generate a layer with dimensions \"wxh\" regarding each box")
	flag.BoolVar(&greedy, "greedy", false, "when calculating price material's area lost is considered at full working price")
	flag.BoolVar(&vendorsellint, "vendorsellint", true, "vendors sells an integer number of sheet length")
	flag.BoolVar(&deep, "deep", false, "calculate all boxes permutations")
	flag.BoolVar(&showOffer, "offer", false, "show a text representing offer")

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
		panic("need to specify dimensions for big box")
	}
	width, err = strconv.ParseFloat(wh[0], 64)
	if err != nil {
		panic("can't get width")
	}
	height, err = strconv.ParseFloat(wh[1], 64)
	if err != nil {
		panic("can't get height")
	}
	dimensions = flag.Args()

	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "inkscape":
			plain = false
		case "tight":
			tight = true
		case "showdim":
			showDim = true
		case "greedy":
			greedy = true
		case "deep":
			deep = true
		case "offer":
			showOffer = true
		}
	})
}

var selltext string = `Oferta pentru suprafetele
{{.Dimensions}} in {{.Unit}}
este de {{printf "%.2f" .Price}} euro + TVA.
Include productie, montaj, deplasare.`

type offer struct {
	*packong.Report
	Dimensions, Unit string
}

func main() {
	param()

	rep, outs, err := packong.NewOp(width, height, dimensions, unit).
		Outname(outname).
		Appearance(plain, showDim).
		Price(mu, ml, pp, pd).
		Greedy(greedy).
		VendorSellInt(vendorsellint).
		Fit(deep)

	if err != nil {
		panic(err)
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	if rep.UnfitLen > 0 {
		fmt.Fprintf(tw, "%s\t%d\n", "UnfitLen", rep.UnfitLen)
		fmt.Fprintf(tw, "%s\t%s\n", "UnfitCode", rep.UnfitCode)
	}

	pieces := strings.Join(dimensions, unit+" ")
	fmt.Fprintf(tw, "%s\t%s\n", "StragegyName", rep.WiningStrategyName)
	fmt.Fprintf(tw, "%s\t%s\n", "Pieces", pieces+unit)
	fmt.Fprintf(tw, "%s\t%.2f\n", "BoxesArea", rep.BoxesArea)
	fmt.Fprintf(tw, "%s\t%.2f\n", "UsedArea", rep.UsedArea)
	fmt.Fprintf(tw, "%s\t%.2f\n", "LostArea", rep.LostArea)
	fmt.Fprintf(tw, "%s\t%.2f\n", "VendoredArea", rep.VendoredArea)
	fmt.Fprintf(tw, "%s\t%.2f\n", "ProcentArea", rep.ProcentArea)
	fmt.Fprintf(tw, "%s\t%.2f\n", "NumSheetUsed", rep.NumSheetUsed)
	fmt.Fprintf(tw, "%s\t%.2f\n", "Price", rep.Price)
	if showOffer {
		tpl, err := template.New("offer").Parse(selltext)
		if err != nil {
			panic(err)
		}
		sell := offer{rep, pieces, unit}
		var bb bytes.Buffer
		if err := tpl.Execute(&bb, sell); err != nil {
			panic(err)
		}
		dotted := strings.Repeat("-", 30)
		offerTxt := bb.String()
		fmt.Fprintf(tw, "%s\n%s\n", dotted, offerTxt)
		clipboard.WriteAll(offerTxt)
	}
	tw.Flush()
	if len(outname) > 0 {
		errs := writeFiles(outs)
		if len(errs) > 0 {
			log.Println(errs)
		}
	}
}

func writeFiles(outs []packong.FitReader) (errs []error) {
	for _, out := range outs {
		for nm, r := range out {
			w, err := os.Create(nm)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			defer w.Close()
			_, err = io.Copy(w, r)
			if err != nil {
				errs = append(errs, err)
				continue
			}
		}
	}
	return
}
