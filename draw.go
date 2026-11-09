//go:build !headless

package main

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Everything is laid out in logical pixels (roughly CSS pixels at a 1440x900
// window) and drawn at the screen's real resolution, so art and text stay sharp.

type fontKind int

const (
	fSlab  fontKind = iota // Oswald (headings)
	fBody                  // Rubik
	fItal                  // Rubik Italic
	fDispI                 // Pacifico (titles)
)

type font struct {
	k    fontKind
	size float64
	wght float64
}

func slab(size float64) font         { return font{fSlab, size, 800} }
func dispI(size float64) font        { return font{fDispI, size, 800} }
func body(size, wght float64) font   { return font{fBody, size, wght} }
func italic(size, wght float64) font { return font{fItal, size, wght} }

var wghtTag = text.MustParseTag("wght")

type faceKey struct {
	k    fontKind
	px   float64
	wght float64
}

var faceCache = map[faceKey]*text.GoTextFace{}

func face(f font, s float64) *text.GoTextFace {
	key := faceKey{f.k, math.Round(f.size*s*4) / 4, f.wght}
	if fc, ok := faceCache[key]; ok {
		return fc
	}
	src := art.slab
	switch f.k {
	case fBody:
		src = art.bitter
	case fItal:
		src = art.ital
	case fDispI:
		src = art.dispI
	}
	fc := &text.GoTextFace{Source: src, Size: key.px}
	fc.SetVariation(wghtTag, float32(f.wght))
	faceCache[key] = fc
	return fc
}

// pen draws onto dst in logical coordinates; (ox, oy) is a logical offset.
type pen struct {
	dst    *ebiten.Image
	s      float64
	ox, oy float64
}

func (p pen) at(ox, oy float64) pen { return pen{p.dst, p.s, p.ox + ox, p.oy + oy} }

func (p pen) X(x float64) float32 { return float32((x + p.ox) * p.s) }
func (p pen) Y(y float64) float32 { return float32((y + p.oy) * p.s) }

// geo maps logical coordinates to the screen (apply after local transforms).
func (p pen) geo(g *ebiten.GeoM) {
	g.Translate(p.ox, p.oy)
	g.Scale(p.s, p.s)
}

func measure(str string, f font) float64 {
	w, _ := text.Measure(str, face(f, 1), 0)
	return w
}

const (
	alignL = iota
	alignC
	alignR
)

// text draws str with its baseline at y.
func (p pen) text(str string, f font, x, y float64, clr color.Color, align int) {
	fc := face(f, p.s)
	op := &text.DrawOptions{}
	op.GeoM.Translate(math.Round(float64(p.X(x))), math.Round(float64(p.Y(y))-fc.Metrics().HAscent))
	switch align {
	case alignC:
		op.PrimaryAlign = text.AlignCenter
	case alignR:
		op.PrimaryAlign = text.AlignEnd
	}
	op.ColorScale.ScaleWithColor(clr)
	op.Filter = ebiten.FilterLinear
	text.Draw(p.dst, str, fc, op)
}

// textShadow draws a hard drop shadow first (CSS text-shadow 0 dy 0 c).
func (p pen) textShadow(str string, f font, x, y float64, clr, sh color.Color, dy float64, align int) {
	p.text(str, f, x, y+dy, sh, align)
	p.text(str, f, x, y, clr, align)
}

var whiteSub *ebiten.Image

func whitePixel() *ebiten.Image {
	if whiteSub == nil {
		whiteSub = art.white.SubImage(image.Rect(1, 1, 3, 3)).(*ebiten.Image)
	}
	return whiteSub
}

// fill draws a path given in logical coordinates (optionally transformed by m first).
func (p pen) fillPath(path *vector.Path, clr color.Color, m *ebiten.GeoM) {
	vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)
	p.drawVerts(vs, is, clr, m)
}

func (p pen) strokePath(path *vector.Path, w float64, clr color.Color, m *ebiten.GeoM) {
	vs, is := path.AppendVerticesAndIndicesForStroke(nil, nil, &vector.StrokeOptions{Width: float32(w), LineJoin: vector.LineJoinRound, LineCap: vector.LineCapRound})
	p.drawVerts(vs, is, clr, m)
}

func (p pen) drawVerts(vs []ebiten.Vertex, is []uint16, clr color.Color, m *ebiten.GeoM) {
	r, g, b, a := clr.RGBA()
	if a == 0 {
		return
	}
	var t ebiten.GeoM
	if m != nil {
		t = *m
	}
	p.geo(&t)
	for i := range vs {
		x, y := t.Apply(float64(vs[i].DstX), float64(vs[i].DstY))
		vs[i].DstX, vs[i].DstY = float32(x), float32(y)
		vs[i].SrcX, vs[i].SrcY = 2, 2
		vs[i].ColorR = float32(r) / 0xffff
		vs[i].ColorG = float32(g) / 0xffff
		vs[i].ColorB = float32(b) / 0xffff
		vs[i].ColorA = float32(a) / 0xffff
	}
	op := &ebiten.DrawTrianglesOptions{AntiAlias: true, ColorScaleMode: ebiten.ColorScaleModePremultipliedAlpha}
	p.dst.DrawTriangles(vs, is, whitePixel(), op)
}

func rrPath(x, y, w, h, r float64) *vector.Path {
	r = math.Min(r, math.Min(w, h)/2)
	var path vector.Path
	X, Y, W, H, R := float32(x), float32(y), float32(w), float32(h), float32(r)
	path.MoveTo(X+R, Y)
	path.ArcTo(X+W, Y, X+W, Y+H, R)
	path.ArcTo(X+W, Y+H, X, Y+H, R)
	path.ArcTo(X, Y+H, X, Y, R)
	path.ArcTo(X, Y, X+W, Y, R)
	path.Close()
	return &path
}

func (p pen) rr(x, y, w, h, r float64, clr color.Color) { p.fillPath(rrPath(x, y, w, h, r), clr, nil) }

func (p pen) rrStroke(x, y, w, h, r, lw float64, clr color.Color) {
	// stroke centred on the inside edge, like a CSS border
	p.strokePath(rrPath(x+lw/2, y+lw/2, w-lw, h-lw, math.Max(0, r-lw/2)), lw, clr, nil)
}

// box draws a CSS-style bordered box with a solid offset "ledge" shadow and an
// optional soft shadow.
func (p pen) box(x, y, w, h, r, border float64, fill, edge color.Color, ledge float64, soft float64) {
	if soft > 0 {
		for i := 4; i >= 1; i-- {
			f := float64(i)
			p.rr(x-f*soft/8, y+soft/3+f*soft/6, w+f*soft/4, h+f*soft/6, r+f*2, color.NRGBA{0, 0, 0, uint8(14)})
		}
	}
	if ledge > 0 {
		p.rr(x, y+ledge, w, h, r, edge)
	}
	p.rr(x, y, w, h, r, edge)
	p.rr(x+border, y+border, w-2*border, h-2*border, math.Max(0, r-border), fill)
}

func (p pen) rect(x, y, w, h float64, clr color.Color) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(w/2, h/2)
	op.GeoM.Translate(x, y)
	p.geo(&op.GeoM)
	op.ColorScale.ScaleWithColor(clr)
	p.dst.DrawImage(whitePixel(), op)
}

func (p pen) circle(cx, cy, r float64, clr color.Color) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-32, -32)
	op.GeoM.Scale(r/31.5, r/31.5)
	op.GeoM.Translate(cx, cy)
	p.geo(&op.GeoM)
	op.ColorScale.ScaleWithColor(clr)
	op.Filter = ebiten.FilterLinear
	p.dst.DrawImage(art.dot, op)
}

// image draws im centred at (cx, cy), w logical pixels wide, rotated by deg.
func (p pen) image(im *ebiten.Image, cx, cy, w, h, deg float64, cs ebiten.ColorScale) {
	b := im.Bounds()
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Translate(-float64(b.Dx())/2, -float64(b.Dy())/2)
	op.GeoM.Scale(w/float64(b.Dx()), h/float64(b.Dy()))
	op.GeoM.Rotate(deg * math.Pi / 180)
	op.GeoM.Translate(cx, cy)
	p.geo(&op.GeoM)
	op.ColorScale = cs
	p.dst.DrawImage(im, op)
}

func (p pen) imageTL(im *ebiten.Image, x, y, w, h float64, cs ebiten.ColorScale) {
	p.image(im, x+w/2, y+h/2, w, h, 0, cs)
}

func alpha(a float64) ebiten.ColorScale {
	var c ebiten.ColorScale
	c.ScaleAlpha(float32(a))
	return c
}

func colA(c color.NRGBA, a float64) color.NRGBA { c.A = u8(float64(c.A) / 255 * a); return c }

// The rooftop palette: brick red, night navy, cream paper, charcoal and string-light amber.
var (
	cInk    = hexc("1e1916", 1)
	cCream  = hexc("f6efe2", 1)
	cBarn   = hexc("b23a2a", 1) // brick
	cBarn2  = hexc("86281c", 1)
	cDenim  = hexc("1f2a4a", 1) // navy
	cPaper  = hexc("fffaf2", 1)
	cBrown  = hexc("2b2420", 1) // charcoal
	cMaroon = hexc("4a1a12", 1)
	cMuted  = hexc("7a6a5a", 1)
	cSub    = hexc("4a3c34", 1)
	cSub2   = hexc("5e5048", 1)
	cGold   = hexc("ffd862", 1)
	cWhite  = color.NRGBA{255, 255, 255, 255}
)
