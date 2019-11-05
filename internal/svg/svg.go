package svg

import (
	"fmt"
	"strings"
)

const (
	svgtop = `<?xml version="1.0"?>
<svg`
	svginitfmt = `%s width="%f%s" height="%f%s"`
	svgns      = `
     xmlns="http://www.w3.org/2000/svg"
     xmlns:xlink="http://www.w3.org/1999/xlink"`
	svgnsinkscape = `
   xmlns:sodipodi="http://sodipodi.sourceforge.net/DTD/sodipodi-0.dtd"
   xmlns:inkscape="http://www.inkscape.org/namespaces/inkscape"`
	vbfmt = `viewBox="%f %f %f %f"`

	emptyclose = "/>"
)

func Start(w float64, h float64, unit string, plain bool) string {
	s := fmt.Sprintf(svginitfmt, svgtop, w, unit, h, unit) + " " +
		fmt.Sprintf(vbfmt, 0.0, 0.0, w, h) + svgns
	if plain == false {
		s += svgnsinkscape
	}
	s += ">"
	return s
}

func StartWeb(w float64, h float64, plain bool) string {
	s := svgtop +
		" style=\"positon:absolute;width:100%;height:100%;\" preserveAspectRatio=\"xMidYMid meet\" " +
		fmt.Sprintf(vbfmt, 0.0, 0.0, w, h) + svgns
	if plain == false {
		s += svgnsinkscape
	}
	s += ">"
	s += Rect(0.0, 0.0, w, h, "stroke:gray;stroke-width:2;fill:none")
	return s
}

func End(s string) string {
	return s + "</svg>"
}

func GroupStart(ss ...string) string {
	gs := ""
	for _, s := range ss {
		gs += s + " "
	}
	gs = strings.TrimSpace(gs)
	return fmt.Sprintf("<g %s>", gs)
}

func GroupEnd(g string) string {
	return g + "</g>"
}

func Rect(x float64, y float64, w float64, h float64, s string) string {
	return fmt.Sprintf(`
<rect x="%f" y="%f" width="%f" height="%f" style="%s" />`, x, y, w, h, s)
}

func Text(x float64, y float64, transform, txt string, s string) string {
	return fmt.Sprintf(`
<text x="%f" y="%f" %s style="%s" >
%s
</text>`, x, y, transform, s, txt)
}
