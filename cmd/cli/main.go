package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/atotto/clipboard"
	"github.com/innermond/packong"
	"github.com/innermond/pak"
)

var (
	dimensions []string

	outname, unit string
	bigbox        bigboxes
	width, height float64
	fo            string

	tight bool

	plain, showDim, greedy, vendorsellint, deep bool

	cutwidth, topleftmargin float64

	mu, pp, pd, ph  float64
	ml              price
	showOffer, spor bool
)

func param() error {

	flag.StringVar(&outname, "o", "", "name of the maching project")
	flag.StringVar(&unit, "u", "mm", "unit of measurements")

	flag.Var(&bigbox, "bb", "dimensions as \"wxh\" in units for bigest box / mother surface")

	flag.BoolVar(&tight, "tight", false, "when true only aria used tighten by height is taken into account")
	flag.BoolVar(&plain, "inkscape", true, "when false will save svg as inkscape svg")
	flag.BoolVar(&showDim, "showdim", false, "generate a layer with dimensions \"wxh\" regarding each box")
	flag.BoolVar(&greedy, "greedy", false, "when calculating price material's area lost is considered at full working price")
	flag.BoolVar(&vendorsellint, "vendorsellint", true, "vendors sells an integer number of sheet length")
	flag.BoolVar(&deep, "deep", false, "calculate all boxes permutations")
	flag.BoolVar(&showOffer, "offer", false, "show a text representing offer")
	flag.BoolVar(&spor, "spor", false, "spor")
	flag.StringVar(&fo, "fo", "", "template offer filename")

	flag.Float64Var(&mu, "mu", 15.0, "used material price per 1 square meter")
	flag.Var(&ml, "ml", "lost material price per 1 square meter")
	flag.Float64Var(&pp, "pp", 0.25, "perimeter price per 1 linear meter; used for evaluating cuts price")
	flag.Float64Var(&pd, "pd", 10, "travel price to location")
	flag.Float64Var(&ph, "ph", 3.5, "man power price")
	flag.Float64Var(&cutwidth, "cutwidth", 0.0, "the with of material that is lost due to a cut")
	flag.Float64Var(&topleftmargin, "margin", 0.0, "offset from top left margin")

	flag.Parse()

	if len(ml) == 0 {
		ml = price{5.0}
	}

	if fo != "" {
		bb, err := ioutil.ReadFile(fo)
		if err != nil {
			return err
		}
		selltext = string(bb)
	}

	dimensions = flag.Args()
	if len(dimensions) == 0 {
		return errors.New("dimensions required")
	}

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
		case "spor":
			spor = true
		}
	})
	return nil
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
	var (
		rep  *packong.Report
		outs []packong.FitReader
		err  error
	)

	err = param()
	if err != nil {
		log.Fatal(err)
	}

	bx := bigbox[0]
	width, height = bx[0], bx[1]

	op := packong.NewOp(width, height, dimensions, unit).
		Outname(outname).
		Appearance(plain, showDim).
		Price(mu, ml[0], pp, pd).
		Greedy(greedy).
		VendorSellInt(vendorsellint)
	// if the cut can eat half of its width along cutline
	// we compensate expanding boxes with an entire cut width
	boxes, err := op.BoxesFromString()
	if err != nil {
		log.Fatal(err)
	}
	pp := [][]*pak.Box{boxes}
	if deep {
		pp = packong.Permutations(boxes)
		// take approval from user
		fmt.Printf("%d combinations. Can take a much much longer time. Continue?\n", len(pp))
		var (
			yn string
			r  *bufio.Reader = bufio.NewReader(os.Stdin)
		)
	approve:
		for {
			fmt.Println("Enter y to continue or a n to abort")
			yn, err = r.ReadString('\n')
			yn = strings.TrimRight(yn, "\n")
			if err != nil {
				continue
			}

			switch yn {
			case "y":
				break approve
			case "n":
				fmt.Println("user aborted packing operation")
				os.Exit(0)
			}
		}
	}
	rep, outs, err = op.Fit(pp, deep)
	if err != nil {
		log.Fatal(err)
	}
	writeReport(rep, ml[0], ph, pd, spor, showOffer, selltext, unit)

	for rep.UnfitLen > 0 && len(bigbox) > 1 {
		bigbox = bigbox[1:]
		width, height = bigbox[0][0], bigbox[0][1]
		op.Dimensions(strings.Split(strings.TrimSpace(rep.UnfitCode), ","))
		op.WidthHeight(width, height)
		boxes, err := op.BoxesFromString()
		if err != nil {
			log.Fatal(err)
		}
		pp := [][]*pak.Box{boxes}
		rep, outs, err = op.Fit(pp, deep)
		if err != nil {
			log.Fatal(err)
		}
		if len(ml) > 1 {
			ml = ml[1:]
		}
		writeReport(rep, ml[0], ph, pd, spor, showOffer, selltext, unit)
	}

	if len(outname) > 0 {
		errs := writeFiles(outs)
		if len(errs) > 0 {
			log.Println(errs)
		}
	}

}

func writeReport(rep *packong.Report, ml, ph, pd float64, spor, showOffer bool, selltext, unit string) {
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	if rep.UnfitLen > 0 {
		fmt.Fprintf(tw, "%s\t%d\n", "UnfitLen", rep.UnfitLen)
		fmt.Fprintf(tw, "%s\t%s\n", "UnfitCode", rep.UnfitCode)
	}
	pieces := strings.TrimSpace(rep.FitCode)
	fmt.Fprintf(tw, "%s\t%s\n", "StragegyName", rep.WiningStrategyName)
	fmt.Fprintf(tw, "%s\t%s\n", "Pieces", pieces+unit)
	fmt.Fprintf(tw, "%s\t%.2f\n", "BoxesArea", rep.BoxesArea)
	fmt.Fprintf(tw, "%s\t%.2f\n", "UsedArea", rep.UsedArea)
	fmt.Fprintf(tw, "%s\t%.2f\n", "LostArea", rep.LostArea)
	fmt.Fprintf(tw, "%s\t%.2f\n", "VendoredArea", rep.VendoredArea)
	fmt.Fprintf(tw, "%s\t%.2f\n", "VendoredLength", rep.VendoredLength)
	fmt.Fprintf(tw, "%s\t%.2f\n", "VendoredWidth", rep.VendoredWidth)
	fmt.Fprintf(tw, "%s\t%.2f\n", "ProcentArea", rep.ProcentArea)
	fmt.Fprintf(tw, "%s\t%.2f\n", "NumSheetUsed", rep.NumSheetUsed)
	fmt.Fprintf(tw, "%s\t%.2f\n", "Price", rep.Price)
	fmt.Fprintf(tw, "%s\t%.2f\n", "Materials cost", rep.VendoredArea*ml)
	fmt.Fprintf(tw, "%s\t%.2f\n", "Man cost", rep.BoxesArea*ph)
	fmt.Fprintf(tw, "%s\t%.2f\n", "Travel cost", pd)
	if spor {
		fmt.Fprintf(tw, "%s\t%.2f\n", "Spor", rep.Price-(rep.VendoredArea*ml+rep.BoxesArea*ph+pd))
	}
	if showOffer {
		tpl, err := template.New("offer").Parse(selltext)
		if err != nil {
			log.Fatal(err)
		}
		sell := offer{rep, pieces, unit}
		var bb bytes.Buffer
		if err := tpl.Execute(&bb, sell); err != nil {
			log.Fatal(err)
		}
		dotted := strings.Repeat("-", 30)
		offerTxt := bb.String()
		fmt.Fprintf(tw, "%s\n%s\n", dotted, offerTxt)
		err = clipboard.WriteAll(offerTxt)
		if err != nil {
			log.Println(err)
		}
	}
	tw.Flush()
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
