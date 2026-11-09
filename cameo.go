//go:build !headless

// The rooftop regulars' portraits, drawn in Go at whatever size they're needed.
// The little canvas below reads like the HTML canvas the card art was drawn
// with, so the portrait code matches the cards' hand.
package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

/* ---------------- a little canvas ---------------- */

type cstate struct {
	m          ebiten.GeoM
	fill, strk color.RGBA
	lw         float64
}

type canvas2d struct {
	dst              *ebiten.Image
	st               cstate
	stack            []cstate
	path             *vector.Path
	cx, cy, sx, sy   float64 // current point and subpath start (device space)
	hasCur, needMove bool
}

func newCanvas(dst *ebiten.Image) *canvas2d {
	return &canvas2d{dst: dst, st: cstate{fill: color.RGBA{0, 0, 0, 255}, strk: color.RGBA{0, 0, 0, 255}, lw: 1}, path: &vector.Path{}}
}

func (c *canvas2d) save()    { c.stack = append(c.stack, c.st) }
func (c *canvas2d) restore() { c.st = c.stack[len(c.stack)-1]; c.stack = c.stack[:len(c.stack)-1] }
func (c *canvas2d) translate(x, y float64) {
	var m ebiten.GeoM
	m.Translate(x, y)
	m.Concat(c.st.m)
	c.st.m = m
}
func (c *canvas2d) rotate(a float64) {
	var m ebiten.GeoM
	m.Rotate(a)
	m.Concat(c.st.m)
	c.st.m = m
}
func (c *canvas2d) scale(x, y float64) {
	var m ebiten.GeoM
	m.Scale(x, y)
	m.Concat(c.st.m)
	c.st.m = m
}
func (c *canvas2d) unit() float64 {
	return math.Hypot(c.st.m.Element(0, 0), c.st.m.Element(1, 0))
}
func (c *canvas2d) pt(x, y float64) (float64, float64) { return c.st.m.Apply(x, y) }

func (c *canvas2d) beginPath() { c.path = &vector.Path{}; c.hasCur, c.needMove = false, false }

func (c *canvas2d) moveTo(x, y float64) {
	px, py := c.pt(x, y)
	c.path.MoveTo(float32(px), float32(py))
	c.cx, c.cy, c.sx, c.sy = px, py, px, py
	c.hasCur, c.needMove = true, false
}

func (c *canvas2d) ensure(px, py float64) {
	if !c.hasCur {
		c.path.MoveTo(float32(px), float32(py))
		c.cx, c.cy, c.sx, c.sy = px, py, px, py
		c.hasCur = true
		return
	}
	if c.needMove {
		c.path.MoveTo(float32(c.sx), float32(c.sy))
		c.cx, c.cy = c.sx, c.sy
		c.needMove = false
	}
}

func (c *canvas2d) lineTo(x, y float64) {
	px, py := c.pt(x, y)
	c.ensure(px, py)
	c.path.LineTo(float32(px), float32(py))
	c.cx, c.cy = px, py
}

func (c *canvas2d) quadTo(x1, y1, x, y float64) {
	ax, ay := c.pt(x1, y1)
	px, py := c.pt(x, y)
	c.ensure(ax, ay)
	c.path.QuadTo(float32(ax), float32(ay), float32(px), float32(py))
	c.cx, c.cy = px, py
}

func (c *canvas2d) bezTo(x1, y1, x2, y2, x, y float64) {
	ax, ay := c.pt(x1, y1)
	bx, by := c.pt(x2, y2)
	px, py := c.pt(x, y)
	c.ensure(ax, ay)
	c.path.CubicTo(float32(ax), float32(ay), float32(bx), float32(by), float32(px), float32(py))
	c.cx, c.cy = px, py
}

func (c *canvas2d) closePath() {
	if c.hasCur {
		c.path.Close()
		c.cx, c.cy = c.sx, c.sy
		c.needMove = true
	}
}

// ellipse follows the canvas rules: a line joins the current point to the arc's start.
func (c *canvas2d) ellipse(x, y, rx, ry, rot, a0, a1 float64, ccw bool) {
	const tau = 2 * math.Pi
	var sweep float64
	if !ccw {
		if a1-a0 >= tau {
			sweep = tau
		} else {
			sweep = math.Mod(a1-a0, tau)
			if sweep < 0 {
				sweep += tau
			}
		}
	} else {
		if a0-a1 >= tau {
			sweep = -tau
		} else {
			s := math.Mod(a0-a1, tau)
			if s < 0 {
				s += tau
			}
			sweep = -s
		}
	}
	at := func(a float64) (float64, float64) {
		ex, ey := rx*math.Cos(a), ry*math.Sin(a)
		return c.pt(x+ex*math.Cos(rot)-ey*math.Sin(rot), y+ex*math.Sin(rot)+ey*math.Cos(rot))
	}
	n := int(math.Max(8, math.Ceil(math.Abs(sweep)*math.Max(rx, ry)*c.unit()/2.5)))
	px, py := at(a0)
	if c.hasCur && !c.needMove {
		c.path.LineTo(float32(px), float32(py))
	} else {
		c.path.MoveTo(float32(px), float32(py))
		c.sx, c.sy = px, py
		c.hasCur, c.needMove = true, false
	}
	for i := 1; i <= n; i++ {
		qx, qy := at(a0 + sweep*float64(i)/float64(n))
		c.path.LineTo(float32(qx), float32(qy))
		c.cx, c.cy = qx, qy
	}
}

func (c *canvas2d) arc(x, y, r, a0, a1 float64) { c.ellipse(x, y, r, r, 0, a0, a1, false) }
func (c *canvas2d) circle(x, y, r float64)      { c.arc(x, y, r, 0, 2*math.Pi) }

func (c *canvas2d) rrect(x, y, w, h, r float64) {
	c.moveTo(x+r, y)
	c.lineTo(x+w-r, y)
	c.ellipse(x+w-r, y+r, r, r, 0, -math.Pi/2, 0, false)
	c.lineTo(x+w, y+h-r)
	c.ellipse(x+w-r, y+h-r, r, r, 0, 0, math.Pi/2, false)
	c.lineTo(x+r, y+h)
	c.ellipse(x+r, y+h-r, r, r, 0, math.Pi/2, math.Pi, false)
	c.lineTo(x, y+r)
	c.ellipse(x+r, y+r, r, r, 0, math.Pi, 3*math.Pi/2, false)
	c.closePath()
}

func (c *canvas2d) rect(x, y, w, h float64) {
	c.moveTo(x, y)
	c.lineTo(x+w, y)
	c.lineTo(x+w, y+h)
	c.lineTo(x, y+h)
	c.closePath()
}

func (c *canvas2d) fill()   { cfill(c.dst, c.path, c.st.fill) }
func (c *canvas2d) stroke() { cstroke(c.dst, c.path, c.st.lw*c.unit(), c.st.strk) }
func (c *canvas2d) fillRect(x, y, w, h float64) {
	p := c.path
	c.beginPath()
	c.rect(x, y, w, h)
	c.fill()
	c.path = p
}
func (c *canvas2d) strokeRect(x, y, w, h float64) {
	p := c.path
	c.beginPath()
	c.rect(x, y, w, h)
	c.stroke()
	c.path = p
}

func cfill(dst *ebiten.Image, p *vector.Path, c color.RGBA) {
	vs, is := p.AppendVerticesAndIndicesForFilling(nil, nil)
	paint(dst, vs, is, c)
}

func cstroke(dst *ebiten.Image, p *vector.Path, width float64, c color.RGBA) {
	vs, is := p.AppendVerticesAndIndicesForStroke(nil, nil, &vector.StrokeOptions{Width: float32(width), LineJoin: vector.LineJoinRound, LineCap: vector.LineCapRound})
	paint(dst, vs, is, c)
}

func paint(dst *ebiten.Image, vs []ebiten.Vertex, is []uint16, c color.RGBA) {
	for i := range vs {
		vs[i].SrcX, vs[i].SrcY = 2, 2
		vs[i].ColorR, vs[i].ColorG, vs[i].ColorB, vs[i].ColorA = float32(c.R)/255, float32(c.G)/255, float32(c.B)/255, float32(c.A)/255
	}
	dst.DrawTriangles(vs, is, whitePixel(), &ebiten.DrawTrianglesOptions{AntiAlias: true, FillRule: ebiten.FillRuleNonZero})
}

func rgbaS(r, g, b uint8, a float64) color.RGBA {
	return color.RGBA{r, g, b, uint8(math.Round(a * 255))}
}

/* ---------------- a rooftop regular, head and shoulders ---------------- */

var (
	pInk   = rgb("#2b1d14")
	pWhite = rgb("#f7f3ea")
	pGold  = rgb("#c9a24c")
)

func orC(c, d color.RGBA) color.RGBA {
	if c.A == 0 {
		return d
	}
	return c
}

// shoulderX is how far the shoulders reach at height y (the outline of the
// bust from (22,-64) curving down to (80,0)).
func shoulderX(y float64) float64 {
	if y <= -64 {
		return 22
	}
	bx := func(t float64) (float64, float64) {
		u := 1 - t
		return u*u*u*22 + 3*u*u*t*54 + 3*u*t*t*78 + t*t*t*80, u*u*u*-64 + 3*u*u*t*-60 + 3*u*t*t*-42
	}
	lo, hi := 0.0, 1.0
	for i := 0; i < 30; i++ {
		m := (lo + hi) / 2
		if _, yy := bx(m); yy < y {
			lo = m
		} else {
			hi = m
		}
	}
	x, _ := bx(lo)
	return x
}

func shoulders(c *canvas2d) {
	c.beginPath()
	c.moveTo(-80, 0)
	c.bezTo(-78, -42, -54, -60, -22, -64)
	c.lineTo(22, -64)
	c.bezTo(54, -60, 78, -42, 80, 0)
	c.closePath()
}

// bust draws the head and shoulders, origin at the bottom centre.
func bust(c *canvas2d, o Look) {
	const tau = 2 * math.Pi
	skin, hair := o.Skin, o.Hair
	c.save()
	ink := func() { c.st.strk, c.st.lw = pInk, 2.6 }
	ink()
	hy := -118.0

	// hair behind the head
	c.st.fill = hair
	switch o.HairStyle {
	case "bun":
		c.beginPath()
		c.circle(0, hy-36, 15)
		c.fill()
		c.stroke()
	case "waves":
		c.beginPath()
		c.moveTo(-30, hy-10)
		c.bezTo(-44, hy+20, -40, hy+52, -26, hy+60)
		c.lineTo(26, hy+60)
		c.bezTo(40, hy+52, 44, hy+20, 30, hy-10)
		c.closePath()
		c.fill()
		c.stroke()
	case "curls":
		for _, p := range [][3]float64{{-30, hy + 6, 11}, {30, hy + 6, 11}, {-33, hy + 24, 10}, {33, hy + 24, 10}, {-27, hy + 40, 9}, {27, hy + 40, 9}} {
			c.beginPath()
			c.circle(p[0], p[1], p[2])
			c.fill()
			c.stroke()
		}
	case "bob":
		c.beginPath()
		c.moveTo(-31, hy-8)
		c.bezTo(-38, hy+14, -36, hy+28, -30, hy+32)
		c.lineTo(30, hy+32)
		c.bezTo(36, hy+28, 38, hy+14, 31, hy-8)
		c.closePath()
		c.fill()
		c.stroke()
	}

	// the top they're wearing
	top := o.Coat
	switch o.Top {
	case "stripes", "chef":
		top = pWhite
		if o.Top == "chef" {
			top = rgb("#fbfaf6")
		}
	}
	shoulders(c)
	c.st.fill = top
	c.fill()
	if o.Top == "stripes" {
		c.st.fill = o.Stripe
		for y := -58.0; y < 2; y += 13 {
			y0, y1 := y, math.Min(y+6, 0)
			x0, x1 := shoulderX(y0), shoulderX(y1)
			c.beginPath()
			c.moveTo(-x0, y0)
			c.lineTo(x0, y0)
			c.lineTo(x1, y1)
			c.lineTo(-x1, y1)
			c.closePath()
			c.fill()
		}
	}
	shoulders(c)
	c.stroke()

	// neck
	c.st.fill = skin
	c.beginPath()
	c.moveTo(-11, -90)
	c.lineTo(-12, -61)
	c.lineTo(12, -61)
	c.lineTo(11, -90)
	c.closePath()
	c.fill()
	c.st.fill = rgbaS(120, 60, 40, .18)
	c.beginPath()
	c.moveTo(-11, -88)
	c.quadTo(0, -80, 11, -88)
	c.lineTo(11, -84)
	c.quadTo(0, -76, -11, -84)
	c.closePath()
	c.fill()

	switch o.Top {
	case "stripes": // boat neck
		c.st.fill = top
		c.beginPath()
		c.moveTo(-24, -64)
		c.quadTo(0, -54, 24, -64)
		c.lineTo(22, -64)
		c.quadTo(0, -57, -22, -64)
		c.closePath()
		c.fill()
		c.beginPath()
		c.moveTo(-23, -64)
		c.quadTo(0, -55, 23, -64)
		c.stroke()
	case "blouse":
		c.st.fill = o.Accent
		for _, sx := range []float64{-1, 1} { // collar points
			c.beginPath()
			c.moveTo(sx*2, -62)
			c.lineTo(sx*20, -66)
			c.lineTo(sx*16, -50)
			c.closePath()
			c.fill()
			c.stroke()
		}
		c.st.strk, c.st.lw = shade(o.Coat, .7), 1.4
		c.beginPath()
		c.moveTo(0, -56)
		c.lineTo(0, 0)
		c.stroke()
		ink()
	case "turtleneck":
		c.st.fill = o.Coat
		c.beginPath()
		c.rrect(-15, -82, 30, 22, 6)
		c.fill()
		c.stroke()
		c.st.strk, c.st.lw = shade(o.Coat, .7), 1.2
		for x := -10.0; x <= 10; x += 5 {
			c.beginPath()
			c.moveTo(x, -80)
			c.lineTo(x, -62)
			c.stroke()
		}
		ink()
	case "suit":
		c.st.fill = pWhite
		c.beginPath()
		c.moveTo(-18, -64)
		c.lineTo(0, -14)
		c.lineTo(18, -64)
		c.closePath()
		c.fill()
		c.stroke()
		c.st.fill = o.Dark
		for _, sx := range []float64{-1, 1} { // lapels
			c.beginPath()
			c.moveTo(sx*20, -64)
			c.lineTo(sx*3, -14)
			c.lineTo(sx*30, -36)
			c.lineTo(sx*26, -52)
			c.closePath()
			c.fill()
			c.stroke()
		}
		c.st.fill = pWhite
		for _, sx := range []float64{-1, 1} { // shirt collar
			c.beginPath()
			c.moveTo(sx*1, -62)
			c.lineTo(sx*13, -66)
			c.lineTo(sx*10, -54)
			c.closePath()
			c.fill()
			c.stroke()
		}
		c.st.fill = pGold
		c.beginPath()
		c.circle(-24, -24, 2.4)
		c.fill()
	case "cardigan":
		c.st.fill = o.Accent
		c.beginPath()
		c.moveTo(-18, -64)
		c.lineTo(0, -26)
		c.lineTo(18, -64)
		c.closePath()
		c.fill()
		c.stroke()
		c.st.strk, c.st.lw = o.Dark, 3
		c.beginPath()
		c.moveTo(-18, -63)
		c.lineTo(0, -25)
		c.lineTo(18, -63)
		c.stroke()
		ink()
		c.st.fill = o.Dark
		for _, y := range []float64{-18, -6} {
			c.beginPath()
			c.circle(0, y, 3)
			c.fill()
		}
	case "chef":
		c.st.fill = rgb("#d8d4cc")
		for _, y := range []float64{-46, -30, -14} {
			for _, x := range []float64{-14, 14} {
				c.beginPath()
				c.circle(x, y, 3.2)
				c.fill()
				c.stroke()
			}
		}
		c.st.strk, c.st.lw = rgbaS(0, 0, 0, .2), 1.4
		c.beginPath()
		c.moveTo(-6, -60)
		c.quadTo(10, -30, 30, 0)
		c.stroke()
		ink()
	}

	switch o.Neck {
	case "scarf":
		c.st.fill = o.NeckColor
		c.beginPath()
		c.moveTo(-17, -78)
		c.quadTo(0, -66, 17, -78)
		c.lineTo(19, -66)
		c.quadTo(0, -54, -19, -66)
		c.closePath()
		c.fill()
		c.stroke()
		c.beginPath() // the knot and the tails
		c.moveTo(8, -64)
		c.quadTo(20, -46, 14, -26)
		c.lineTo(24, -30)
		c.quadTo(26, -48, 14, -62)
		c.closePath()
		c.fill()
		c.stroke()
		c.beginPath()
		c.ellipse(9, -63, 6, 5, 0, 0, tau, false)
		c.fill()
		c.stroke()
		c.st.fill = rgbaS(255, 255, 255, .45)
		for _, p := range [][2]float64{{-8, -70}, {2, -66}, {12, -71}, {18, -40}} {
			c.beginPath()
			c.circle(p[0], p[1], 1.8)
			c.fill()
		}
	case "pearls":
		c.st.fill = rgb("#fffaf0")
		c.st.lw = 1.2
		for i := 0; i < 11; i++ {
			a := math.Pi * (0.12 + float64(i)*0.076)
			c.beginPath()
			c.circle(math.Cos(a)*22, -70+math.Sin(a)*16, 3)
			c.fill()
			c.stroke()
		}
		ink()
	case "bowtie":
		c.st.fill = o.NeckColor
		c.beginPath()
		c.moveTo(0, -60)
		c.lineTo(-13, -67)
		c.lineTo(-13, -52)
		c.closePath()
		c.moveTo(0, -60)
		c.lineTo(13, -67)
		c.lineTo(13, -52)
		c.closePath()
		c.fill()
		c.stroke()
		c.beginPath()
		c.rrect(-3.5, -63.5, 7, 7, 2)
		c.fill()
		c.stroke()
	case "tie":
		c.st.fill = o.NeckColor
		c.beginPath()
		c.moveTo(-5, -61)
		c.lineTo(5, -61)
		c.lineTo(3, -55)
		c.lineTo(8, -20)
		c.lineTo(0, -12)
		c.lineTo(-8, -20)
		c.lineTo(-3, -55)
		c.closePath()
		c.fill()
		c.stroke()
	case "kerchief":
		c.st.fill = o.NeckColor
		c.beginPath()
		c.moveTo(-16, -66)
		c.quadTo(0, -58, 16, -66)
		c.lineTo(0, -44)
		c.closePath()
		c.fill()
		c.stroke()
	}

	// ears and head
	c.st.fill = skin
	c.beginPath()
	c.ellipse(-27, hy+2, 5, 8, 0, 0, tau, false)
	c.ellipse(27, hy+2, 5, 8, 0, 0, tau, false)
	c.fill()
	c.stroke()
	if o.Earrings {
		c.st.fill = pGold
		for _, sx := range []float64{-1, 1} {
			c.beginPath()
			c.circle(sx*28, hy+12, 3)
			c.fill()
		}
	}
	c.st.fill = skin
	c.beginPath()
	c.ellipse(0, hy, 27, 33, 0, 0, tau, false)
	c.fill()
	c.stroke()

	// hair in front
	c.st.fill = hair
	switch o.HairStyle {
	case "short", "curly":
		c.beginPath()
		c.ellipse(0, hy-14, 28.5, 22, 0, math.Pi, 0, false)
		c.quadTo(14, hy-22, -4, hy-18)
		c.quadTo(-18, hy-14, -28.5, hy-6)
		c.closePath()
		c.fill()
		c.stroke()
		if o.HairStyle == "curly" {
			for i := 0; i < 7; i++ {
				a := math.Pi*1.05 + float64(i)*math.Pi*0.15
				c.beginPath()
				c.circle(math.Cos(a)*24, hy-14+math.Sin(a)*20, 7)
				c.fill()
				c.stroke()
			}
		}
	case "side":
		c.beginPath()
		c.moveTo(-28.5, hy-2)
		c.bezTo(-34, hy-36, -6, hy-44, 6, hy-38)
		c.bezTo(24, hy-40, 34, hy-24, 28.5, hy-4)
		c.quadTo(24, hy-22, 8, hy-24)
		c.quadTo(-2, hy-26, -10, hy-20)
		c.quadTo(-20, hy-14, -28.5, hy-2)
		c.closePath()
		c.fill()
		c.stroke()
		c.st.strk, c.st.lw = rgbaS(255, 255, 255, .2), 1.4
		c.beginPath()
		c.moveTo(-12, hy-32)
		c.quadTo(4, hy-38, 18, hy-32)
		c.stroke()
		ink()
	case "bald":
		for _, sx := range []float64{-1, 1} {
			c.beginPath()
			c.moveTo(sx*27, hy-12)
			c.quadTo(sx*31, hy+2, sx*26, hy+10)
			c.quadTo(sx*22, hy-2, sx*24, hy-12)
			c.closePath()
			c.fill()
			c.stroke()
		}
		c.st.fill = rgbaS(255, 255, 255, .35)
		c.beginPath()
		c.ellipse(-8, hy-24, 8, 4, -0.4, 0, tau, false)
		c.fill()
	case "bob":
		c.beginPath()
		c.moveTo(-29, hy+14)
		c.bezTo(-36, hy-34, 36, hy-34, 29, hy+14)
		c.lineTo(26, hy-6)
		c.quadTo(18, hy-20, 4, hy-16)
		c.quadTo(-10, hy-20, -20, hy-14)
		c.quadTo(-26, hy-8, -29, hy+14)
		c.closePath()
		c.fill()
		c.stroke()
	case "bun", "waves", "curls":
		c.beginPath()
		c.moveTo(-29, hy+6)
		c.bezTo(-34, hy-30, -14, hy-40, 0, hy-36)
		c.bezTo(14, hy-40, 34, hy-30, 29, hy+6)
		c.bezTo(22, hy-10, 14, hy-24, 0, hy-22)
		c.bezTo(-14, hy-24, -22, hy-10, -29, hy+6)
		c.closePath()
		c.fill()
		c.stroke()
		c.st.strk, c.st.lw = rgbaS(255, 255, 255, .18), 1.2
		c.beginPath()
		c.moveTo(-18, hy-26)
		c.quadTo(-6, hy-34, 6, hy-30)
		c.stroke()
		ink()
	}
	if o.Beard {
		c.st.fill = hair
		c.beginPath()
		c.moveTo(-25, hy+4)
		c.quadTo(-24, hy+42, 0, hy+46)
		c.quadTo(24, hy+42, 25, hy+4)
		c.quadTo(14, hy+22, 0, hy+20)
		c.quadTo(-14, hy+22, -25, hy+4)
		c.fill()
		c.stroke()
	}

	// the face
	c.st.fill = rgbaS(214, 110, 100, .28)
	c.beginPath()
	c.circle(-15, hy+9, 6.5)
	c.circle(15, hy+9, 6.5)
	c.fill()
	if o.Freckle {
		c.st.fill = rgbaS(150, 80, 50, .45)
		for _, p := range [][2]float64{{-17, hy + 5}, {-13, hy + 8}, {-19, hy + 9}, {17, hy + 5}, {13, hy + 8}, {19, hy + 9}} {
			c.beginPath()
			c.circle(p[0], p[1], 1.1)
			c.fill()
		}
	}
	c.st.fill = pInk
	ey := 3.0
	if o.Lady {
		ey = 3.4
	}
	c.beginPath()
	c.ellipse(-10, hy-3, 2.6, ey, 0, 0, tau, false)
	c.ellipse(10, hy-3, 2.6, ey, 0, 0, tau, false)
	c.fill()
	if o.Lady { // lashes
		c.st.lw = 1.4
		for _, sx := range []float64{-1, 1} {
			c.beginPath()
			c.moveTo(sx*12.5, hy-5)
			c.lineTo(sx*15.5, hy-7)
			c.stroke()
		}
	}
	c.st.lw, c.st.strk = 2.2, orC(o.Brow, shade(hair, .85))
	st, st1 := 0.0, 1.0
	if o.Stern {
		st, st1 = 2, 0
	}
	c.beginPath()
	c.moveTo(-15, hy-11+st)
	c.lineTo(-5, hy-12-st1)
	c.moveTo(15, hy-11+st)
	c.lineTo(5, hy-12-st1)
	c.stroke()
	c.st.strk, c.st.lw = pInk, 1.8
	c.beginPath()
	c.moveTo(1, hy)
	c.quadTo(-3, hy+9, 1, hy+11)
	c.stroke()
	if o.Mole {
		c.st.fill = pInk
		c.beginPath()
		c.circle(-13, hy+16, 1.4)
		c.fill()
	}
	switch {
	case o.Lady:
		c.st.fill = o.Lip
		rx, ry := 6.0, 2.8
		if o.Stern {
			rx, ry = 5, 1.8
		}
		c.beginPath()
		if o.Smile {
			c.moveTo(-7, hy+18)
			c.quadTo(0, hy+26, 7, hy+18)
			c.quadTo(0, hy+21, -7, hy+18)
			c.closePath()
		} else {
			c.ellipse(0, hy+20, rx, ry, 0, 0, tau, false)
		}
		c.fill()
	case o.Mustache == "" || o.Mustache == "pencil":
		c.beginPath()
		if o.Smile {
			c.moveTo(-7, hy+18)
			c.quadTo(0, hy+24, 7, hy+18)
		} else {
			c.moveTo(-6, hy+20)
			c.quadTo(0, hy+22, 6, hy+20)
		}
		c.stroke()
	}
	moust := orC(o.Brow, shade(hair, .9))
	switch o.Mustache {
	case "pencil":
		c.st.strk, c.st.lw = moust, 2.2
		c.beginPath()
		c.moveTo(-10, hy+14)
		c.quadTo(0, hy+11, 10, hy+14)
		c.stroke()
	case "handlebar":
		c.st.fill, c.st.strk, c.st.lw = moust, pInk, 1.6
		c.beginPath()
		c.moveTo(0, hy+13)
		c.bezTo(-8, hy+10, -18, hy+18, -24, hy+10)
		c.bezTo(-28, hy+6, -26, hy+2, -22, hy+4)
		c.bezTo(-22, hy+10, -16, hy+20, 0, hy+17)
		c.bezTo(16, hy+20, 22, hy+10, 22, hy+4)
		c.bezTo(26, hy+2, 28, hy+6, 24, hy+10)
		c.bezTo(18, hy+18, 8, hy+10, 0, hy+13)
		c.fill()
		c.stroke()
	case "walrus":
		c.st.fill, c.st.strk, c.st.lw = moust, pInk, 1.6
		c.beginPath()
		c.moveTo(0, hy+11)
		c.bezTo(-10, hy+8, -18, hy+12, -18, hy+22)
		c.quadTo(-8, hy+18, 0, hy+20)
		c.quadTo(8, hy+18, 18, hy+22)
		c.bezTo(18, hy+12, 10, hy+8, 0, hy+11)
		c.fill()
		c.stroke()
	}
	ink()
	switch o.Glasses {
	case "round":
		c.st.strk, c.st.lw = rgb("#3a2a1e"), 2
		c.beginPath()
		c.circle(-10, hy-3, 7.5)
		c.moveTo(-2.5, hy-3)
		c.arc(10, hy-3, 7.5, math.Pi, math.Pi*3)
		c.moveTo(-2.5, hy-4)
		c.lineTo(2.5, hy-4)
		c.moveTo(-17.5, hy-4)
		c.lineTo(-26, hy-7)
		c.moveTo(17.5, hy-4)
		c.lineTo(26, hy-7)
		c.stroke()
		ink()
	case "cateye":
		c.st.strk, c.st.lw = rgb("#1e1414"), 2.4
		for _, sx := range []float64{-1, 1} {
			c.beginPath()
			c.moveTo(sx*3, hy-6)
			c.quadTo(sx*10, hy-9, sx*19, hy-10)
			c.quadTo(sx*18, hy+3, sx*10, hy+3)
			c.quadTo(sx*3, hy+3, sx*3, hy-6)
			c.stroke()
		}
		c.beginPath()
		c.moveTo(-3, hy-5)
		c.lineTo(3, hy-5)
		c.stroke()
		ink()
	}

	// hats
	switch o.Hat {
	case "beret":
		c.save()
		c.translate(4, hy-30)
		c.rotate(-0.14)
		c.st.fill = o.HatColor
		c.beginPath()
		c.ellipse(0, 0, 36, 13, 0, 0, tau, false)
		c.fill()
		c.stroke()
		c.beginPath()
		c.ellipse(-2, -4, 28, 8, 0, math.Pi, 0, false)
		c.st.strk = rgbaS(255, 255, 255, .15)
		c.stroke()
		ink()
		c.st.fill = o.HatColor
		c.beginPath()
		c.moveTo(-2, -12)
		c.lineTo(0, -20)
		c.lineTo(3, -12)
		c.closePath()
		c.fill()
		c.stroke()
		c.restore()
	case "cloche":
		c.st.fill = o.HatColor
		c.beginPath()
		c.moveTo(-33, hy-4)
		c.bezTo(-36, hy-50, 36, hy-50, 33, hy-4)
		c.quadTo(0, hy-16, -33, hy-4)
		c.closePath()
		c.fill()
		c.stroke()
		c.st.strk, c.st.lw = shade(o.HatColor, 1.6), 4
		c.beginPath()
		c.moveTo(-31, hy-14)
		c.quadTo(0, hy-26, 31, hy-14)
		c.stroke()
		ink()
		c.st.fill = rgb("#e8b4b8")
		for _, p := range [][2]float64{{20, hy - 22}, {26, hy - 18}, {22, hy - 15}} {
			c.beginPath()
			c.circle(p[0], p[1], 4)
			c.fill()
			c.stroke()
		}
	case "cap":
		c.st.fill = o.HatColor
		c.beginPath()
		c.moveTo(-30, hy-14)
		c.bezTo(-30, hy-42, 26, hy-46, 32, hy-22)
		c.quadTo(4, hy-26, -30, hy-14)
		c.closePath()
		c.fill()
		c.stroke()
		c.beginPath()
		c.moveTo(4, hy-24)
		c.quadTo(34, hy-28, 40, hy-16)
		c.quadTo(24, hy-12, 4, hy-18)
		c.closePath()
		c.fill()
		c.stroke()
		c.st.fill = shade(o.HatColor, .8)
		c.beginPath()
		c.circle(-2, hy-40, 3)
		c.fill()
	case "fedora":
		c.st.fill = o.HatColor
		c.beginPath()
		c.ellipse(0, hy-28, 46, 9, 0, 0, tau, false)
		c.fill()
		c.stroke()
		c.beginPath()
		c.moveTo(-25, hy-30)
		c.bezTo(-28, hy-60, -10, hy-66, 0, hy-58)
		c.bezTo(10, hy-66, 28, hy-60, 25, hy-30)
		c.closePath()
		c.fill()
		c.stroke()
		c.st.fill = rgb("#1e1414")
		c.beginPath()
		c.moveTo(-25, hy-32)
		c.lineTo(25, hy-32)
		c.lineTo(26, hy-40)
		c.lineTo(-26, hy-40)
		c.closePath()
		c.fill()
	case "toque":
		c.st.fill = rgb("#fdfcf8")
		c.beginPath()
		c.rrect(-24, hy-46, 48, 16, 3)
		c.fill()
		c.stroke()
		for _, p := range [][3]float64{{-18, hy - 60, 15}, {18, hy - 60, 15}, {0, hy - 68, 18}} {
			c.beginPath()
			c.circle(p[0], p[1], p[2])
			c.fill()
			c.stroke()
		}
		c.beginPath()
		c.rect(-22, hy-58, 44, 14)
		c.fill()
		c.st.strk, c.st.lw = rgbaS(0, 0, 0, .18), 1.2
		for _, x := range []float64{-10, 0, 10} {
			c.beginPath()
			c.moveTo(x, hy-46)
			c.lineTo(x, hy-32)
			c.stroke()
		}
		ink()
	}
	c.restore()
}

// renderCameo draws a round, brass-framed portrait against a dusk sky.
func renderCameo(o Look, size int) *ebiten.Image {
	S := float64(size)
	r := S / 2
	out := ebiten.NewImage(size, size)
	px := make([]byte, size*size*4)
	in, edge := rgb("#ffd9a0"), rgb("#8a5a8a")
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			fx, fy := float64(x)+.5, float64(y)+.5
			d := math.Hypot(fx-r, fy-r)
			a := clamp(r-2-d+.5, 0, 1)
			t := clamp((math.Hypot(fx-r, fy-r*0.8)-0.1*r)/(0.9*r), 0, 1)
			cc := lerpRGBA(in, edge, t)
			// a skyline low behind the shoulders
			if fy > S*(0.62+0.06*math.Abs(math.Sin(math.Floor(fx/S*7)*1.7))) {
				cc = lerpRGBA(cc, rgb("#3a2a4a"), 0.75)
			}
			i := (y*size + x) * 4
			px[i], px[i+1], px[i+2], px[i+3] = uint8(float64(cc.R)*a), uint8(float64(cc.G)*a), uint8(float64(cc.B)*a), uint8(255*a)
		}
	}
	out.WritePixels(px)
	fig := ebiten.NewImage(size, size)
	cv := newCanvas(fig)
	cv.translate(r, S*1.02)
	k := S / 190
	cv.scale(k, k)
	bust(cv, o)
	mask := ebiten.NewImage(size, size)
	mpx := make([]byte, size*size*4)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			a := uint8(255 * clamp(r-7-math.Hypot(float64(x)+.5-r, float64(y)+.5-r)+.5, 0, 1))
			i := (y*size + x) * 4
			mpx[i], mpx[i+1], mpx[i+2], mpx[i+3] = a, a, a, a
		}
	}
	mask.WritePixels(mpx)
	fig.DrawImage(mask, &ebiten.DrawImageOptions{Blend: ebiten.BlendDestinationIn})
	out.DrawImage(fig, nil)
	ring := func(rad, w float64, col color.RGBA) {
		p := &vector.Path{}
		p.Arc(float32(r), float32(r), float32(rad), 0, 2*math.Pi, vector.Clockwise)
		p.Close()
		cstroke(out, p, w, col)
	}
	ring(r-S*0.03, S*0.05, rgb("#b8913f"))
	ring(r-S*0.012, 1.5, rgb("#e8cf8a"))
	ring(r-S*0.06, 1.5, rgb("#6a4a1e"))
	fig.Deallocate()
	mask.Deallocate()
	return out
}

func lerpRGBA(a, b color.RGBA, t float64) color.RGBA {
	l := func(x, y uint8) uint8 { return uint8(math.Round(float64(x) + (float64(y)-float64(x))*t)) }
	return color.RGBA{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), l(a.A, b.A)}
}

var cameoCache = map[string]*ebiten.Image{}

// cameoImg is the portrait for a face (made the first time it's needed).
func cameoImg(face string) *ebiten.Image {
	if im, ok := cameoCache[face]; ok {
		return im
	}
	f, ok := parseFace(face)
	if !ok {
		return nil
	}
	if len(cameoCache) > 60 {
		for k, im := range cameoCache {
			im.Deallocate()
			delete(cameoCache, k)
		}
	}
	im := renderCameo(f.look(), 256)
	cameoCache[face] = im
	return im
}
