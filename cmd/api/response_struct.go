package main

type ResponseData struct {
	// dimensions's boxes
	Dimensions []string `json:"dimensions"`
	// filename of a graphic file (svg) with boxes packed
	Outname string `json:"outname"`
	// measurement unit: mm, cm
	Unit string `json:"unit"`
	// mother box dimensions
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	// when true boxes area is surround exactly area boxes swarm
	Tight bool `json:"tight"`
	// plain FALSE indicates svg output is as-inkscape
	Plain bool `json:"plain"`
	// will rendered "wxh" dimensions pair on every box
	ShowDim bool `json:"showdim"`
	// amount of expanding area's box in order to accomodate to loosing material
	// when a physical cut (that has real width which eats from box area) occurs
	Cutwidth float64 `json:"cutwidth"`
	// point from where boxes are lay down
	Topleftmargin float64 `json:"topleftmargin"`

	// prices:
	// mu - material used, a price that reflects man's work
	// ml - material lost, a price regarding raw material - that's it it doesn't contains man's work
	// pp - perimeter price, a price connected with number of cuts needed for breaking big sheet to needed pieces
	// pd - move on the spot price
	Mu float64 `json:"mu"`
	Ml float64 `json:"ml"`
	Pp float64 `json:"pp"`
	Pd float64 `json:"pd"`

	// it considers lost material as valuable as used material
	Greedy bool `json:"greedy"`
	// vendors are selling lengths of sheets measured by natural numbers
	Vendorsellint bool `json:"vendorsellint"`
}
