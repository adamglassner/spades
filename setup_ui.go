//go:build !headless

// The Play Together screens: Same Computer or Online, seating, host/join,
// the lobby, "pass the computer" and messages. Also the text fields and
// face pickers they use.
package main

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

func (g *game) anyPanel() bool {
	return g.showTitle || g.showRules || g.showCredits || g.showResult || g.showTogether || g.showHotseat ||
		g.showOnline || g.showLobby || g.passCh != nil || g.message != ""
}

// newGameKey is the New Game button and F2.
func (g *game) newGameKey() {
	if g.mode == ModeSolo || !g.started && g.mode != ModeHost && g.mode != ModeClient {
		g.goPlay()
		return
	}
	g.newDeal()
}

// setupClick handles the Play Together screens' buttons.
func (g *game) setupClick(id string) bool {
	st := &g.setup
	switch {
	case id == "together":
		g.showTitle, g.showTogether = false, true
	case id == "togBack":
		g.showTogether, g.showTitle = false, true
	case id == "optHot":
		g.showTogether, g.showHotseat = false, true
	case id == "optNet":
		g.showTogether, g.showOnline = false, true
	case id == "hotBack" || id == "onBack":
		g.showHotseat, g.showOnline, g.showTogether, g.focus = false, false, true, ""
	case id == "hotGo":
		g.startHotseat()
	case id == "hostGo":
		g.keepSetup()
		g.showTitle = false
		g.hostGame()
	case id == "joinGo":
		g.keepSetup()
		g.joinGame(st.addr)
	case id == "lobbyGo":
		g.startAudio()
		g.startHosted()
	case id == "lobbyClose":
		g.leaveOnline()
		g.closePanels()
		g.toMenu()
	case id == "passGo":
		if g.passCh != nil {
			snd.play("tick")
			select {
			case g.passCh <- struct{}{}:
			default:
			}
		}
	case id == "msgOK":
		if g.client != nil && g.client.conn == nil { // still connecting: give up
			g.leaveOnline()
		}
		g.message = ""
		if !g.anyPanel() && !g.started {
			g.showTitle = true
		}
	case strings.HasPrefix(id, "hotn="):
		st.hotN, _ = strconv.Atoi(id[5:])
		g.keepSetup()
		snd.play("tick")
	case strings.HasPrefix(id, "face:"):
		parts := strings.SplitN(id, ":", 3)
		if v := facePtr(st, parts[1]); v != nil {
			*v = parts[2]
			g.keepSetup()
			snd.play("tick")
		}
	case strings.HasPrefix(id, "field:"):
		g.focus = id[6:]
	default:
		return false
	}
	return true
}

func facePtr(st *setupState, grp string) *string {
	if grp == "me" {
		return &st.face
	}
	if strings.HasPrefix(grp, "hot") {
		if i, err := strconv.Atoi(grp[3:]); err == nil && i >= 0 && i < 4 {
			return &st.hotFaces[i]
		}
	}
	return nil
}

func fieldPtr(st *setupState, id string) *string {
	switch id {
	case "name":
		return &st.name
	case "addr":
		return &st.addr
	}
	if strings.HasPrefix(id, "hot") {
		if i, err := strconv.Atoi(id[3:]); err == nil && i >= 0 && i < 4 {
			return &st.hotNames[i]
		}
	}
	return nil
}

/* ---------------- text fields and face pickers ---------------- */

// typeText edits the focused field.
func (g *game) typeText() {
	v := fieldPtr(&g.setup, g.focus)
	if v == nil {
		g.focus = ""
		return
	}
	limit := 14
	if g.focus == "addr" {
		limit = 60
	}
	changed := false
	for _, r := range ebiten.AppendInputChars(nil) {
		if r >= ' ' && len([]rune(*v)) < limit {
			*v += string(r)
			changed = true
		}
	}
	if d := inpututil.KeyPressDuration(ebiten.KeyBackspace); d == 1 || (d > 25 && d%3 == 0) {
		if rs := []rune(*v); len(rs) > 0 {
			*v = string(rs[:len(rs)-1])
			changed = true
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter) || inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		g.focus = ""
	}
	if changed {
		g.keepSetup()
	}
}

func (g *game) field(c *col, id string, x, y, w float64, placeholder string) {
	if !c.draw {
		return
	}
	val := fieldPtr(&g.setup, id)
	const h = 36.0
	p := c.p
	on := g.focus == id
	edge := colA(cBrown, 0.55)
	if on {
		edge = cBarn
	}
	p.rr(x, y, w, h, 7, edge)
	p.rr(x+1.5, y+1.5, w-3, h-3, 5.5, hexc("fffdf5", 1))
	f := body(16, 400)
	b := base(f, y, h)
	txt := *val
	if txt == "" && !on {
		p.text(placeholder, f, x+12, b, colA(cInk, 0.4), alignL)
	} else {
		for measure(txt, f) > w-26 && txt != "" {
			txt = string([]rune(txt)[1:])
		}
		p.text(txt, f, x+12, b, cInk, alignL)
	}
	if on && time.Now().UnixMilli()/530%2 == 0 {
		p.rect(x+12+measure(txt, f)+1, y+8, 1.5, h-16, cInk)
	}
	g.addHit(hit{id: "field:" + id, x: x, y: y, w: w, h: h, ptr: true, clip: c.clip})
}

func facePickerW(d float64) float64 { return float64(len(pickerFaces))*(d+6) - 6 }

func (g *game) facePicker(c *col, grp string, x, y, d float64) {
	if !c.draw {
		return
	}
	val := facePtr(&g.setup, grp)
	for i, f := range pickerFaces {
		cx, cy := x+float64(i)*(d+6)+d/2, y+d/2
		id := "face:" + grp + ":" + f
		switch {
		case *val == f:
			c.p.circle(cx, cy, d/2+4, cGold)
		case g.ui.hover == id:
			c.p.circle(cx, cy, d/2+3, colA(cBarn, 0.45))
		}
		c.p.cameo(f, cx, cy, d, nil)
		g.addHit(hit{id: id, x: cx - d/2, y: y, w: d, h: d, ptr: true, clip: c.clip})
	}
}

// chips draws a segmented control; vals are (id, label) pairs.
func (g *game) chips(c *col, cx, y float64, cur string, vals [][2]string) float64 {
	f := body(15, 700)
	const h = 2 + 5 + 20.25 + 5 + 2
	w := 4.0 + 2*float64(len(vals)-1)
	for _, v := range vals {
		w += measure(v[1], f) + 26
	}
	if !c.draw {
		return h
	}
	x := cx - w/2
	c.p.rr(x, y, w, h, 8, cBrown)
	bx := x + 2
	for j, v := range vals {
		bw := measure(v[1], f) + 26
		on := v[0] == cur
		fill, ink := hexc("fffdf5", 1), cInk
		if on {
			fill, ink = cBarn, cWhite
		} else if g.ui.hover == v[0] {
			fill = cWhite
		}
		r := 0.0
		if j == 0 || j == len(vals)-1 {
			r = 6
		}
		c.p.rr(bx, y+2, bw, h-4, r, fill)
		if j > 0 {
			c.p.rect(bx, y+2, bw/2, h-4, fill)
		}
		if j < len(vals)-1 {
			c.p.rect(bx+bw/2, y+2, bw/2, h-4, fill)
		}
		c.p.text(v[1], f, bx+bw/2, base(f, y+2+5, 20.25), ink, alignC)
		g.addHit(hit{id: v[0], x: bx, y: y, w: bw, h: h, ptr: true, clip: c.clip})
		bx += bw + 2
	}
	return h
}

func (c *col) heading(kick, title string) {
	c.y += 6
	c.spacedLine(kick, slab(13), 0.14, cMuted, 15.6)
	c.y += 6
	if c.draw {
		b := base(slab(44), c.y, 48)
		c.p.text(title, slab(44), c.cx(), b+3, cMaroon, alignC)
		c.p.text(title, slab(44), c.cx(), b, cBarn, alignC)
	}
	c.y += 48 + 12
}

/* ---------------- panels ---------------- */

func (g *game) drawTogether(p pen) {
	g.panel(p, 780, func(c *col) {
		c.heading("PLAY TOGETHER", "Who's Playing?")
		cw := (c.w - 20) / 2
		const h = 176.0
		opts := []struct{ id, title, desc string }{
			{"optHot", "Same Computer", "Two to four of you take turns at this computer. Cards turn face down between turns, so pass the mouse and look away."},
			{"optNet", "Online", "Host a game or join a friend's, over your home network or the internet. Everyone plays on their own computer."},
		}
		if c.draw {
			for i, o := range opts {
				x := c.x + float64(i)*(cw+20)
				hot := g.ui.hover == o.id
				fill := hexc("f3e6c6", 1)
				if hot {
					fill = hexc("f8eed4", 1)
				}
				c.p.rr(x, c.y+3, cw, h, 12, colA(cBrown, 0.5))
				c.p.rr(x, c.y, cw, h, 12, cBrown)
				c.p.rr(x+2, c.y+2, cw-4, h-4, 10, fill)
				if hot {
					c.p.rrStroke(x+2, c.y+2, cw-4, h-4, 10, 2, cBarn)
				}
				tf := slab(24)
				c.p.text(o.title, tf, x+cw/2, base(tf, c.y+18, 32), cBarn, alignC)
				df := body(15, 400)
				ty := c.y + 62
				for _, l := range wrap(o.desc, df, cw-44) {
					c.p.text(l, df, x+cw/2, base(df, ty, 21), cSub, alignC)
					ty += 21
				}
				g.addHit(hit{id: o.id, x: x, y: c.y, w: cw, h: h, ptr: true, clip: c.clip})
			}
		}
		c.y += h + 22
		c.buttonRow(slab(20), 7, []rowBtn{{"togBack", "Back", stGhost}})
	})
}

func (g *game) hotSummary() string {
	st := &g.setup
	names := [4]string{"a regular", "a regular", "a regular", "a regular"}
	for k, s := range st.hotSeats() {
		names[s] = cleanName(st.hotNames[k], fmt.Sprintf("Player %d", k+1))
	}
	return "Round the table:  " + strings.Join(names[:], ", ")
}

func (g *game) drawHotseat(p pen) {
	st := &g.setup
	g.panel(p, 780, func(c *col) {
		c.heading("SAME COMPUTER", "Pull Up a Chair")
		lf := body(15, 700)
		c.line("How many people?", italic(14, 700), cSub2, 19, alignC)
		c.y += 4
		var opts [][2]string
		for n := 2; n <= 4; n++ {
			opts = append(opts, [2]string{"hotn=" + itoa(n), itoa(n) + " people"})
		}
		c.y += g.chips(c, c.cx(), c.y, "hotn="+itoa(st.hotN), opts) + 18
		const d = 32.0
		rowW := 78 + 170 + 14 + facePickerW(d)
		x0 := c.cx() - rowW/2
		for i := 0; i < st.hotN; i++ {
			if c.draw {
				c.p.text(fmt.Sprintf("Player %d", i+1), lf, x0, base(lf, c.y, 36), cSub, alignL)
			}
			g.field(c, "hot"+itoa(i), x0+78, c.y, 170, "Name")
			g.facePicker(c, "hot"+itoa(i), x0+78+170+14, c.y+2, d)
			c.y += 36 + 12
		}
		c.y += 6
		c.line(g.hotSummary(), body(15, 700), cInk, 21, alignC)
		left := 4 - st.hotN
		note := "Every seat is taken by a person."
		if left > 0 {
			note = map[int]string{1: "The other seat goes to a rooftop regular (a computer player).", 2: "The other two seats go to rooftop regulars (computer players)."}[left]
		}
		c.line(note, italic(14, 400), cSub2, 19, alignC)
		c.y += 16
		c.buttonRow(slab(24), 9, []rowBtn{{"hotGo", "Deal", stBig}, {"hotBack", "Back", stGhost}})
	})
}

func (g *game) drawOnline(p pen) {
	g.panel(p, 800, func(c *col) {
		c.heading("ONLINE", "Play With Friends")
		lf := body(15, 700)
		const d = 34.0
		rowW := 96 + facePickerW(d)
		x0 := c.cx() - rowW/2
		if c.draw {
			c.p.text("Your name", lf, x0, base(lf, c.y, 36), cSub, alignL)
		}
		g.field(c, "name", x0+96, c.y, 220, "Name")
		c.y += 36 + 12
		if c.draw {
			c.p.text("Your face", lf, x0, base(lf, c.y, d+4), cSub, alignL)
		}
		g.facePicker(c, "me", x0+96, c.y+2, d)
		c.y += d + 4 + 20
		cw := (c.w - 20) / 2
		const h = 176.0
		if c.draw {
			tf, df := slab(22), body(14, 400)
			for i := 0; i < 2; i++ {
				x := c.x + float64(i)*(cw+20)
				c.p.rr(x, c.y, cw, h, 12, cBrown)
				c.p.rr(x+2, c.y+2, cw-4, h-4, 10, hexc("f3e6c6", 1))
			}
			x := c.x
			c.p.text("Host a Game", tf, x+cw/2, base(tf, c.y+14, 30), cBarn, alignC)
			ty := c.y + 52
			for _, l := range wrap(fmt.Sprintf("You deal. Friends join with this computer's address (port %d).", netPort), df, cw-40) {
				c.p.text(l, df, x+cw/2, base(df, ty, 20), cSub, alignC)
				ty += 20
			}
			x = c.x + cw + 20
			c.p.text("Join a Game", tf, x+cw/2, base(tf, c.y+14, 30), cBarn, alignC)
		}
		g.field(c, "addr", c.x+cw+20+20, c.y+56, cw-40, "Host's address, e.g. 192.168.1.20")
		bf := slab(20)
		bh := 3*2 + 7*2 + bf.size*1.2
		for i, b := range []rowBtn{{"hostGo", "Host", stBig}, {"joinGo", "Join", stBig}} {
			bw := measure(b.label, bf) + 56 + 6
			x := c.x + float64(i)*(cw+20) + (cw-bw)/2
			if c.draw {
				dy, _ := g.button(c.p, b.id, x, c.y+h-bh-22, bw, bh, b.st)
				g.ui.hits[len(g.ui.hits)-1].clip = c.clip
				c.p.text(b.label, bf, x+bw/2, base(bf, c.y+h-bh-22+dy+3+7, bf.size*1.2), b.st.ink, alignC)
			}
		}
		c.y += h + 14
		c.para(fmt.Sprintf("Playing across the internet? The host may need to forward port %d on their router. Windows may ask whether to let Spades through the firewall: allow it.", netPort), body(13, 400), hexc("7a6048", 1), 19, alignC)
		c.y += 12
		c.buttonRow(slab(20), 7, []rowBtn{{"onBack", "Back", stGhost}})
	})
}

func (g *game) drawLobby(p pen) {
	host := g.mode == ModeHost
	g.panel(p, 700, func(c *col) {
		c.heading("ONLINE", "Waiting at the Table")
		if host && g.server != nil {
			msg := "Friends join with this computer's address."
			if len(g.server.addrs) > 0 {
				msg = "Friends join with:  " + strings.Join(g.server.addrs, "  or  ")
			}
			c.para(msg, body(16, 700), cInk, 23, alignC)
		} else {
			c.line("You're in. Waiting for the host to deal…", body(16, 400), cInk, 23, alignC)
		}
		c.y += 12
		rows := g.lobbyRows()
		if len(rows) != 4 {
			return
		}
		for i := 0; i < 4; i++ {
			s := rows[i]
			const rh = 54.0
			if c.draw {
				c.p.rr(c.x, c.y, c.w, rh, 9, color.NRGBA{255, 255, 255, 110})
				c.p.rrStroke(c.x, c.y, c.w, rh, 9, 1, colA(cBrown, 0.25))
				if faceOK(s.Face) {
					c.p.cameo(s.Face, c.x+10+21, c.y+rh/2, 42, nil)
				}
				name, status := s.Name, "Rooftop regular · seat open for a friend"
				switch {
				case i == 0 && host:
					status = "You · hosting"
				case i == g.viewer && !host:
					status = "You"
				case i == 0:
					status = "Host"
				case s.Kind != KindAI:
					status = "Joined"
				}
				nf := body(17, 700)
				c.p.text(name, nf, c.x+10+42+14, base(nf, c.y, rh), cInk, alignL)
				sf := italic(14, 400)
				c.p.text(status, sf, c.x+c.w-14, base(sf, c.y, rh), cSub2, alignR)
			}
			c.y += rh + 8
		}
		c.y += 8
		if host {
			c.buttonRow(slab(24), 9, []rowBtn{{"lobbyGo", "Deal", stBig}, {"lobbyClose", "Close", stGhost}})
		} else {
			c.buttonRow(slab(24), 9, []rowBtn{{"leave", "Leave", stGhost}})
		}
	})
}

func (g *game) drawPass(p pen) {
	pl := g.players[g.passTo]
	g.panel(p, 600, func(c *col) {
		c.y += 6
		c.spacedLine("PASS THE COMPUTER TO", slab(13), 0.14, cMuted, 15.6)
		c.y += 16
		if c.draw {
			cx := c.cx()
			c.p.circle(cx, c.y+60, 62, color.NRGBA{0, 0, 0, 40})
			if faceOK(pl.face) {
				c.p.cameo(pl.face, cx, c.y+58, 116, nil)
			}
		}
		c.y += 116 + 12
		if c.draw {
			b := base(slab(52), c.y, 56)
			c.p.text(pl.name, slab(52), c.cx(), b+4, cMaroon, alignC)
			c.p.text(pl.name, slab(52), c.cx(), b, cBarn, alignC)
		}
		c.y += 56 + 4
		c.line("Everyone else, no peeking!", italic(16, 600), cSub2, 22, alignC)
		c.y += 18
		c.buttonRow(slab(22), 9, []rowBtn{{"passGo", "I'm " + pl.name + " · Show My Cards", stBig}})
	})
}

func (g *game) drawMsg(p pen) {
	g.panel(p, 620, func(c *col) {
		c.y += 6
		for i, para := range strings.Split(g.message, "\n\n") {
			if i > 0 {
				c.y += 10
			}
			c.para(para, body(17, 400), cInk, 25, alignC)
		}
		c.y += 22
		label := "OK"
		if g.client != nil && g.client.conn == nil {
			label = "Cancel"
		}
		c.buttonRow(slab(20), 7, []rowBtn{{"msgOK", label, stBig}})
	})
}
