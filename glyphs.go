//go:build !headless

package main

import (
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// The game fonts have no suit or music symbols, so ♠ ♥ ♦ ♣ ♫ are drawn as
// shapes (the same suit outlines as the card art) wherever they appear in text.

const specials = "♠♥♦♣♫☐☑"

type fill struct{ path vector.Path }

func heartPath(p *vector.Path, x, y, s float32, rot bool) {
	pt := func(px, py float32) (float32, float32) {
		if rot { // rotate 180° about (x, y)
			return 2*x - px, 2*y - py
		}
		return px, py
	}
	m := func(px, py float32) { a, b := pt(px, py); p.MoveTo(a, b) }
	c := func(a1, b1, a2, b2, a3, b3 float32) {
		x1, y1 := pt(a1, b1)
		x2, y2 := pt(a2, b2)
		x3, y3 := pt(a3, b3)
		p.CubicTo(x1, y1, x2, y2, x3, y3)
	}
	m(x, y+s*0.45)
	c(x-s*0.62, y+s*0.02, x-s*0.52, y-s*0.55, x, y-s*0.2)
	c(x+s*0.52, y-s*0.55, x+s*0.62, y+s*0.02, x, y+s*0.45)
	p.Close()
}

func circlePath(p *vector.Path, x, y, r float32) {
	p.MoveTo(x+r, y)
	p.Arc(x, y, r, 0, 2*math.Pi, vector.Clockwise)
	p.Close()
}

// suitPath builds a symbol centred at (x, y), size em tall.
func suitPath(r rune, x, y, size float32) *vector.Path {
	var p vector.Path
	s := size
	switch r {
	case '♥':
		heartPath(&p, x, y, s, false)
	case '♦':
		p.MoveTo(x, y-s*0.5)
		p.QuadTo(x+s*0.18, y-s*0.18, x+s*0.4, y)
		p.QuadTo(x+s*0.18, y+s*0.18, x, y+s*0.5)
		p.QuadTo(x-s*0.18, y+s*0.18, x-s*0.4, y)
		p.QuadTo(x-s*0.18, y-s*0.18, x, y-s*0.5)
		p.Close()
	case '♠':
		heartPath(&p, x, y-s*0.08, s*0.9, true)
		p.MoveTo(x, y+s*0.05)
		p.QuadTo(x-s*0.05, y+s*0.38, x-s*0.24, y+s*0.5)
		p.LineTo(x+s*0.24, y+s*0.5)
		p.QuadTo(x+s*0.05, y+s*0.38, x, y+s*0.05)
		p.Close()
	case '♣':
		rr := s * 0.205
		circlePath(&p, x, y-s*0.24, rr)
		circlePath(&p, x-s*0.22, y+s*0.06, rr)
		circlePath(&p, x+s*0.22, y+s*0.06, rr)
		circlePath(&p, x, y, s*0.12)
		p.MoveTo(x, y+s*0.02)
		p.QuadTo(x-s*0.04, y+s*0.38, x-s*0.22, y+s*0.5)
		p.LineTo(x+s*0.22, y+s*0.5)
		p.QuadTo(x+s*0.04, y+s*0.38, x, y+s*0.02)
		p.Close()
	case '♫': // two beamed eighth notes
		hx := s * 0.17
		for _, cx := range []float32{x - s*0.2, x + s*0.22} {
			cy := y + s*0.28
			if cx > x {
				cy = y + s*0.2
			}
			p.MoveTo(cx+hx, cy)
			p.Arc(cx, cy, hx, 0, 2*math.Pi, vector.Clockwise)
			p.Close()
			p.MoveTo(cx+hx-s*0.07, cy)
			p.LineTo(cx+hx-s*0.07, cy-s*0.62+(y+s*0.28-cy))
			p.LineTo(cx+hx, cy-s*0.62+(y+s*0.28-cy))
			p.LineTo(cx+hx, cy)
			p.Close()
		}
		x0, x1 := x-s*0.2+hx-s*0.07, x+s*0.22+hx
		p.MoveTo(x0, y-s*0.34)
		p.LineTo(x1, y-s*0.42)
		p.LineTo(x1, y-s*0.28)
		p.LineTo(x0, y-s*0.2)
		p.Close()
	}
	return &p
}

func glyphW(r rune, size float64) float64 {
	switch r {
	case '♫':
		return size * 0.85
	case '☐', '☑':
		return size * 0.95
	}
	return size * 0.86
}

// richWidth measures text that may contain symbol runes.
func richWidth(str string, f font) float64 {
	if !strings.ContainsAny(str, specials) {
		return measure(str, f)
	}
	w := 0.0
	for _, seg := range splitSpecial(str) {
		if seg.sym != 0 {
			w += glyphW(seg.sym, f.size)
		} else {
			w += measure(seg.s, f)
		}
	}
	return w
}

type tseg struct {
	s   string
	sym rune
}

func splitSpecial(str string) []tseg {
	var out []tseg
	var b strings.Builder
	for _, r := range str {
		if strings.ContainsRune(specials, r) {
			if b.Len() > 0 {
				out = append(out, tseg{s: b.String()})
				b.Reset()
			}
			out = append(out, tseg{sym: r})
			continue
		}
		b.WriteRune(r)
	}
	if b.Len() > 0 {
		out = append(out, tseg{s: b.String()})
	}
	return out
}

// rich draws text (baseline at y) with symbols drawn as shapes.
func (p pen) rich(str string, f font, x, y float64, clr color.Color, align int) {
	if !strings.ContainsAny(str, specials) {
		p.text(str, f, x, y, clr, align)
		return
	}
	w := richWidth(str, f)
	switch align {
	case alignC:
		x -= w / 2
	case alignR:
		x -= w
	}
	for _, seg := range splitSpecial(str) {
		if seg.sym == 0 {
			p.text(seg.s, f, x, y, clr, alignL)
			x += measure(seg.s, f)
			continue
		}
		gw := glyphW(seg.sym, f.size)
		p.symbol(seg.sym, x+gw/2, y-f.size*0.36, f.size*0.8, clr)
		x += gw
	}
}

func (p pen) symbol(r rune, cx, cy, size float64, clr color.Color) {
	if r == '☐' || r == '☑' {
		s := size * 0.8
		p.rrStroke(cx-s/2, cy-s/2, s, s, 2, math.Max(1.5, size*0.1), clr)
		if r == '☑' {
			var path vector.Path
			path.MoveTo(float32(cx-s*0.28), float32(cy))
			path.LineTo(float32(cx-s*0.06), float32(cy+s*0.24))
			path.LineTo(float32(cx+s*0.3), float32(cy-s*0.26))
			p.strokePath(&path, math.Max(1.8, size*0.13), clr, nil)
		}
		return
	}
	p.fillNZ(suitPath(r, float32(cx), float32(cy), float32(size)), clr, nil)
}

// fillNZ fills with the non-zero rule (needed for concave shapes).
func (p pen) fillNZ(path *vector.Path, clr color.Color, m *ebiten.GeoM) {
	vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)
	r, g, b, a := clr.RGBA()
	var t ebiten.GeoM
	if m != nil {
		t = *m
	}
	p.geo(&t)
	for i := range vs {
		x, y := t.Apply(float64(vs[i].DstX), float64(vs[i].DstY))
		vs[i].DstX, vs[i].DstY = float32(x), float32(y)
		vs[i].SrcX, vs[i].SrcY = 2, 2
		vs[i].ColorR, vs[i].ColorG, vs[i].ColorB, vs[i].ColorA = float32(r)/0xffff, float32(g)/0xffff, float32(b)/0xffff, float32(a)/0xffff
	}
	op := &ebiten.DrawTrianglesOptions{AntiAlias: true, FillRule: ebiten.FillRuleNonZero, ColorScaleMode: ebiten.ColorScaleModePremultipliedAlpha}
	p.dst.DrawTriangles(vs, is, whitePixel(), op)
}

// spaced draws letter-spaced text (CSS letter-spacing in em).
func (p pen) spaced(str string, f font, x, y, em float64, clr color.Color, align int) {
	w := spacedWidth(str, f, em)
	switch align {
	case alignC:
		x -= w / 2
	case alignR:
		x -= w
	}
	for _, r := range str {
		s := string(r)
		p.rich(s, f, x, y, clr, alignL)
		x += richWidth(s, f) + em*f.size
	}
}

func spacedWidth(str string, f font, em float64) float64 {
	w := 0.0
	for _, r := range str {
		w += richWidth(string(r), f) + em*f.size
	}
	return w - em*f.size
}

// wrap breaks text into lines no wider than maxW.
func wrap(str string, f font, maxW float64) []string {
	var lines []string
	for _, para := range strings.Split(str, "\n") {
		words := strings.Fields(para)
		line := ""
		for _, w := range words {
			try := w
			if line != "" {
				try = line + " " + w
			}
			if line != "" && richWidth(try, f) > maxW {
				lines = append(lines, line)
				line = w
			} else {
				line = try
			}
		}
		lines = append(lines, line)
	}
	return lines
}
