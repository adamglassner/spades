//go:build !headless

package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand/v2"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	sp "github.com/adamglassner/spades/internal/spades"
)

// The screen: a brick rooftop parapet with string lights on top, the card
// table below it, and the title / rules / credits / result panels over everything. All of it is
// laid out in logical pixels and drawn at the monitor's real resolution.

const (
	barH          = 50.0
	tableX        = 14.0
	tableY        = barH + 14
	tableMarginLR = 28.0
	tableMarginB  = 14.0
)

type hit struct {
	id         string
	x, y, w, h float64 // window coordinates
	rotated    bool    // a card: (x, y) is its centre, rot its angle
	rot        float64
	card       *cardV
	ptr        bool
	clip       *image.Rectangle // only clickable inside this (scrolling panels)
}

func (h hit) contains(mx, my float64) bool {
	if h.clip != nil {
		c := *h.clip
		if mx < float64(c.Min.X) || mx >= float64(c.Max.X) || my < float64(c.Min.Y) || my >= float64(c.Max.Y) {
			return false
		}
	}
	if h.rotated {
		a := -h.rot * math.Pi / 180
		dx, dy := mx-h.x, my-h.y
		lx := dx*math.Cos(a) - dy*math.Sin(a)
		ly := dx*math.Sin(a) + dy*math.Cos(a)
		return math.Abs(lx) <= h.w/2 && math.Abs(ly) <= h.h/2
	}
	return mx >= h.x && mx < h.x+h.w && my >= h.y && my < h.y+h.h
}

type uiState struct {
	winW, winH, ds float64
	last           time.Time
	tbl, tblMask   *ebiten.Image
	off            *ebiten.Image // scratch for rotated things (the banner)
	hits           []hit
	hover          string
	hoverCard      *cardV
	mx, my         float64
	down           bool
	dragVol        bool
	volX, volW     float64
	volDirty       bool
	scroll         float64
	scrollMax      float64
	steamT, clock  float64
	promptText     string
	promptA        float64
	fullscreen     bool
	hitOX, hitOY   float64
}

/* ---------------- ebiten.Game ---------------- */

func (g *game) Layout(w, h int) (int, int) { panic("LayoutF is used") }

func (g *game) LayoutF(w, h float64) (float64, float64) {
	ds := ebiten.Monitor().DeviceScaleFactor()
	if ds <= 0 {
		ds = 1
	}
	g.mu.Lock()
	g.ui.winW, g.ui.winH, g.ui.ds = w, h, ds
	g.mu.Unlock()
	return w * ds, h * ds
}

func (g *game) resize() {
	u := &g.ui
	if u.winW == 0 {
		return
	}
	W, H := math.Max(200, u.winW-tableMarginLR), math.Max(200, u.winH-tableY-tableMarginB)
	if W == g.Wd && H == g.Ht {
		return
	}
	g.Wd, g.Ht = W, H
	g.measure()
	g.layoutCards(true)
}

func (g *game) Update() error {
	now := time.Now()
	u := &g.ui
	dt := 1.0 / 60
	if !u.last.IsZero() {
		dt = clamp(now.Sub(u.last).Seconds(), 0, 0.05)
	}
	u.last = now
	snd.update(dt)

	g.mu.Lock()
	defer g.mu.Unlock()
	u.clock += dt
	u.steamT += dt * 1000
	g.resize()

	cx, cy := ebiten.CursorPosition()
	ds := math.Max(u.ds, 1e-3)
	u.mx, u.my = float64(cx)/ds, float64(cy)/ds

	var top *hit
	for i := range u.hits {
		if u.hits[i].contains(u.mx, u.my) {
			top = &u.hits[i]
		}
	}
	hov := ""
	var hc *cardV
	if top != nil {
		hov, hc = top.id, top.card
	}
	if hc != nil && !hc.pick {
		hc = nil
	}
	if hc != nil && hc != u.hoverCard {
		snd.play("hover")
	}
	u.hoverCard, u.hover = hc, hov
	if top != nil && top.ptr && (top.card == nil || top.card.pick) || u.dragVol {
		ebiten.SetCursorShape(ebiten.CursorShapePointer)
	} else {
		ebiten.SetCursorShape(ebiten.CursorShapeDefault)
	}
	for _, c := range g.deck {
		t := 0.0
		if c == hc {
			t = 1
		}
		c.lift += (t - c.lift) * math.Min(1, dt/0.05)
	}

	u.down = ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if top == nil || !strings.HasPrefix(top.id, "field:") {
			g.focus = ""
		}
		if top != nil {
			g.click(*top)
		}
	}
	if u.dragVol {
		if u.down {
			g.setVolAt(u.mx)
		} else {
			u.dragVol = false
		}
	}
	if u.volDirty && !u.dragVol {
		u.volDirty = false
		g.persist()
	}
	if _, wy := ebiten.Wheel(); wy != 0 {
		u.scroll = clamp(u.scroll-wy*48, 0, u.scrollMax)
	}

	typing := g.focus != ""
	if typing {
		g.typeText()
	}
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyF2):
		g.newGameKey()
	case !typing && inpututil.IsKeyJustPressed(ebiten.KeyM):
		g.toggleMusic()
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		g.showRules, g.showCredits, g.focus = false, false, ""
	case inpututil.IsKeyJustPressed(ebiten.KeyF11):
		u.fullscreen = !u.fullscreen
		ebiten.SetFullscreen(u.fullscreen)
	}

	if g.prompt != "" {
		u.promptText = g.prompt
		u.promptA = math.Min(1, u.promptA+dt/0.25)
	} else {
		u.promptA = math.Max(0, u.promptA-dt/0.25)
	}
	g.stepParticles(dt)
	return nil
}

func (g *game) toggleMusic() {
	g.save.Settings.Music = !g.save.Settings.Music
	snd.setMusic(g.save.Settings.Music)
	g.persist()
}

func (g *game) setVolAt(mx float64) {
	u := &g.ui
	v := int(math.Round(clamp((mx-u.volX)/u.volW, 0, 1) * 100))
	if v != g.save.Settings.Vol {
		g.save.Settings.Vol = v
		snd.setVolume(float64(v) / 100)
		u.volDirty = true
	}
}

func (g *game) click(h hit) {
	u := &g.ui
	if h.card != nil {
		if w := g.wait; w != nil && h.card.pick {
			g.wait = nil
			w.ch <- h.card
		}
		return
	}
	id := h.id
	if g.setupClick(id) {
		return
	}
	switch {
	case id == "new":
		g.newGameKey()
	case id == "start":
		g.goPlay()
	case id == "menu" || id == "toMenu":
		g.toMenu()
	case id == "leave":
		g.leaveOnline()
		g.closePanels()
		g.toMenu()
	case id == "rules" || id == "how":
		g.showRules, u.scroll = true, 0
	case id == "credits":
		g.showCredits, u.scroll = true, 0
	case id == "closeRules":
		g.showRules = false
	case id == "closeCredits":
		g.showCredits = false
	case id == "again":
		snd.play("tick")
		g.newDeal()
	case id == "music":
		g.toggleMusic()
	case id == "sfx":
		g.save.Settings.Sfx = !g.save.Settings.Sfx
		snd.setSfx(g.save.Settings.Sfx)
		g.persist()
		snd.play("tick")
	case id == "vol":
		u.dragVol = true
		g.setVolAt(u.mx)
	case strings.HasPrefix(id, "ask:"):
		i, _ := strconv.Atoi(id[4:])
		if a := g.ask; a != nil && i < len(a.btns) {
			snd.play("tick")
			g.ask = nil
			a.ch <- askRes{btn: a.btns[i], idx: i}
		}
	case strings.HasPrefix(id, "opt:"):
		kv := strings.SplitN(id[4:], "=", 2)
		st := &g.save.Settings
		switch kv[0] {
		case "target":
			st.Target, _ = strconv.Atoi(kv[1])
		case "speed":
			st.Speed = kv[1]
		}
		g.persist()
		snd.play("tick")
	}
}

// addHit registers a clickable area; while the table is being drawn the
// coordinates are table coordinates (hitOX/hitOY move them to the window).
func (g *game) addHit(h hit) {
	h.x += g.ui.hitOX
	h.y += g.ui.hitOY
	g.ui.hits = append(g.ui.hits, h)
}

func (g *game) tableHit(h hit) { g.addHit(h) }

/* ---------------- particles ---------------- */

var confettiCols = []color.NRGBA{hexc("b8432b", 1), hexc("ffd25a", 1), hexc("2c5a8c", 1), hexc("f6efe2", 1), hexc("3ec7b4", 1), hexc("f08a3c", 1)}

func (g *game) stepParticles(dt float64) {
	f := dt * 1000 / 16.7
	t := g.ui.clock * 1000
	out := g.parts[:0]
	for i := range g.parts {
		p := g.parts[i]
		p.x += p.vx * f
		p.y += p.vy * f
		p.life -= p.d * f
		switch p.k {
		case 0:
			p.vy += 0.25 * f
			p.rot += p.vr * f
			if p.y > g.Ht+20 {
				p.life = 0
			}
		case 1:
			p.vy += 0.05 * f
		case 2:
			p.vx += math.Sin(t/300+float64(i)) * 0.03
			p.rot += p.vr * f
			if p.y > g.Ht+20 {
				p.life = 0
			}
		}
		if p.life > 0 {
			out = append(out, p)
		}
	}
	g.parts = out
}

/* ---------------- drawing helpers ---------------- */

// base returns the baseline for a line box of height lh whose top is at top.
func base(f font, top, lh float64) float64 {
	m := face(f, 1).Metrics()
	return top + (lh-(m.HAscent+m.HDescent))/2 + m.HAscent
}

func (p pen) vgrad(x, y, w, h float64, c1, c2 color.NRGBA) {
	var t ebiten.GeoM
	p.geo(&t)
	pt := func(px, py float64, c color.NRGBA) ebiten.Vertex {
		X, Y := t.Apply(px, py)
		return ebiten.Vertex{DstX: float32(X), DstY: float32(Y), SrcX: 2, SrcY: 2,
			ColorR: float32(c.R) / 255, ColorG: float32(c.G) / 255, ColorB: float32(c.B) / 255, ColorA: float32(c.A) / 255}
	}
	vs := []ebiten.Vertex{pt(x, y, c1), pt(x+w, y, c1), pt(x, y+h, c2), pt(x+w, y+h, c2)}
	p.dst.DrawTriangles(vs, []uint16{0, 1, 2, 1, 3, 2}, whitePixel(), &ebiten.DrawTrianglesOptions{})
}

// soft approximates a CSS blurred box-shadow with stacked translucent shapes.
func (p pen) soft(x, y, w, h, r, blur, a float64) {
	const n = 7
	for i := n; i >= 1; i-- {
		e := blur * float64(i) / n * 0.55
		p.rr(x-e, y-e, w+2*e, h+2*e, r+e, color.NRGBA{0, 0, 0, u8(a / n * 1.15)})
	}
}

func (p pen) rotRect(cx, cy, w, h, rot float64, c color.Color) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(w/2, h/2)
	op.GeoM.Translate(-w/2, -h/2)
	op.GeoM.Rotate(rot)
	op.GeoM.Translate(cx, cy)
	p.geo(&op.GeoM)
	op.ColorScale.ScaleWithColor(c)
	p.dst.DrawImage(whitePixel(), op)
}

func (p pen) dashH(x0, x1, y float64, c color.Color) {
	for x := x0; x < x1; x += 6 {
		p.rect(x, y, math.Min(3, x1-x), 1, c)
	}
}

// scaled returns a pen that scales everything by k about (cx, cy).
func (p pen) scaled(cx, cy, k float64) pen {
	return pen{p.dst, p.s * k, (cx*(1-k) + p.ox*1) / k, (cy*(1-k) + p.oy) / k}
}

func (p pen) drawImg(im *ebiten.Image, m ebiten.GeoM, cs ebiten.ColorScale) {
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM = m
	p.geo(&op.GeoM)
	op.ColorScale = cs
	p.dst.DrawImage(im, op)
}

func (p pen) drawImgCM(im *ebiten.Image, m ebiten.GeoM, cm colorm.ColorM) {
	op := &colorm.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM = m
	p.geo(&op.GeoM)
	colorm.DrawImage(p.dst, im, cm, op)
}

// cameo draws a regular's round portrait.
func (p pen) cameo(who string, cx, cy, d float64, cm *colorm.ColorM) {
	im := cameoImg(who)
	if im == nil {
		return
	}
	b := im.Bounds()
	var m ebiten.GeoM
	m.Translate(-float64(b.Dx())/2, -float64(b.Dy())/2)
	m.Scale(d/float64(b.Dx()), d/float64(b.Dy()))
	m.Translate(cx, cy)
	if cm != nil {
		p.drawImgCM(im, m, *cm)
		return
	}
	p.drawImg(im, m, ebiten.ColorScale{})
}

// cardImg draws a card face (or back) with its shadow, centred at (cx, cy).
func (p pen) cardImg(id string, cx, cy, w, rot float64) {
	h := w * cardH / cardW
	var m ebiten.GeoM
	m.Translate(-(cardW+2*cardPad)/2, -(cardH+2*cardPad)/2)
	m.Scale(w/cardW, h/cardH)
	m.Rotate(rot * math.Pi / 180)
	m.Translate(cx, cy)
	p.drawImg(art.cardShadow, m, ebiten.ColorScale{})
	m.Reset()
	m.Translate(-cardW/2, -cardH/2)
	m.Scale(w/cardW, h/cardH)
	m.Rotate(rot * math.Pi / 180)
	m.Translate(cx, cy)
	p.drawImg(art.cards[id], m, ebiten.ColorScale{})
}

/* ---------------- buttons ---------------- */

type btnStyle struct {
	fill, edge, ink color.NRGBA
	ledge, border   float64
	radius          float64
	hoverLight      float64 // brightness on hover
	hoverFill       *color.NRGBA
}

var (
	stBar   = btnStyle{fill: hexc("fff6e2", 1), edge: cBrown, ink: cInk, ledge: 2, border: 2, radius: 6, hoverFill: &cWhite}
	stBtn   = btnStyle{fill: cBarn, edge: cMaroon, ink: cWhite, ledge: 3, border: 2, radius: 8, hoverLight: 1.1}
	stPass  = btnStyle{fill: cWhite, edge: cBrown, ink: cInk, ledge: 3, border: 2, radius: 8, hoverLight: 1.1}
	stBig   = btnStyle{fill: cBarn, edge: cMaroon, ink: cWhite, ledge: 5, border: 3, radius: 10, hoverLight: 1.08}
	stGhost = btnStyle{fill: cWhite, edge: cBrown, ink: cBarn, ledge: 5, border: 3, radius: 10, hoverLight: 1.08}
	stOff   = btnStyle{fill: hexc("d8cfc0", 1), edge: hexc("9a8a7a", 1), ink: hexc("7a6a5a", 1), ledge: 3, border: 2, radius: 8}
)

func lighten(c color.NRGBA, k float64) color.NRGBA {
	l := func(v uint8) uint8 { return uint8(math.Min(255, float64(v)*k)) }
	return color.NRGBA{l(c.R), l(c.G), l(c.B), c.A}
}

// button draws a CSS-style button; returns the pen offset for its label
// (it sinks when pressed).
func (g *game) button(p pen, id string, x, y, w, h float64, st btnStyle) (dy float64, hover bool) {
	hover = g.ui.hover == id
	pressed := hover && g.ui.down
	fill, edge := st.fill, st.edge
	if hover {
		if st.hoverFill != nil {
			fill = *st.hoverFill
		} else if st.hoverLight > 0 {
			fill, edge = lighten(fill, st.hoverLight), lighten(edge, st.hoverLight)
		}
	}
	ledge := st.ledge
	if pressed {
		dy = math.Min(st.ledge, 3)
		ledge = st.ledge - dy
		if st.ledge >= 5 {
			ledge = 2
		}
	}
	p.box(x, y+dy, w, h, st.radius, st.border, fill, edge, ledge, 0)
	g.addHit(hit{id: id, x: x + p.ox, y: y + p.oy, w: w, h: h + st.ledge, ptr: true})
	return dy, hover
}

/* ---------------- Draw ---------------- */

func (g *game) Draw(screen *ebiten.Image) {
	g.mu.Lock()
	defer g.mu.Unlock()
	u := &g.ui
	if u.winW == 0 {
		return
	}
	u.hits = u.hits[:0]
	now := time.Now()
	p := pen{dst: screen, s: u.ds}
	if showFaces {
		g.drawFaces(p)
		return
	}
	g.drawRoom(p)
	g.drawTable(screen, now)
	g.drawBar(p)
	if g.showTitle {
		g.drawTitle(p)
	}
	if g.showRules {
		g.drawRules(p)
	}
	if g.showCredits {
		g.drawCredits(p)
	}
	if g.showTogether {
		g.drawTogether(p)
	}
	if g.showHotseat {
		g.drawHotseat(p)
	}
	if g.showOnline {
		g.drawOnline(p)
	}
	if g.showLobby {
		g.drawLobby(p)
	}
	if g.showResult {
		g.drawResult(p)
	}
	if g.passCh != nil {
		g.drawPass(p)
	}
	if g.message != "" {
		g.drawMsg(p)
	}
	if !g.anyPanel() {
		u.scrollMax = 0
	}
}

// drawRoom is the rooftop behind the table: tar paper under a dusk sky.
func (g *game) drawRoom(p pen) {
	u := &g.ui
	p.vgrad(0, barH, u.winW, u.winH-barH, hexc("2a2238", 1), hexc("16121c", 1))
}

/* ---------------- the table ---------------- */

func (g *game) drawTable(screen *ebiten.Image, now time.Time) {
	u := &g.ui
	W, H := g.Wd, g.Ht
	pw, ph := int(math.Ceil(W*u.ds)), int(math.Ceil(H*u.ds))
	if u.tbl == nil || u.tbl.Bounds().Dx() != pw || u.tbl.Bounds().Dy() != ph {
		if u.tbl != nil {
			u.tbl.Deallocate()
			u.tblMask.Deallocate()
		}
		u.tbl = ebiten.NewImage(pw, ph)
		u.tblMask = ebiten.NewImage(pw, ph)
		pen{dst: u.tblMask, s: u.ds}.rr(0, 0, W, H, 22, cWhite)
	}
	sp := pen{dst: screen, s: u.ds}
	sp.soft(tableX, tableY+12, W, H, 22, 40, 0.65)
	sp.rr(tableX-6, tableY-6, W+12, H+12, 28, hexc("2a1c12", 1)) // a painted wood rim
	sp.rr(tableX-4, tableY-4, W+8, H+8, 26, hexc("8a5a34", 1))
	sp.rr(tableX-2, tableY-2, W+4, H+4, 24, hexc("4a3020", 1))

	t := u.tbl
	t.Clear()
	u.hitOX, u.hitOY = tableX, tableY
	defer func() { u.hitOX, u.hitOY = 0, 0 }()
	p := pen{dst: t, s: u.ds}
	// weathered deck boards
	b := art.deck.Bounds()
	k := math.Max(W/float64(b.Dx()), H/float64(b.Dy()))
	p.image(art.deck, W/2, H/2, float64(b.Dx())*k, float64(b.Dy())*k, 0, ebiten.ColorScale{})
	p.imageTL(art.vignette, 0, 0, W, H, ebiten.ColorScale{})

	// string lights strung across the top of the table
	lb := art.lights.Bounds()
	lk := W / float64(lb.Dx())
	p.imageTL(art.lights, 0, 0, W, float64(lb.Dy())*lk, ebiten.ColorScale{})

	// a painted ring where the tricks are played
	ring := g.ch * 2.5 * float64(matImg) / (float64(matImg) - 100)
	p.image(art.mat, W/2, g.cy(), ring, ring, 0, ebiten.ColorScale{})

	// a cold soda and a slice
	ms, ps := g.propSize(), g.propSize()*1.1
	p.imageTL(art.drink, 10, H-ms-6, ms, ms, ebiten.ColorScale{})
	p.imageTL(art.snack, W-ps-8, H-ps*0.9, ps, ps, ebiten.ColorScale{})

	if g.started {
		for _, pl := range g.players {
			g.drawSeat(p, pl, now)
		}
		cs := append([]*cardV(nil), g.deck...)
		sort.SliceStable(cs, func(i, j int) bool { return cs[i].z < cs[j].z })
		for _, c := range cs {
			g.drawCard(p, c, now)
		}
		for _, pl := range g.players {
			g.drawBubble(p, pl, now)
		}
	}

	p.image(art.lamp, W/2, H/2, W*1.4, H*1.4, 0, ebiten.ColorScale{})
	if g.started {
		g.drawHandBox(p)
		g.drawScoreBox(p)
	}
	g.drawPrompt(p)
	if g.ask != nil {
		g.drawAsk(p)
	}
	g.drawFx(p, ms)
	g.drawBanner(p, now)
	p.text("v"+version, slab(13), W-ps-18, H-10-3, color.NRGBA{240, 220, 180, 110}, alignR)

	t.DrawImage(u.tblMask, &ebiten.DrawImageOptions{Blend: ebiten.BlendDestinationIn})
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(math.Round(tableX*u.ds), math.Round(tableY*u.ds))
	screen.DrawImage(t, op)
}

func (g *game) drawCard(p pen, c *cardV, now time.Time) {
	if c.zn.K == zNone {
		return
	}
	x, y, r, s := c.pos(now)
	a := math.Mod(math.Abs(c.angle(now)), 360)
	sx := math.Abs(math.Cos(a * math.Pi / 180))
	img := art.cards["back"]
	faceUp := (a < 90 || a > 270) && c.id != ""
	if faceUp {
		img = art.cards[c.id]
	}
	selT := 0.0
	if c.sel {
		selT = 1
	}
	c.sl += (selT - c.sl) * 0.25
	lift := c.lift*0.12*g.ch + c.sl*0.22*g.ch
	geo := func(iw, ih float64) ebiten.GeoM {
		var m ebiten.GeoM
		m.Translate(-iw/2, -ih/2)
		m.Scale(g.cw/cardW*sx*s, g.ch/cardH*s)
		m.Translate(0, -lift)
		m.Rotate(r * math.Pi / 180)
		m.Translate(x, y)
		return m
	}
	var cs ebiten.ColorScale
	sh := art.cardShadow
	fresh := c.fresh && faceUp && c.zn.K == zHand && c.zn.P == g.viewer
	switch {
	case c.win:
		sh = art.ringWin
	case c.sel:
		sh = art.ringSel
	case fresh:
		sh = art.ringFresh
	case c.pick:
		sh = art.ringPick
	}
	p.drawImg(sh, geo(cardW+2*cardPad, cardH+2*cardPad), cs)
	if c.dim {
		var cm colorm.ColorM
		cm.ChangeHSV(0, 0.6, 0.62)
		p.drawImgCM(img, geo(cardW, cardH), cm)
	} else {
		p.drawImg(img, geo(cardW, cardH), cs)
	}
	if g.wait != nil && c.pick {
		g.tableHit(hit{x: x, y: y - lift, w: g.cw * s, h: g.ch * s, rot: r, rotated: true, card: c, ptr: true})
	}
}

/* ---------------- seats ---------------- */

type seatBox struct {
	x, y, w, h       float64 // whole seat
	camX, camY       float64
	whoX, whoY, whoW float64
	align            int // alignC, alignL, alignR
}

const whoH = 2 + 3 + 21.6 + 17.55 + 4 + 2

func (g *game) seatChips(pl *player) []string {
	var out []string
	if g.bids[pl.i] == sp.Nil {
		out = append(out, "NIL")
	}
	return out
}

func chipW(s string) float64 { return richWidth(s, slab(11)) + 12 }

func (g *game) seatName(pl *player) (string, string) {
	v := g.viewer
	sc := "Partner"
	switch {
	case pl.i == v:
		sc = "Partner: " + g.partnerName(v)
	case team(pl.i) != team(v):
		sc = "Opponent"
	}
	if b := g.bids[pl.i]; b != noBid {
		sc = "Bid " + bidWord(b) + " · Took " + itoa(g.tricks[pl.i])
	} else if pl.isTurn && g.started && !g.finished && len(g.history) == 0 && len(g.trick) == 0 && len(pl.hand) == 13 {
		sc = "Bidding…"
	}
	if pl.i == v {
		nm := "You"
		if g.mode == ModeHotseat {
			nm = pl.name
		}
		return nm, sc
	}
	return pl.full, sc
}

// hasCam: everyone but the person the table is drawn for shows their face.
func (g *game) hasCam(pl *player) bool { return pl.i != g.viewer && faceOK(pl.face) }

func (g *game) propSize() float64 { return math.Min(150, g.ch*0.9) }

func (g *game) sideSpread() float64 { return math.Min(g.cw*0.28, g.Ht*0.3/12) }

func (g *game) seatLayout(pl *player) seatBox {
	nm, ct := g.seatName(pl)
	nw := measure(nm, slab(16))
	for _, c := range g.seatChips(pl) {
		nw += 6 + chipW(c)
	}
	ww := math.Max(nw, measure(ct, italic(13, 600))) + 22 + 4
	cam := 0.0
	if g.hasCam(pl) {
		cam = 72 + 9
	}
	w, h := ww+cam, math.Max(whoH, 72)
	if !g.hasCam(pl) {
		h = whoH
	}
	var sb seatBox
	sb.w, sb.h = w, h
	seat := g.seatOf(pl.i)
	_, ay := g.anchor(seat)
	switch seat {
	case seatS:
		half := 6*math.Min(g.cw*0.56, g.Wd*0.5/12) + g.cw*0.55
		x := g.Wd/2 - half - 18 - w/2
		x = math.Max(g.propSize()+18+w/2, x)
		sb.x, sb.y, sb.align = x-w/2, ay+g.ch*0.12-h/2, alignC
	case seatN:
		half := 6*math.Min(g.cw*0.3, g.Wd*0.34/12) + g.cw*0.45
		x := g.Wd/2 - half - 22 - w/2
		sb.x, sb.y, sb.align = x-w/2, ay-h/2+6, alignC
	case seatW:
		sb.x, sb.y, sb.align = 14, g.cy()+6*g.sideSpread()+g.cw*0.45+18, alignL
	default:
		sb.x, sb.y, sb.align = g.Wd-14-w, g.cy()+6*g.sideSpread()+g.cw*0.45+18, alignR
	}
	if seat == seatE { // label first, then the cameo
		sb.whoX, sb.camX = sb.x, sb.x+ww+9+36
	} else {
		sb.camX, sb.whoX = sb.x+36, sb.x+cam
	}
	sb.camY, sb.whoY, sb.whoW = sb.y+h/2, sb.y+h/2-whoH/2, ww
	return sb
}

func (g *game) drawSeat(p pen, pl *player, now time.Time) {
	sb := g.seatLayout(pl)
	turnK := 0.0
	if pl.isTurn {
		turnK = math.Min(1, now.Sub(pl.turnAt).Seconds()/0.3)
	}
	if g.hasCam(pl) {
		k := 1 + 0.06*turnK
		cx, cy, d := sb.camX, sb.camY, 72*k
		if turnK > 0 {
			for i := 5; i >= 1; i-- {
				e := 4 + 14*float64(i)/5
				p.circle(cx, cy, d/2+e, color.NRGBA{255, 214, 98, u8(0.75 * turnK / 5)})
			}
			p.circle(cx, cy, d/2+4, colA(cGold, turnK))
		} else {
			p.circle(cx, cy+4, d/2+4, color.NRGBA{0, 0, 0, 70})
			p.circle(cx, cy+4, d/2+1, color.NRGBA{0, 0, 0, 50})
		}
		p.cameo(pl.face, cx, cy, d, nil)
	}
	edge := cBrown
	if pl.isTurn {
		edge = hexc("c78a12", 1)
	}
	x, y, w := sb.whoX, sb.whoY, sb.whoW
	p.rr(x, y+3, w, whoH, 8, cBrown)
	p.rr(x, y, w, whoH, 8, edge)
	p.rr(x+2, y+2, w-4, whoH-4, 6, cPaper)
	nm, ct := g.seatName(pl)
	tx := x + 13
	align := alignL
	if sb.align == alignR {
		tx, align = x+w-13, alignR
	}
	by := base(slab(16), y+5, 21.6)
	lx := tx
	if align == alignR {
		lw := measure(nm, slab(16))
		for _, c := range g.seatChips(pl) {
			lw += 6 + chipW(c)
		}
		lx = tx - lw
	}
	p.text(nm, slab(16), lx, by, cBarn2, alignL)
	cx := lx + measure(nm, slab(16))
	for _, c := range g.seatChips(pl) {
		cx += 6
		cw := chipW(c)
		p.rr(cx, by-13, cw, 17, 5, cBarn)
		p.rich(c, slab(11), cx+cw/2, by-0.5, cWhite, alignC)
		cx += cw
	}
	p.text(ct, italic(13, 600), tx, base(italic(13, 600), y+5+21.6, 17.55), cSub, align)
}

func (g *game) drawBubble(p pen, pl *player, now time.Time) {
	if pl.say == "" || now.Before(pl.sayAt) {
		return
	}
	k := math.Min(1, now.Sub(pl.sayAt).Seconds()/0.2)
	if now.After(pl.sayEnd) {
		k = 1 - now.Sub(pl.sayEnd).Seconds()/0.2
	}
	if k <= 0 {
		return
	}
	sb := g.seatLayout(pl)
	f := body(16, 700)
	bw := measure(pl.say, f) + 28 + 4
	bh := 2 + 7 + 21.6 + 7 + 2
	cx := sb.x + sb.w/2
	switch sb.align {
	case alignL:
		cx = math.Max(sb.x+36, sb.x+bw/2-10)
	case alignR:
		cx = math.Min(sb.x+sb.w-36, sb.x+sb.w-bw/2+10)
	}
	cx = clamp(cx, bw/2+8, g.Wd-bw/2-8)
	top := g.seatOf(pl.i) == seatN
	y := sb.y - 10 - bh
	if top {
		y = sb.y + sb.h + 10
	}
	tailX := clamp(sb.camX, cx-bw/2+18, cx+bw/2-18)
	sc := 0.6 + 0.4*easeCSS(k)
	q := p.scaled(cx, y+bh/2, sc)
	q.soft(cx-bw/2, y+6, bw, bh, 16, 16, 0.35*k)
	q.rr(cx-bw/2, y, bw, bh, 16, colA(cBrown, k))
	q.rr(cx-bw/2+2, y+2, bw-4, bh-4, 14, colA(hexc("fffdf5", 1), k))
	var tail vector.Path
	if top {
		tail.MoveTo(float32(tailX-8), float32(y))
		tail.LineTo(float32(tailX+8), float32(y))
		tail.LineTo(float32(tailX), float32(y-8))
	} else {
		tail.MoveTo(float32(tailX-8), float32(y+bh))
		tail.LineTo(float32(tailX+8), float32(y+bh))
		tail.LineTo(float32(tailX), float32(y+bh+8))
	}
	tail.Close()
	q.fillPath(&tail, colA(cBrown, k), nil)
	q.text(pl.say, f, cx, base(f, y+9, 21.6), colA(cInk, k), alignC)
}

/* ---------------- info boxes ---------------- */

func (p pen) infoBox(x, y, w, h float64) {
	p.soft(x, y+10, w, h, 10, 24, 0.35)
	p.box(x, y, w, h, 10, 3, cPaper, cBrown, 4, 0)
}

func (g *game) drawHandBox(p pen) {
	lines := []string{"Spades ♠ not broken"}
	if g.broken {
		lines[0] = "Spades ♠ broken"
	}
	nb := 0
	for _, b := range g.bids {
		if b != noBid {
			nb++
		}
	}
	if nb < 4 {
		lines = append(lines, "Bidding…")
	} else {
		lines = append(lines, "Trick "+itoa(min(13, len(g.history)+1))+" of 13")
	}
	f := italic(13, 600)
	cw := 0.0
	for _, l := range lines {
		cw = math.Max(cw, richWidth(l, f))
	}
	w := math.Max(140, cw+30)
	h := 3 + 8 + 16.2 + 46 + 17.55*float64(len(lines)) + 10 + 3
	x, y := 16.0, 14.0
	p.infoBox(x, y, w, h)
	cx := x + w/2
	ty := y + 11
	p.spaced("HAND "+itoa(g.handNo+1), slab(12), cx, base(slab(12), ty, 16.2), 0.1, cMuted, alignC)
	ty += 16.2
	clr := cMuted
	if g.broken {
		clr = cBarn
	}
	p.symbol('♠', cx, ty+23, 40, clr)
	ty += 46
	for _, l := range lines {
		p.rich(l, f, cx, base(f, ty, 17.55), cSub, alignC)
		ty += 17.55
	}
}

func (g *game) drawScoreBox(p pen) {
	lbl := "SCORE · TO " + itoa(g.rules.Target)
	lblW := spacedWidth(lbl, slab(12), 0.1)
	v := g.viewer
	type row struct {
		nm, sub string
		pts     int
		me      bool
	}
	var rows []row
	for k := 0; k < 2; k++ {
		t := (team(v) + k) % 2
		bid, took, any := g.teamBid(t)
		sub := "Bags " + itoa(g.bags[t])
		if any {
			sub = "Bid " + itoa(bid) + " · Took " + itoa(took) + " · " + sub
			for _, i := range []int{t, t + 2} {
				if g.bids[i] == sp.Nil {
					sub += " · " + g.Who(i, v) + " Nil"
					if g.tricks[i] > 0 {
						sub += " ✗"
					}
				}
			}
		}
		rows = append(rows, row{g.teamName(t, v), sub, g.score[t], t == team(v)})
	}
	nf, sf, hf := slab(16), slab(22), italic(12.5, 600)
	inner := lblW
	for _, r := range rows {
		inner = math.Max(inner, math.Max(measure(r.nm, nf), richWidth(r.sub, hf))+16+measure("-200", sf))
	}
	w := math.Max(240, inner+30)
	const rh = 44.0
	h := 3 + 8 + 16.2 + 4 + rh*2 + 8 + 3
	x, y := g.Wd-16-w, 14.0
	p.infoBox(x, y, w, h)
	lx, rx := x+15, x+w-15
	ty := y + 11
	p.spaced(lbl, slab(12), lx, base(slab(12), ty, 16.2), 0.1, cMuted, alignL)
	ty += 16.2 + 4
	for i, r := range rows {
		if i > 0 {
			p.dashH(lx, rx, ty, color.NRGBA{90, 52, 23, 70})
		}
		col := cInk
		if r.me {
			col = cBarn2
		}
		p.text(r.nm, nf, lx, base(nf, ty+2, 22), col, alignL)
		p.rich(r.sub, hf, lx, base(hf, ty+22, 18), cSub, alignL)
		p.text(itoa(r.pts), sf, rx, base(sf, ty, rh)+1, cInk, alignR)
		ty += rh
	}
}

func (g *game) drawPrompt(p pen) {
	u := &g.ui
	if u.promptA <= 0 || u.promptText == "" {
		return
	}
	f := body(19, 700)
	w := richWidth(u.promptText, f) + 32
	h := 4 + 25.65 + 5
	bottom := g.Ht - (g.ch*1.32 + 20)
	x, y := g.Wd/2-w/2, bottom-h
	a := u.promptA
	p.rr(x, y, w, h, 20, color.NRGBA{20, 10, 6, u8(0.66 * a)})
	b := base(f, y+4, 25.65)
	p.rich(u.promptText, f, g.Wd/2, b+2, color.NRGBA{0, 0, 0, u8(0.6 * a)}, alignC)
	p.rich(u.promptText, f, g.Wd/2, b, colA(cWhite, a), alignC)
}

/* ---------------- the pass box ---------------- */

func (g *game) drawAsk(p pen) {
	a := g.ask
	qF, sF := body(17, 700), italic(13, 400)
	n := 0
	for _, c := range a.all {
		if c.sel {
			n++
		}
	}
	sub := a.sub
	if a.need > 0 {
		sub = fmt.Sprintf("%s  (%d of %d chosen)", a.sub, n, a.need)
	}
	rowW, rowH := 0.0, 38.4
	for _, b := range a.btns {
		rowW += 4 + 40 + measure(b.Label, slab(16)) + 8
	}
	rowW -= 8
	inner := math.Max(rowW, math.Max(richWidth(a.q, qF), measure(sub, sF)))
	w := inner + 32 + 6
	h := 3 + 10 + 22.95 + 17.55 + 8 + rowH + 12 + 3
	x := g.Wd/2 - w/2
	y := g.Ht - (g.ch*1.32 + 16) - h
	p.soft(x, y+14, w, h, 12, 30, 0.35)
	p.box(x, y, w, h, 12, 3, cPaper, cBrown, 5, 0)
	cx := x + w/2
	ty := y + 13
	p.rich(a.q, qF, cx, base(qF, ty, 22.95), cInk, alignC)
	ty += 22.95
	p.text(sub, sF, cx, base(sF, ty, 17.55), cSub2, alignC)
	ty += 17.55 + 8
	bx := cx - rowW/2
	ready := a.need == 0 || n == a.need
	for i, b := range a.btns {
		bw := 4 + 40 + measure(b.Label, slab(16))
		st := stBtn
		if !ready {
			st = stOff
		}
		id := "ask:" + itoa(i)
		dy, _ := g.button(p, id, bx, ty, bw, rowH, st)
		p.text(b.Label, slab(16), bx+bw/2, base(slab(16), ty+dy+2, rowH-4), st.ink, alignC)
		bx += bw + 8
	}
}

/* ---------------- effects & banner ---------------- */

func (g *game) drawFx(p pen, ms float64) {
	u := &g.ui
	// bubbles fizzing up in the soda
	sx, sy := 10+ms*0.47, g.Ht-ms-6+ms*0.47
	for j := 0; j < 7; j++ {
		ph := math.Mod(u.steamT/1400+float64(j)*0.37, 1)
		a := math.Sin(ph * math.Pi)
		r := ms * 0.2 * ph
		ang := float64(j) * 2.39
		p.circle(sx+math.Cos(ang)*r, sy+math.Sin(ang)*r, 1.2+1.4*ph, color.NRGBA{255, 255, 255, u8(0.55 * a)})
	}
	for _, q := range g.parts {
		switch q.k {
		case 0:
			al := math.Min(1, q.life*2)
			var path vector.Path
			heartPath(&path, 0, 0, float32(q.s), false)
			var m ebiten.GeoM
			m.Rotate(q.rot)
			m.Translate(q.x, q.y)
			p.fillPath(&path, colA(hexc("d8344a", 1), al), &m)
		case 1:
			p.circle(q.x, q.y, q.s, colA(cGold, math.Max(0, q.life)))
		case 2:
			p.rotRect(q.x, q.y, q.s, q.s/2, q.rot, confettiCols[q.col])
		}
	}
}

func (g *game) drawBanner(p pen, now time.Time) {
	b := g.ban
	if b.title == "" {
		return
	}
	var k, a float64 // scale progress and opacity
	if now.Before(b.until) {
		k = easeBounce(now.Sub(b.at).Seconds() / 0.35)
		a = math.Min(1, now.Sub(b.at).Seconds()/0.25)
	} else {
		t := now.Sub(b.until).Seconds()
		if t > 0.35 {
			return
		}
		k = 1 - easeBounce(t/0.35)
		a = math.Max(0, 1-t/0.25)
	}
	if a <= 0 {
		return
	}
	tf, sf := dispI(46), italic(18, 700)
	w := math.Max(richWidth(b.title, tf), measure(b.small, sf)) + 60 + 8
	h := 4 + 8 + 55.2 + 12 + 4
	if b.small != "" {
		h += 24.3
	}
	fill, outline := cBarn, cMaroon
	if b.blue {
		fill, outline = cDenim, hexc("121e36", 1)
	}
	const m = 60 // margin for shadow and outline
	u := &g.ui
	iw, ih := int(math.Ceil((w+2*m)*u.ds)), int(math.Ceil((h+2*m)*u.ds))
	if u.off == nil || u.off.Bounds().Dx() < iw || u.off.Bounds().Dy() < ih {
		if u.off != nil {
			u.off.Deallocate()
		}
		u.off = ebiten.NewImage(iw+64, ih+64)
	}
	u.off.Clear()
	q := pen{dst: u.off, s: u.ds, ox: m, oy: m}
	q.soft(0, 16, w, h, 10, 40, 0.5)
	q.rr(-3, -3, w+6, h+6, 13, outline)
	q.rr(0, 0, w, h, 10, hexc("e8c35a", 1))
	q.rr(4, 4, w-8, h-8, 7, fill)
	tb := base(tf, 12, 55.2)
	q.rich(b.title, tf, w/2, tb+3, outline, alignC)
	q.rich(b.title, tf, w/2, tb, cWhite, alignC)
	if b.small != "" {
		q.text(b.small, sf, w/2, base(sf, 12+55.2, 24.3), hexc("fff2d0", 1), alignC)
	}
	sub := u.off.SubImage(image.Rect(0, 0, iw, ih)).(*ebiten.Image)
	sc := 0.6 + 0.4*k
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Translate(-float64(iw)/2, -float64(ih)/2)
	op.GeoM.Scale(sc, sc)
	op.GeoM.Rotate(-3 * math.Pi / 180)
	op.GeoM.Translate((g.Wd/2+p.ox)*p.s, (g.Ht*0.4+p.oy)*p.s)
	op.ColorScale.ScaleAlpha(float32(a))
	p.dst.DrawImage(sub, op)
}

/* ---------------- the top bar: a brick parapet ---------------- */

func (g *game) drawBar(p pen) {
	u := &g.ui
	W := u.winW
	for i := 0; i < 14; i++ {
		p.rect(0, barH+float64(i), W, 1, color.NRGBA{0, 0, 0, u8(0.5 * math.Pow(1-float64(i)/14, 2) * 0.6)})
	}
	p.rect(0, 0, W, barH-4, hexc("4a2a22", 1)) // mortar
	const brW, brH = 44.0, 13.0
	for row, y := 0, 0.0; y < barH-10; row, y = row+1, y+brH+2 {
		off := float64(row%2) * brW / 2
		for x, k := -off, 0; x < W; x, k = x+brW+2, k+1 {
			c := []color.NRGBA{hexc("9a3a28", 1), hexc("8a3222", 1), hexc("a8442e", 1), hexc("7e2e20", 1)}[(k*7+row*3)%4]
			p.rect(x, y+1, brW, brH, c)
		}
	}
	p.rect(0, barH-10, W, 6, hexc("8a8078", 1)) // coping stone
	p.vgrad(0, 0, W, barH-4, color.NRGBA{255, 255, 255, 28}, color.NRGBA{0, 0, 0, 60})
	p.rect(0, barH-6, W, 2, color.NRGBA{0, 0, 0, 50})
	mid := (barH - 5) / 2
	narrow := W < 1100
	// a dark plaque behind the title and buttons so they read on the stripes
	x := 14.0
	tf := dispI(26)
	tb := base(tf, mid-13, 26)
	bf := body(14, 700)
	bh := 2 + 4 + 18.9 + 4 + 2
	by := mid - bh/2 - 1
	titleW := spacedWidth("Spades", tf, 0.02)
	plaque := func(x0, x1 float64) {
		p.rr(x0, 5, x1-x0, barH-14, 9, color.NRGBA{30, 14, 10, 215})
		p.rrStroke(x0, 5, x1-x0, barH-14, 9, 1.5, hexc("d9b45a", 0.8))
	}
	// measure the left group first
	lw := titleW + 10
	if !narrow {
		lw += measure("Up on the roof", italic(13, 600)) + 10
	}
	btnW := func(label string) float64 { return richWidth(label, bf) + 24 + 4 }
	for _, l := range []string{"New Game", "Menu", "Rules"} {
		lw += btnW(l) + 8
	}
	lw += 16 + btnW("♫ Music") + 8 + 90 + 8 + btnW("Sounds")
	plaque(x-8, x+lw+8)
	p.spaced("Spades", tf, x, tb+2, 0.02, hexc("1a0a06", 1), alignL)
	p.spaced("Spades", tf, x, tb, 0.02, hexc("ffd88a", 1), alignL)
	x += titleW
	if !narrow {
		sf := italic(13, 600)
		x += 10
		p.text("Up on the roof", sf, x, tb-1, hexc("e8d8c8", 1), alignL)
		x += measure("Up on the roof", sf)
	}
	x += 10
	barBtn := func(id, label string, off bool) {
		w := btnW(label)
		dy, _ := g.button(p, id, x, by, w, bh, stBar)
		ink := cInk
		if off {
			ink = colA(cInk, 0.5)
		}
		b := base(bf, by+dy+2, bh-4)
		p.rich(label, bf, x+w/2, b, ink, alignC)
		if off {
			lw := richWidth(label, bf)
			p.rect(x+w/2-lw/2, b-4.5, lw, 1.3, ink)
		}
		x += w + 8
	}
	barBtn("new", "New Game", false)
	barBtn("menu", "Menu", false)
	barBtn("rules", "Rules", false)
	x += 8
	barBtn("music", "♫ Music", !g.save.Settings.Music)
	// volume slider
	u.volX, u.volW = x+8, 74
	ty := mid
	p.rr(x, ty-3, 90, 6, 3, cBrown)
	p.rr(x+1, ty-2, 88, 4, 2, hexc("fff6e2", 1))
	f := float64(g.save.Settings.Vol) / 100
	tx := u.volX + u.volW*f
	p.rr(x+1, ty-2, tx-x-1, 4, 2, cBarn)
	p.circle(tx, ty, 8, hexc("d9b45a", 1))
	p.circle(tx, ty, 6.5, cBarn)
	g.addHit(hit{id: "vol", x: x, y: ty - 12, w: 90, h: 24, ptr: true})
	x += 90 + 8
	barBtn("sfx", "Sounds", !g.save.Settings.Sfx)

	// stats on the right
	st := g.save.Stats
	lf, nf := body(13, 700), slab(17)
	items := []struct {
		l string
		n int
	}{{"Games", st.Played}, {"Won", st.Won}, {"Nils", st.NilsMade}}
	sw := 0.0
	for i, it := range items {
		if i > 0 {
			sw += 12
		}
		sw += measure(it.l+" ", lf) + measure(itoa(it.n), nf)
	}
	sx := W - 14 - sw
	plaque(sx-10, W-6)
	sb := base(lf, mid-9, 18)
	for _, it := range items {
		p.text(it.l+" ", lf, sx, sb, hexc("e8d8b8", 1), alignL)
		sx += measure(it.l+" ", lf)
		p.text(itoa(it.n), nf, sx, sb, hexc("f2d68e", 1), alignL)
		sx += measure(itoa(it.n), nf) + 12
	}
	if !narrow && g.save.Settings.Music && g.started {
		nf := italic(13, 600)
		s := "♫  " + snd.tuneName()
		room := W - 14 - sw - 30 - x
		if room > 60 {
			for r := []rune(s); richWidth(s, nf) > room-24 && len(r) > 5; {
				r = r[:len(r)-1]
				s = strings.TrimRight(string(r), " ") + "…"
			}
			mx := x + room/2
			tw := richWidth(s, nf)
			p.rr(mx-tw/2-12, 9, tw+24, barH-22, 8, color.NRGBA{30, 14, 10, 190})
			p.rich(s, nf, mx, sb, hexc("e8d8b8", 1), alignC)
		}
	}
}

// showFaces draws a sheet of random regulars instead of the game (art checks only).
var showFaces bool

func (g *game) drawFaces(p pen) {
	u := &g.ui
	p.rect(0, 0, u.winW, u.winH, hexc("2a160e", 1))
	r := rand.New(rand.NewPCG(99, 7))
	sz := 120.0
	cols := int(u.winW / (sz + 30))
	for i := 0; i < cols*int(u.winH/(sz+44)); i++ {
		f := Face{Female: i%2 == 0, Seed: r.Uint64()}
		x, y := 15+float64(i%cols)*(sz+30), 10+float64(i/cols)*(sz+44)
		p.cameo(f.String(), x+sz/2, y+sz/2, sz, nil)
		_, full, _ := f.nameFor(map[string]bool{})
		p.text(full, body(12, 700), x+sz/2, y+sz+16, cCream, alignC)
	}
}
