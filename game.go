package main

import (
	"fmt"
	"math"
	"math/rand/v2"
	"strings"
	"sync"
	"time"

	sp "github.com/adamglassner/spades/internal/spades"
)

/* ---------------- who's at the table ---------------- */

// Kind says who plays a seat.
type Kind int

const (
	KindAI     Kind = iota // a computer player
	KindLocal              // someone at this computer
	KindRemote             // someone playing over the network
)

// Mode is how this game is being played.
type Mode int

const (
	ModeSolo    Mode = iota // you and a computer partner against two computer players
	ModeHotseat             // 2-4 people sharing this computer
	ModeHost                // hosting an online game
	ModeClient              // playing in someone else's online game
)

// Seats as drawn: the viewer always sits at the bottom (S), the next player
// to their left (W), their partner across (N) and the last to their right (E).
const (
	seatS = 'S'
	seatW = 'W'
	seatN = 'N'
	seatE = 'E'
)

var seatOrder = [4]byte{seatS, seatW, seatN, seatE}

type player struct {
	i          int
	kind       Kind
	face       string // a Face as text, "" for none
	name, full string // "Ray", "Coach Ray"
	hand       []*cardV
	say        string
	sayAt      time.Time
	sayEnd     time.Time
	turnAt     time.Time
	isTurn     bool
	peer       *peer
}

type play struct {
	p    int
	card *cardV
}

/* ---------------- where a card is ---------------- */

type zoneK int

const (
	zNone zoneK = iota
	zDeck       // the shuffled pile by the dealer
	zHand       // in player P's hand, position N
	zPlay       // played to the trick by P (N = order)
	zWon        // in the tricks P won (M = their trick number, N = card)
)

type zone struct {
	K zoneK   `json:"k"`
	P int     `json:"p,omitempty"`
	N int     `json:"n,omitempty"`
	M int     `json:"m,omitempty"`
	J float64 `json:"j,omitempty"` // a little random rotation
}

type cardV struct {
	c   sp.Card
	id  string // "S14"; "" when this computer doesn't know the face
	idx int
	tok int // what guests call this card (shuffled every hand)
	zn  zone
	// position animation (table coordinates, card centre)
	fx, fy, fr, fs, tx, ty, tr, ts float64
	t0                             time.Time
	dur                            float64
	placed                         bool
	// flip animation: 0 = face up, 180 = face down
	fa0, fa1 float64
	ft0      time.Time
	fdur     float64
	z        int
	speed    int // 0 normal, 1 fast, 2 instant
	pick     bool
	dim      bool
	win      bool
	sel      bool
	fresh    bool
	lift     float64
	sl       float64 // selection lift (eased)
	x, y     float64
}

func (c *cardV) pos(now time.Time) (x, y, r, s float64) {
	if c.dur <= 0 {
		return c.tx, c.ty, c.tr, c.ts
	}
	t := easeCard(now.Sub(c.t0).Seconds() / c.dur)
	l := func(a, b float64) float64 { return a + (b-a)*t }
	return l(c.fx, c.tx), l(c.fy, c.ty), l(c.fr, c.tr), l(c.fs, c.ts)
}

func (c *cardV) angle(now time.Time) float64 {
	if c.fdur <= 0 {
		return c.fa1
	}
	t := easeInOut(now.Sub(c.ft0).Seconds() / c.fdur)
	return c.fa0 + (c.fa1-c.fa0)*t
}

/* ---------------- game state ---------------- */

type askBtn struct {
	Label string `json:"label"`
	Sub   string `json:"sub,omitempty"`
	Cls   string `json:"cls,omitempty"` // "", "nil", "hint"
}

// askT is a question in the box above your hand: your bid.
type askT struct {
	q, sub string
	btns   []askBtn
	need   int
	all    []*cardV
	at     time.Time
	ch     chan askRes
}

type askRes struct {
	btn   askBtn
	idx   int
	cards []*cardV
}

type waitT struct {
	ok  []*cardV
	all []*cardV
	ch  chan *cardV
}

type bannerT struct {
	title, small string
	blue         bool
	at, until    time.Time
}

type particle struct {
	k                        int // 0 spade, 1 spark, 2 confetti
	x, y, vx, vy, life, d, s float64
	rot, vr                  float64
	col                      int
}

// resRow is one team on the end-of-game card.
type resRow struct {
	Name  string   `json:"n"`
	Faces []string `json:"f,omitempty"`
	Score int      `json:"s"`
	Bags  int      `json:"b,omitempty"`
	Place int      `json:"p"`
	Me    bool     `json:"m,omitempty"`
}

type resultT struct {
	kick, title, sub string
	rows             []resRow
	rec              string
	won              bool
}

// tally is one seat's haul this game, for guests' records.
type tally struct{ Hands, Made, Sets, Nils, NilsMade, Bags int }

const noBid = -1

type game struct {
	mu sync.Mutex

	save    saveData
	mode    Mode
	viewer  int  // whose seat the table is drawn from
	hidden  bool // shared computer: hands face down between turns
	players [4]*player
	deck    []*cardV
	byCard  map[sp.Card]*cardV
	byTok   [52]*cardV

	dealer   int
	handNo   int // hands dealt this game (the first is 0)
	trick    []play
	history  [][]sp.Play
	broken   bool
	bids     [4]int // noBid until the seat has bid
	tricks   [4]int // tricks taken this hand
	score    [2]int
	bags     [2]int
	last     [2]int // what each team scored last hand
	rules    sp.Rules
	handsTot int // hands dealt since the program started (for guests' snapshots)

	gameID   int
	cancel   chan struct{}
	finished bool
	started  bool
	wait     *waitT
	ask      *askT
	remote   map[int]*remoteWait
	prompt   string
	promptAt time.Time
	ban      bannerT
	pace     float64
	parts    []particle

	tallies [4]tally

	// layout, in logical pixels (table space)
	Wd, Ht, cw, ch float64

	// overlays
	showTitle, showRules, showCredits, showResult bool
	showTogether, showHotseat, showOnline         bool
	showLobby                                     bool
	passTo                                        int
	passCh                                        chan struct{}
	message                                       string
	result                                        resultT
	resultAt                                      time.Time
	ui                                            uiState

	setup  setupState
	focus  string // text field being typed in
	server *server
	client *client
	lobby  []lobbySeat
	cliEnd chan struct{} // closed to drop a guest's pending ask

	nextFolks []folk // who sits down for the next single-player game

	rng *rand.Rand
}

var errCancel = fmt.Errorf("cancelled")

func newGameState() *game {
	seed := uint64(time.Now().UnixNano())
	g := &game{rng: rand.New(rand.NewPCG(seed, seed>>7^0x5eed)), pace: 1, showTitle: true, remote: map[int]*remoteWait{}}
	g.save = loadSave()
	g.setup = newSetup(&g.save)
	g.rules = g.houseRules()
	g.bids = [4]int{noBid, noBid, noBid, noBid}
	g.resetDeck(false)
	g.soloTable()
	g.nextFolks = nil // the first game seats the neighbors already at the table
	for _, i := range []int{1, 2, 3} {
		p := g.players[i]
		g.nextFolks = append(g.nextFolks, folk{face: p.face, name: p.name, full: p.full})
	}
	g.Wd, g.Ht = 1400, 820
	g.measure()
	return g
}

func (g *game) houseRules() sp.Rules {
	r := sp.DefaultRules
	r.Target = g.save.Settings.Target
	return r
}

// resetDeck makes the 52 cards. A guest's cards start out unknown.
func (g *game) resetDeck(guest bool) {
	g.deck = nil
	g.byCard = map[sp.Card]*cardV{}
	for i, c := range sp.Deck() {
		v := &cardV{c: c, id: c.ID(), idx: i, tok: i, ts: 1, fs: 1, fa1: 180}
		if guest {
			v.id = ""
		}
		g.deck = append(g.deck, v)
		g.byCard[c] = v
		g.byTok[i] = v
	}
}

func (g *game) rand(n int) int      { return g.rng.IntN(n) }
func pickS[T any](g *game, a []T) T { return a[g.rand(len(a))] }

/* pacing */
var paceOf = map[string]float64{"relaxed": 1.35, "normal": 1, "brisk": 0.6}

func (g *game) T(k string) float64 {
	v := map[string]float64{"deal": 45, "move": 420, "play": 360, "think": 600, "trick": 950, "quip": 1900, "banner": 1700}[k]
	v *= g.pace
	if testFast {
		v *= 0.04
	}
	return v
}

// sleep releases the table while waiting; a new game cancels the old one.
func (g *game) sleep(ms float64) {
	id, cancel := g.gameID, g.cancel
	g.mu.Unlock()
	select {
	case <-time.After(time.Duration(ms * float64(time.Millisecond))):
	case <-cancel:
	}
	g.mu.Lock()
	if id != g.gameID {
		panic(errCancel)
	}
}

// waitOn releases the table until ch delivers (or the game is cancelled).
func waitOn[T any](g *game, ch chan T) T {
	id, cancel := g.gameID, g.cancel
	g.mu.Unlock()
	var v T
	select {
	case v = <-ch:
	case <-cancel:
	}
	g.mu.Lock()
	if id != g.gameID {
		panic(errCancel)
	}
	return v
}

/* ---------------- rules helpers ---------------- */

func team(i int) int { return sp.Team(i) }

func (g *game) trickPlays() []sp.Play {
	t := make([]sp.Play, len(g.trick))
	for i, x := range g.trick {
		t[i] = sp.Play{P: x.p, Card: x.card.c}
	}
	return t
}

func cardsOf(h []*cardV) []sp.Card {
	out := make([]sp.Card, len(h))
	for i, c := range h {
		out[i] = c.c
	}
	return out
}

func (g *game) legal(p *player) []*cardV {
	var out []*cardV
	for _, c := range sp.Legal(cardsOf(p.hand), g.trickPlays(), g.broken) {
		out = append(out, g.byCard[c])
	}
	return out
}

// bidsNow is every seat's bid with the ones not made yet as 0.
func (g *game) bidsNow() ([4]int, [4]bool) {
	var b [4]int
	var d [4]bool
	for i, x := range g.bids {
		if x != noBid {
			b[i], d[i] = x, true
		}
	}
	return b, d
}

func (g *game) aiPlay(p *player) *cardV {
	b, _ := g.bidsNow()
	c := sp.ChoosePlay(sp.PlayState{Seat: p.i, Hand: cardsOf(p.hand), Trick: g.trickPlays(), History: g.history, Broken: g.broken, Bids: b, Tricks: g.tricks})
	return g.byCard[c]
}

func (g *game) aiBid(p *player) int {
	b, d := g.bidsNow()
	return sp.ChooseBid(sp.BidState{Seat: p.i, Hand: cardsOf(p.hand), Bids: b, Done: d, Score: g.score, Bags: g.bags, Rules: g.rules})
}

// teamBid is what a team's bidders asked for (nil bids count 0) and what they've taken.
func (g *game) teamBid(t int) (bid, took int, any bool) {
	for _, i := range []int{t, t + 2} {
		if g.bids[i] != noBid {
			any = true
			if g.bids[i] != sp.Nil {
				bid += g.bids[i]
				took += g.tricks[i]
			}
		}
	}
	return
}

func bidWord(b int) string {
	switch b {
	case noBid:
		return "–"
	case sp.Nil:
		return "Nil"
	}
	return itoa(b)
}

/* ---------------- names, as each seat sees them ---------------- */

// who is how player i is named to the person in seat v ("you" or a name).
func (g *game) who(i, v int) string {
	if i == v && g.mode != ModeHotseat {
		return "you"
	}
	return g.players[i].name
}

func (g *game) Who(i, v int) string {
	s := g.who(i, v)
	if s == "you" {
		return "You"
	}
	return s
}

func (g *game) partnerName(i int) string { return g.players[sp.PartnerOf(i)].name }

// teamName is a team as the person in seat v would put it: "You & Ray", "Dot & Sal".
func (g *game) teamName(t, v int) string {
	a, b := t, t+2
	if v == b {
		a, b = b, a
	}
	return g.Who(a, v) + " & " + g.Who(b, v)
}

func (g *game) verb(i, v int, you, they string) string {
	if g.who(i, v) == "you" {
		return you
	}
	return they
}

/* ---------------- layout ---------------- */

func (g *game) measure() {
	g.ch = math.Max(92, math.Min(math.Min(g.Ht*0.2, g.Wd*0.12), 200))
	g.cw = g.ch * cardW / cardH
}

func (g *game) cy() float64 { return g.Ht * 0.46 }

func (g *game) seatOf(i int) byte { return seatOrder[((i-g.viewer)%4+4)%4] }

func (g *game) anchor(seat byte) (float64, float64) {
	switch seat {
	case seatS:
		return g.Wd / 2, g.Ht - g.ch*0.58 - 4
	case seatN:
		return g.Wd / 2, g.ch*0.42 + 2
	case seatW:
		return g.ch*0.5 + 4, g.cy()
	}
	return g.Wd - g.ch*0.5 - 4, g.cy()
}

func (g *game) handPos(seat byte, i, n int) (x, y, r, s float64) {
	ax, ay := g.anchor(seat)
	k := float64(i) - float64(n-1)/2
	switch seat {
	case seatS:
		sp := math.Min(g.cw*0.56, g.Wd*0.5/12)
		return ax + k*sp, ay + k*k*1.1, k * 2.4, 1
	case seatN:
		sp := math.Min(g.cw*0.3, g.Wd*0.34/12)
		return ax - k*sp, ay - k*k*0.5, 180 - k*2, 0.8
	}
	sp, dir := math.Min(g.cw*0.28, g.Ht*0.3/12), 1.0
	if seat == seatE {
		dir = -1
	}
	return ax - dir*k*k*0.5, ay + dir*k*sp, dir*90 + dir*k*2, 0.8
}

func (g *game) playSpot(seat byte) (float64, float64, float64) {
	x, y, d := g.Wd/2, g.cy(), g.ch*0.38
	switch seat {
	case seatS:
		return x, y + d, -3
	case seatN:
		return x, y - d, 183
	case seatW:
		return x - g.cw*0.85, y, 87
	}
	return x + g.cw*0.85, y, -86
}

// zonePos is where a card in a zone sits, as the viewer sees the table.
func (g *game) zonePos(c *cardV) (x, y, r, s float64, up bool, z int) {
	zn := c.zn
	N := float64(zn.N)
	switch zn.K {
	case zDeck:
		dx, dy := g.anchor(g.seatOf(g.dealer))
		sx, sy := (dx+g.Wd/2)/2, (dy+g.cy())/2
		return sx + N*0.12, sy - N*0.2, zn.J, 1, false, 20 + zn.N
	case zHand:
		p := g.players[zn.P]
		x, y, r, s := g.handPos(g.seatOf(zn.P), zn.N, len(p.hand))
		return x, y, r, s, zn.P == g.viewer && !g.hidden, 100 + zn.N
	case zPlay:
		sx, sy, sr := g.playSpot(g.seatOf(zn.P))
		return sx, sy, sr + zn.J, 1, true, 300 + zn.N
	case zWon:
		// each player's tricks in a little fanned row in front of them
		ax, ay := g.anchor(g.seatOf(zn.P))
		tx, ty := g.Wd/2+(ax-g.Wd/2)*0.6, g.cy()+(ay-g.cy())*0.6
		M := float64(zn.M)
		rot := 0.0
		dx, dy := M*g.cw*0.16, 0.0
		switch s := g.seatOf(zn.P); s {
		case seatW, seatE:
			rot = 90
			dx, dy = 0, M*g.cw*0.16
			ty -= g.cw * 1.25
		case seatS:
			tx += g.cw * 1.3
		case seatN:
			tx -= g.cw*1.3 + 12*g.cw*0.16
		}
		return tx + dx + N*0.5, ty + dy - N*0.5, rot + zn.J, 0.5, false, 20 + zn.M*4 + zn.N
	}
	return c.tx, c.ty, c.tr, c.ts, false, c.z
}

// layoutCards moves every card to where its zone says (animated unless instant).
func (g *game) layoutCards(instant bool) {
	for _, c := range g.deck {
		if c.zn.K == zNone {
			continue
		}
		x, y, r, s, up, z := g.zonePos(c)
		sp := c.speed
		if instant {
			c.speed = 2
		}
		if !c.placed || x != c.tx || y != c.ty || r != c.tr || s != c.ts || z != c.z {
			g.place(c, x, y, r, z, s)
		}
		g.flip(c, up && c.id != "")
		c.speed = sp
	}
}

func (g *game) place(c *cardV, x, y, r float64, z int, scale float64) {
	now := time.Now()
	c.fx, c.fy, c.fr, c.fs = c.pos(now)
	c.tx, c.ty, c.tr, c.ts = x, y, r, scale
	c.x, c.y = x, y
	c.t0 = now
	switch c.speed {
	case 0:
		c.dur = 0.42
	case 1:
		c.dur = 0.24
	default:
		c.dur = 0
	}
	if !c.placed { // first placement: appear in place rather than fly in from the corner
		c.dur, c.placed = 0, true
	}
	if z >= 0 {
		c.z = z
	}
}

func (g *game) flip(c *cardV, up bool) {
	now := time.Now()
	target := 180.0
	if up {
		target = 0
	}
	if c.fa1 == target {
		return
	}
	c.fa0, c.fa1, c.ft0 = c.angle(now), target, now
	c.fdur = 0.36
	if c.speed == 2 {
		c.fdur = 0
	}
}

// setHand gives every card in p's hand its zone (after the hand changes).
func (g *game) setHand(p *player) {
	for i, c := range p.hand {
		c.zn = zone{K: zHand, P: p.i, N: i}
	}
}

func (g *game) sortHand(p *player) {
	cs := cardsOf(p.hand)
	sp.SortHand(cs)
	for i, c := range cs {
		p.hand[i] = g.byCard[c]
	}
	g.setHand(p)
}

// sync lays the table out here and sends every guest their view of it.
func (g *game) sync() {
	g.layoutCards(false)
	if g.server != nil {
		g.server.sendSnaps(g, false)
	}
}

func (g *game) setTurn(i int) {
	for _, p := range g.players {
		on := p.i == i
		if on && !p.isTurn {
			p.turnAt = time.Now()
		}
		p.isTurn = on
	}
}

func (g *game) setPrompt(t string) {
	if t != g.prompt {
		g.promptAt = time.Now()
	}
	g.prompt = t
}

func (g *game) banner(title, small string, blue bool, ms float64) {
	if ms == 0 {
		ms = 1700 * g.pace
		if testFast {
			ms *= 0.2
		}
	}
	now := time.Now()
	g.ban = bannerT{title, small, blue, now, now.Add(time.Duration(ms) * time.Millisecond)}
}

/* ---------------- events everyone sees (each worded for their seat) ---------------- */

type event struct {
	K    string  `json:"k"`
	P    int     `json:"p,omitempty"`
	N    int     `json:"n,omitempty"`
	Text string  `json:"text,omitempty"`
	Sub  string  `json:"sub,omitempty"`
	Blue bool    `json:"blue,omitempty"`
	Ms   float64 `json:"ms,omitempty"`
}

// emit shows an event here and sends it to every guest, worded for their seat.
func (g *game) emit(f func(v int) *event) {
	if e := f(g.viewer); e != nil {
		g.apply(e)
	}
	if g.server != nil {
		for _, pr := range g.server.peers {
			if e := f(pr.seat); e != nil {
				pr.send(Msg{T: "ev", Ev: e})
			}
		}
	}
}

func (g *game) emitAll(e event) { g.emit(func(int) *event { c := e; return &c }) }

func (g *game) sound(name string) { g.emitAll(event{K: "snd", Text: name}) }

func (g *game) apply(e *event) {
	switch e.K {
	case "snd":
		snd.play(e.Text)
	case "say":
		if e.P < 0 || e.P > 3 || g.players[e.P] == nil {
			return
		}
		p := g.players[e.P]
		now := time.Now().Add(time.Duration(e.Ms) * time.Millisecond)
		p.say, p.sayAt, p.sayEnd = e.Text, now, now.Add(time.Duration(g.T("quip")+500)*time.Millisecond)
	case "banner":
		g.banner(e.Text, e.Sub, e.Blue, e.Ms)
	case "spark":
		x, y, _ := g.playSpot(g.seatOf(e.P))
		g.burst(x, y, 1, 12)
	case "spades":
		if e.Text == "low" {
			g.burst(g.Wd/2, g.Ht*0.8, 0, e.N)
		} else {
			g.burst(g.Wd/2, g.cy(), 0, e.N)
		}
	case "confetti":
		g.burst(0, 0, 2, e.N)
	case "prompt":
		g.setPrompt(e.Text)
	}
}

// say has a computer player say one of their lines; {p} becomes about's
// name, {n} the number n.
func (g *game) say(p *player, kind string, chance float64, about int, delay float64) {
	g.sayN(p, kind, chance, about, -1, delay)
}

func (g *game) sayN(p *player, kind string, chance float64, about, n int, delay float64) {
	f, ok := parseFace(p.face)
	if p.kind != KindAI || !ok || g.rng.Float64() > chance {
		return
	}
	ls := f.lines(kind)
	if len(ls) == 0 {
		return
	}
	line := pickS(g, ls)
	if about >= 0 && about < 4 {
		nm := g.players[about].name
		if nm == "You" || nm == "" {
			line = strings.ReplaceAll(line, ", {p}", "")
		}
		line = strings.ReplaceAll(line, "{p}", nm)
	}
	line = strings.ReplaceAll(line, "{n}", itoa(n))
	g.emitAll(event{K: "say", P: p.i, Text: line, Ms: delay})
}

/* ---------------- effects ---------------- */

func (g *game) burst(x, y float64, kind int, n int) {
	R := g.rng.Float64
	for i := 0; i < n; i++ {
		switch kind {
		case 0:
			a, v := -math.Pi/2+(R()-.5)*2.2, 3+R()*6
			g.parts = append(g.parts, particle{k: 0, x: x, y: y, vx: math.Cos(a) * v, vy: math.Sin(a) * v, life: 1, d: 0.009, s: 9 + R()*9, rot: (R() - .5) * 0.8, vr: (R() - .5) * .08})
		case 1:
			a, v := R()*math.Pi*2, 1.5+R()*4
			g.parts = append(g.parts, particle{k: 1, x: x, y: y, vx: math.Cos(a) * v, vy: math.Sin(a)*v - 1, life: 1, d: 0.02 + R()*0.02, s: 2 + R()*3})
		case 2:
			g.parts = append(g.parts, particle{k: 2, x: R() * g.Wd, y: -20 - R()*g.Ht*0.5, vx: (R() - .5) * 2, vy: 2 + R()*2.5, life: 1, d: 0.003, s: 6 + R()*6, rot: R() * 6, vr: (R() - .5) * .25, col: g.rand(6)})
		}
	}
}

/* ---------------- decisions: here, over the network, or the computer ---------------- */

func (g *game) chooseCard(ok, all []*cardV, text string) *cardV {
	g.setPrompt(text)
	in := func(c *cardV) bool {
		for _, x := range ok {
			if x == c {
				return true
			}
		}
		return false
	}
	for _, c := range all {
		c.pick, c.dim = in(c), !in(c)
	}
	w := &waitT{ok: ok, all: all, ch: make(chan *cardV, 1)}
	g.wait = w
	defer func() {
		for _, x := range all {
			x.pick, x.dim = false, false
		}
		g.setPrompt("")
	}()
	return waitOn(g, w.ch)
}

func (g *game) askUser(a *askT) askRes {
	a.at, a.ch = time.Now(), make(chan askRes, 1)
	g.ask = a
	return waitOn(g, a.ch)
}

// localTurn is true when someone at this computer decides for seat i.
func (g *game) localTurn(i int) bool { return g.players[i].kind == KindLocal && !autopilot }

// ensureView turns a shared computer over to the player in seat i: a
// "pass the computer" screen while everyone's cards are face down, then the
// table turns to their seat.
func (g *game) ensureView(i int) {
	if g.mode != ModeHotseat || (g.viewer == i && !g.hidden) {
		return
	}
	if g.viewer != i {
		g.hidden = true
		g.sync()
		g.passTo = i
		ch := make(chan struct{}, 1)
		g.passCh = ch
		func() {
			defer func() { g.passCh = nil }()
			waitOn(g, ch)
		}()
		g.viewer = i
	}
	g.hidden = false
	g.sync()
	g.sleep(g.T("move"))
}

func (g *game) hideAfter() {
	if g.mode == ModeHotseat {
		g.hidden = true
		g.sync()
	}
}

func (g *game) decideCard(i int, ok []*cardV, text string, think float64, ai func() *cardV) *cardV {
	p := g.players[i]
	switch {
	case g.localTurn(i):
		g.ensureView(i)
		c := g.chooseCard(ok, p.hand, text)
		g.hideAfter()
		return c
	case p.kind == KindRemote:
		a := &wireAsk{K: "card", Prompt: text}
		for _, c := range ok {
			a.Toks = append(a.Toks, c.tok)
		}
		if r, okk := g.remoteAsk(i, a); okk {
			if c := g.byTok[r.Tok]; c != nil {
				for _, x := range ok {
					if x == c {
						return c
					}
				}
			}
		}
	}
	g.sleep(think)
	return ai()
}

// bidButtons are Nil and 1 to 13.
func bidButtons() []askBtn {
	bs := []askBtn{{Label: "Nil", Cls: "nil"}}
	for n := 1; n <= 13; n++ {
		bs = append(bs, askBtn{Label: itoa(n)})
	}
	return bs
}

// bidQuestion is the bid box for seat i.
func (g *game) bidQuestion(i int) (string, string) {
	pa := sp.PartnerOf(i)
	q := "Your bid: how many tricks will you take?"
	if g.mode == ModeHotseat {
		q = g.players[i].name + ", how many tricks will you take?"
	}
	var bits []string
	if g.bids[pa] != noBid {
		bits = append(bits, g.players[pa].name+" (partner) bid "+bidWord(g.bids[pa]))
	} else {
		bits = append(bits, "Your partner bids after you")
	}
	ob, _, any := g.teamBid(1 - team(i))
	if any {
		bits = append(bits, "they bid "+itoa(ob))
	}
	est := int(sp.Estimate(cardsOf(g.players[i].hand)) + 0.35)
	bits = append(bits, "a fair bid for this hand is about "+itoa(max(1, est)))
	return q, strings.Join(bits, " · ")
}

func (g *game) decideBid(i int) int {
	p := g.players[i]
	btns := bidButtons()
	switch {
	case g.localTurn(i):
		g.ensureView(i)
		q, sub := g.bidQuestion(i)
		g.setPrompt("")
		r := g.askUser(&askT{q: q, sub: sub, btns: btns})
		g.hideAfter()
		return r.idx
	case p.kind == KindRemote:
		q, sub := g.bidQuestion(i)
		if r, ok := g.remoteAsk(i, &wireAsk{K: "bid", Q: q, Sub: sub, Btns: btns}); ok && r.Btn >= 0 && r.Btn <= 13 {
			return r.Btn
		}
	}
	g.sleep(g.T("think") * (0.9 + g.rng.Float64()*0.6))
	return g.aiBid(p)
}

/* ---------------- the neighbors ---------------- */

// soloTable seats you with the computer players the title card shows and picks who comes next time.
func (g *game) soloTable() {
	g.mode, g.viewer, g.hidden = ModeSolo, 0, false
	if len(g.nextFolks) != 3 {
		g.rollNext()
	}
	g.players[0] = &player{i: 0, kind: KindLocal, name: "You", full: "You"}
	for k, seat := range []int{1, 2, 3} {
		f := g.nextFolks[k]
		g.players[seat] = &player{i: seat, kind: KindAI, face: f.face, name: f.name, full: f.full}
	}
	g.rollNext()
}

func (g *game) rollNext() {
	g.nextFolks = randomFolks(g.rng, 3, map[string]bool{}, map[string]bool{"You": true})
}

// fillFolks gives every empty or computer seat a random neighbor.
func (g *game) fillFolks() {
	faces, names := map[string]bool{}, map[string]bool{}
	for _, p := range g.players {
		if p != nil && p.kind != KindAI {
			faces[p.face], names[p.name] = true, true
		}
	}
	var open []int
	for i, p := range g.players {
		if p == nil || p.kind == KindAI {
			open = append(open, i)
		}
	}
	fs := randomFolks(g.rng, len(open), faces, names)
	for k, i := range open {
		f := fs[k%len(fs)]
		g.players[i] = &player{i: i, kind: KindAI, face: f.face, name: f.name, full: f.full}
	}
}

func (g *game) setPlayer(i int, kind Kind, face, name string) *player {
	p := &player{i: i, kind: kind, face: face, name: name, full: name}
	g.players[i] = p
	return p
}

/* ---------------- a game ---------------- */

// newGame is called with the lock held.
func (g *game) newGame() {
	g.gameID++
	if g.cancel != nil {
		close(g.cancel)
	}
	g.cancel = make(chan struct{})
	g.finished, g.started = false, true
	g.pace = paceOf[g.save.Settings.Speed]
	if g.pace == 0 {
		g.pace = 1
	}
	if g.mode == ModeSolo { // new neighbors every game
		g.soloTable()
	}
	g.rules = g.houseRules()
	g.showResult, g.wait, g.ask, g.passCh = false, nil, nil, nil
	g.remote = map[int]*remoteWait{}
	g.prompt = ""
	g.ban = bannerT{}
	g.parts = nil
	for _, p := range g.players {
		p.hand, p.isTurn, p.say = nil, false, ""
	}
	if g.mode == ModeHotseat {
		g.viewer, g.hidden = 0, true
	}
	for _, c := range g.deck {
		c.speed, c.pick, c.dim, c.win, c.sel, c.fresh = 2, false, false, false, false, false
		c.zn = zone{}
	}
	g.score, g.bags, g.last = [2]int{}, [2]int{}, [2]int{}
	g.tricks, g.bids = [4]int{}, [4]int{noBid, noBid, noBid, noBid}
	g.tallies = [4]tally{}
	g.handNo, g.dealer = 0, g.rand(4)
	g.trick, g.history, g.broken = nil, nil, false
	if g.server != nil {
		g.server.open = false
		g.server.sendSnaps(g, true)
	}
	go g.run(g.gameID)
}

func (g *game) run(id int) {
	g.mu.Lock()
	defer func() {
		if r := recover(); r != nil && r != errCancel {
			g.mu.Unlock()
			panic(r)
		}
		g.mu.Unlock()
	}()
	if id != g.gameID {
		return
	}
	for !sp.GameOver(g.score, g.rules) {
		g.playHand()
		g.handNo++
		g.dealer = (g.dealer + 1) % 4
	}
	g.endGame()
}

func (g *game) playHand() {
	g.trick, g.history, g.broken = nil, nil, false
	g.tricks, g.bids = [4]int{}, [4]int{noBid, noBid, noBid, noBid}
	g.handsTot++
	for _, p := range g.players {
		p.hand = nil
	}
	g.setTurn(-1)
	// guests know a card only by a token that changes every hand
	for i, t := range g.rng.Perm(52) {
		g.deck[i].tok = t
		g.byTok[t] = g.deck[i]
	}
	deck := append([]*cardV(nil), g.deck...)
	g.rng.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
	for i, c := range deck {
		c.speed, c.win, c.pick, c.dim, c.sel, c.fresh = 1, false, false, false, false, false
		c.zn = zone{K: zDeck, N: i, J: (g.rng.Float64() - .5) * 4}
	}
	g.sync()
	g.sound("shuffle")
	d := g.dealer
	g.emit(func(v int) *event {
		if g.who(d, v) == "you" {
			return &event{K: "prompt", Text: "You deal…"}
		}
		return &event{K: "prompt", Text: g.players[d].name + " deals…"}
	})
	g.sleep(g.T("move") * 1.6)
	k := 0
	for _, round := range []int{4, 3, 3, 3} {
		for j := 0; j < 4; j++ {
			p := g.players[(g.dealer+1+j)%4]
			for n := 0; n < round; n++ {
				p.hand = append(p.hand, deck[k])
				k++
			}
			g.sortHand(p)
			g.sync()
			g.sound("deal")
			g.sleep(g.T("deal") * 2.2)
		}
	}
	for _, c := range deck {
		c.speed = 0
	}
	g.emitAll(event{K: "prompt"})
	g.sleep(g.T("move"))
	g.bidding()
	g.playTricks()
	g.scoreHand()
}

/* ---------------- bidding ---------------- */

func (g *game) bidding() {
	for k := 1; k <= 4; k++ {
		i := (g.dealer + k) % 4
		p := g.players[i]
		g.setTurn(i)
		g.sync()
		if !g.localTurn(i) {
			g.emit(func(v int) *event {
				return &event{K: "prompt", Text: g.Who(i, v) + " " + g.verb(i, v, "are", "is") + " bidding…"}
			})
		}
		b := g.decideBid(i)
		g.emitAll(event{K: "prompt"})
		g.bids[i] = b
		g.setTurn(-1)
		g.sync()
		if b == sp.Nil {
			g.sound("nil")
		} else {
			g.sound("bid")
		}
		if p.kind == KindAI {
			if b == sp.Nil {
				g.say(p, "nil", 1, -1, 0)
			} else {
				g.sayN(p, "bid", 1, -1, b, 0)
			}
		} else {
			txt := "I'll take " + itoa(b) + "."
			if b == 1 {
				txt = "Put me down for one."
			}
			if b == sp.Nil {
				txt = "Nil!"
			}
			g.emitAll(event{K: "say", P: i, Text: txt})
		}
		g.sleep(g.T("play") * 1.4)
	}
	// the contract, as each side sees it
	g.emit(func(v int) *event {
		us, them := team(v), 1-team(v)
		ub, _, _ := g.teamBid(us)
		tb, _, _ := g.teamBid(them)
		sub := fmt.Sprintf("%s need %d  ·  %s need %d", g.teamName(us, v), ub, g.teamName(them, v), tb)
		for i := 0; i < 4; i++ {
			if g.bids[i] == sp.Nil {
				sub += "  ·  " + g.Who(i, v) + " bid Nil"
			}
		}
		return &event{K: "banner", Text: "The Bids Are In", Sub: sub, Blue: true, Ms: g.T("banner") * 1.1}
	})
	g.sleep(g.T("banner") * 0.9)
}

/* ---------------- tricks ---------------- */

func without(h []*cardV, c *cardV) []*cardV {
	out := make([]*cardV, 0, len(h))
	for _, x := range h {
		if x != c {
			out = append(out, x)
		}
	}
	return out
}

func (g *game) playTricks() {
	leader := (g.dealer + 1) % 4
	for t := 0; t < 13; t++ {
		g.trick = nil
		for j := 0; j < 4; j++ {
			i := (leader + j) % 4
			p := g.players[i]
			g.setTurn(i)
			g.sync()
			L := g.legal(p)
			text := "Your lead"
			switch {
			case len(g.trick) == 0 && !g.broken && len(L) < len(p.hand):
				text = "Your lead · spades aren't broken yet"
			case len(g.trick) > 0 && L[0].c.S == g.trick[0].card.c.S && len(L) < len(p.hand):
				text = "Follow suit: " + g.trick[0].card.c.S.Name()
			case len(g.trick) > 0:
				text = "No " + g.trick[0].card.c.S.Name() + " · trump with a spade or throw off"
			}
			if g.bids[i] == sp.Nil && g.tricks[i] == 0 && g.localTurn(i) {
				text += "  (you bid Nil: don't win it!)"
			}
			c := g.decideCard(i, L, text, g.T("think")*(0.7+g.rng.Float64()*0.5), func() *cardV { return g.aiPlay(p) })
			ok := false
			for _, x := range g.legal(p) {
				ok = ok || x == c
			}
			if !ok {
				panic("illegal play " + c.id)
			}
			if i == g.viewer || g.mode == ModeHotseat {
				for _, x := range p.hand {
					x.fresh = false
				}
			}
			trump := len(g.trick) > 0 && c.c.S == sp.Spades && g.trick[0].card.c.S != sp.Spades
			p.hand = without(p.hand, c)
			g.trick = append(g.trick, play{i, c})
			c.zn = zone{K: zPlay, P: i, N: len(g.trick), J: (g.rng.Float64() - .5) * 8}
			g.setHand(p)
			g.sync()
			switch {
			case c.c.S == sp.Spades && !g.broken:
				g.sound("broken")
			case trump:
				g.sound("trump")
			default:
				g.sound("play")
			}
			if c.c.S == sp.Spades && !g.broken {
				g.broken = true
				g.emitAll(event{K: "banner", Text: "Spades Are Broken!", Sub: "Spades may be led from now on.", Ms: g.T("banner") * 0.8})
				g.say(p, "broken", 0.6, -1, 300)
			} else if trump {
				g.say(p, "trump", 0.25, -1, 200)
			}
			g.sleep(g.T("play"))
		}
		g.setTurn(-1)
		w := sp.Winning(g.trickPlays())
		wp := w.P
		g.tricks[wp]++
		g.history = append(g.history, g.trickPlays())
		leader = wp
		wc := g.byCard[w.Card]
		wc.win = true
		g.sync()
		pw := g.players[wp]
		switch {
		case g.bids[wp] == sp.Nil && g.tricks[wp] == 1:
			g.sound("bust")
			g.emit(func(v int) *event {
				return &event{K: "banner", Text: "Nil Busted!", Sub: g.Who(wp, v) + " " + g.verb(wp, v, "take", "takes") + " a trick after bidding Nil.", Blue: team(wp) != team(v)}
			})
			g.say(pw, "bust", 1, -1, 0)
			for _, o := range []int{(wp + 1) % 4, (wp + 3) % 4} {
				if g.players[o].kind == KindAI {
					g.say(g.players[o], "gotcha", 0.7, wp, 900)
					break
				}
			}
		default:
			g.sound("trick")
			g.emitAll(event{K: "spark", P: wp})
			if b, took, _ := g.teamBid(team(wp)); b > 0 && took == b && g.bids[wp] != sp.Nil {
				g.say(pw, "made", 0.8, -1, 0)
			} else if w.Card.S == sp.Spades && len(g.history[len(g.history)-1]) > 0 && g.history[len(g.history)-1][0].Card.S != sp.Spades {
				g.say(pw, "trumped", 0.3, -1, 0)
			} else if g.trick[0].p != wp && g.players[sp.PartnerOf(wp)].kind == KindAI && g.rng.Float64() < 0.1 {
				g.say(g.players[sp.PartnerOf(wp)], "partner", 1, wp, 0)
			} else {
				g.say(pw, "trick", 0.1, -1, 0)
			}
		}
		g.sleep(g.T("trick"))
		m := g.tricks[wp] - 1
		for n, pl := range g.trick {
			pl.card.win = false
			pl.card.zn = zone{K: zWon, P: wp, M: m, N: n, J: (g.rng.Float64() - .5) * 6}
		}
		g.trick = nil
		g.sync()
		g.sound("sweep")
		g.sleep(g.T("move"))
	}
}

// statSeat is the seat whose record this computer keeps (none on a shared computer).
func (g *game) statSeat() int {
	if g.mode == ModeHotseat {
		return -1
	}
	return 0
}

func (g *game) scoreHand() {
	var bids [4]int
	for i, b := range g.bids {
		bids[i] = max(0, b)
	}
	res := sp.ScoreHand(bids, g.tricks, g.bags, g.rules)
	me := g.statSeat()
	st := &g.save.Stats
	for t := 0; t < 2; t++ {
		g.score[t] += res[t].Points
		g.bags[t] = res[t].BagsAfter
		g.last[t] = res[t].Points
	}
	for i := 0; i < 4; i++ {
		h := res[team(i)]
		ty := &g.tallies[i]
		ty.Hands++
		ty.Bags += h.Bags
		if h.Contract > 0 {
			if h.Made {
				ty.Made++
			} else {
				ty.Sets++
			}
		}
		if bids[i] == sp.Nil {
			ty.Nils++
			if g.tricks[i] == 0 {
				ty.NilsMade++
			}
		}
	}
	if me >= 0 {
		h := res[team(me)]
		st.Hands++
		st.Bags += h.Bags
		if h.Contract > 0 {
			if h.Made {
				st.Made++
			} else {
				st.Sets++
			}
		}
		if bids[me] == sp.Nil {
			st.Nils++
			if g.tricks[me] == 0 {
				st.NilsMade++
			}
		}
	}

	// nils first
	for i := 0; i < 4; i++ {
		if bids[i] != sp.Nil || g.tricks[i] != 0 {
			continue
		}
		g.sound("nilmade")
		g.emit(func(v int) *event {
			return &event{K: "banner", Text: "Nil Made!", Sub: g.Who(i, v) + " took no tricks at all: +100.", Blue: team(i) != team(v), Ms: g.T("banner") * 1.1}
		})
		g.emit(func(v int) *event {
			if team(i) == team(v) || g.mode == ModeHotseat {
				return &event{K: "spades", N: 30}
			}
			return nil
		})
		g.say(g.players[i], "nilmade", 1, -1, 0)
		g.sleep(g.T("banner") * 1.1)
	}

	// then the contracts, as each side sees them
	line := func(t, v int) string {
		h := res[t]
		s := fmt.Sprintf("%s %+d", g.teamName(t, v), h.Points)
		if h.Bags > 0 {
			s += fmt.Sprintf(" (%d bag%s)", h.Bags, map[bool]string{true: "", false: "s"}[h.Bags == 1])
		}
		return s
	}
	us0 := team(g.viewer)
	snd0 := "made"
	if res[us0].Set || g.mode == ModeHotseat && (res[0].Set || res[1].Set) {
		snd0 = "set"
	}
	g.emit(func(v int) *event {
		snd := "made"
		if res[team(v)].Set {
			snd = "set"
		}
		if g.mode == ModeHotseat {
			snd = snd0
		}
		return &event{K: "snd", Text: snd}
	})
	g.emit(func(v int) *event {
		us, them := team(v), 1-team(v)
		title, blue := "Hand "+itoa(g.handNo+1), true
		switch {
		case g.mode == ModeHotseat:
			switch {
			case res[0].Set && res[1].Set:
				title = "Everybody's Set!"
			case res[0].Set || res[1].Set:
				t := 0
				if res[1].Set {
					t = 1
				}
				title = g.teamName(t, v) + " Got Set!"
				if len(title) > 26 {
					title = "Set!"
				}
			default:
				title = "Both Made It!"
			}
		case res[us].Set:
			title, blue = "Set!", false
		case res[them].Set && res[us].Contract > 0:
			title = "Made It & Set Them!"
		case res[us].Contract > 0:
			title = "Made It!"
		}
		return &event{K: "banner", Text: title, Sub: line(us, v) + "  ·  " + line(them, v), Blue: blue, Ms: g.T("banner") * 1.6}
	})
	g.emit(func(v int) *event {
		h := res[team(v)]
		if g.mode != ModeHotseat && h.Made && h.Contract > 0 {
			return &event{K: "spades", N: 30}
		}
		return nil
	})
	for t := 0; t < 2; t++ {
		var ai []*player
		for _, i := range []int{t, t + 2} {
			if g.players[i].kind == KindAI {
				ai = append(ai, g.players[i])
			}
		}
		if len(ai) == 0 {
			continue
		}
		p := pickS(g, ai)
		switch {
		case res[t].Set:
			g.say(p, "set", 0.8, -1, 300)
		case res[1-t].Set && res[1-t].Contract > 0:
			g.say(p, "setThem", 0.7, -1, 700)
		}
	}
	g.sync()
	g.sleep(g.T("banner") * 1.7)

	// bags
	for t := 0; t < 2; t++ {
		if !res[t].BagHit {
			continue
		}
		g.sound("bags")
		g.emit(func(v int) *event {
			return &event{K: "banner", Text: "Bagged Out!", Sub: g.teamName(t, v) + " hit " + itoa(g.rules.BagLimit) + " bags: −100.", Blue: t != team(v), Ms: g.T("banner") * 1.2}
		})
		for _, i := range []int{t, t + 2} {
			if g.players[i].kind == KindAI {
				g.say(g.players[i], "bags", 0.8, -1, 200)
				break
			}
		}
		g.sleep(g.T("banner") * 1.3)
	}
	g.persist()
}

// resultFor is the end-of-game card for seat v.
func (g *game) resultFor(v int) resultT {
	var r resultT
	w := sp.Winner(g.score)
	for _, t := range []int{team(v), 1 - team(v)} {
		pl := 2
		if t == w {
			pl = 1
		}
		a, b := t, t+2
		if v == b {
			a, b = b, a
		}
		r.rows = append(r.rows, resRow{Name: g.teamName(t, v), Faces: []string{g.players[a].face, g.players[b].face}, Score: g.score[t], Bags: g.bags[t], Place: pl,
			Me: t == team(v) && g.mode != ModeHotseat})
	}
	if r.rows[1].Place == 1 {
		r.rows[0], r.rows[1] = r.rows[1], r.rows[0]
	}
	wn := g.teamName(w, v)
	if g.mode == ModeHotseat {
		r.kick, r.title, r.won = "Last hand of the night", "Game Over!", true
		r.sub = fmt.Sprintf("%s win, %d to %d.", wn, g.score[w], g.score[1-w])
		return r
	}
	r.won = w == team(v)
	if r.won {
		r.kick, r.title = "Drinks are on them", "You Win!"
		r.sub = fmt.Sprintf("You and %s take it, %d to %d.", g.partnerName(v), g.score[w], g.score[1-w])
	} else {
		r.kick, r.title = "Run it back next time", "Good Game!"
		r.sub = fmt.Sprintf("%s win, %d to %d.", wn, g.score[w], g.score[1-w])
	}
	return r
}

func (g *game) endGame() {
	g.finished = true
	g.setTurn(-1)
	g.emitAll(event{K: "prompt"})
	me := g.statSeat()
	w := sp.Winner(g.score)
	if me >= 0 {
		g.recordGame(team(me) == w, g.score[team(me)])
	}
	g.emit(func(v int) *event {
		if team(v) == w || g.mode == ModeHotseat {
			return &event{K: "snd", Text: "win"}
		}
		return &event{K: "snd", Text: "lose"}
	})
	g.emit(func(v int) *event {
		if team(v) == w || g.mode == ModeHotseat {
			return &event{K: "confetti", N: 160}
		}
		return nil
	})
	g.emit(func(v int) *event {
		if team(v) == w || g.mode == ModeHotseat {
			return &event{K: "spades", N: 60, Text: "low"}
		}
		return nil
	})
	var wAI, lAI []*player
	for _, p := range g.players {
		if p.kind == KindAI {
			if team(p.i) == w {
				wAI = append(wAI, p)
			} else {
				lAI = append(lAI, p)
			}
		}
	}
	if len(wAI) > 0 {
		g.say(pickS(g, wAI), "win", 1, -1, 0)
	}
	if len(lAI) > 0 {
		g.say(pickS(g, lAI), "lose", 1, -1, 300)
	}
	ms := 1400.0
	if testFast {
		ms = 140
	}
	g.sleep(ms)
	g.result = g.resultFor(g.viewer)
	g.result.rec = g.recordText()
	g.showResult = true
	g.resultAt = time.Now()
	if g.server != nil {
		g.server.sendResults(g)
	}
}

func (g *game) recordGame(won bool, score int) {
	st := &g.save.Stats
	st.Played++
	if won {
		st.Won++
		st.Streak++
		st.BestStreak = max(st.BestStreak, st.Streak)
		if score > st.Best {
			st.Best = score
		}
	} else {
		st.Lost++
		st.Streak = 0
	}
	g.persist()
}

func (g *game) recordText() string {
	s := g.save.Stats
	if s.Played == 0 {
		return "No games played yet."
	}
	out := fmt.Sprintf("Games: %d · Won: %d · Contracts made: %d of %d", s.Played, s.Won, s.Made, s.Made+s.Sets)
	if s.Nils > 0 {
		out += fmt.Sprintf(" · Nils: %d of %d", s.NilsMade, s.Nils)
	}
	if s.Won > 0 {
		out += fmt.Sprintf(" · Best win: %d", s.Best)
	}
	return out
}

// handsFromZones rebuilds each player's hand from the card zones (guests).
func (g *game) handsFromZones() {
	for _, p := range g.players {
		p.hand = p.hand[:0]
	}
	for _, c := range g.deck {
		if c.zn.K == zHand && c.zn.P >= 0 && c.zn.P < 4 {
			g.players[c.zn.P].hand = append(g.players[c.zn.P].hand, c)
		}
	}
	for _, p := range g.players {
		ps := p.hand
		for a := 1; a < len(ps); a++ {
			for b := a; b > 0 && ps[b].zn.N < ps[b-1].zn.N; b-- {
				ps[b], ps[b-1] = ps[b-1], ps[b]
			}
		}
	}
}
