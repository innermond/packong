package 2pack

type FitOp struct {
	dimensions []string

	outname, unit, bigbox string
	wh                    []string
	width, height         float64

	modeReportAria string
	tight          bool

	output, plain, showDim bool

	cutwidth, topleftmargin float64

	mu, ml, pp, pd float64
}
