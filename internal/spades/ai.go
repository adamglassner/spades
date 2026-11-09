package spades

import "sort"

/* ---------------- bidding ---------------- */

// BidState is what a computer player knows when it's their turn to bid.
type BidState struct {
	Seat  int
	Hand  []Card
	Bids  [4]int  // bids made so far
	Done  [4]bool // who has bid
	Score [2]int
	Bags  [2]int
	Rules Rules
}

func bySuit(h []Card) map[Suit][]Card {
	m := map[Suit][]Card{}
	for _, c := range h {
		m[c.S] = append(m[c.S], c)
	}
	for _, cs := range m {
		sort.Slice(cs, func(i, j int) bool { return cs[i].R > cs[j].R })
	}
	return m
}

func hasRank(cs []Card, r int) bool {
	for _, c := range cs {
		if c.R == r {
			return true
		}
	}
	return false
}

// Estimate is how many tricks a hand should take, by the usual rules of thumb.
func Estimate(hand []Card) float64 {
	m := bySuit(hand)
	sp := m[Spades]
	ns := len(sp)
	est := 0.0
	for _, s := range []Suit{Hearts, Diamonds, Clubs} {
		cs := m[s]
		n := len(cs)
		a, k, q := hasRank(cs, 14), hasRank(cs, 13), hasRank(cs, 12)
		if a {
			if n <= 6 {
				est += 1
			} else {
				est += 0.7
			}
		}
		if k {
			switch {
			case n == 1:
				est += 0.15
			case a && n <= 5:
				est += 0.9
			case n <= 5:
				est += 0.6
			default:
				est += 0.3
			}
		}
		if q && n >= 3 && n <= 4 {
			if a || k {
				est += 0.4
			} else {
				est += 0.2
			}
		}
	}
	// spades: the honours, then long trumps
	honours, used := 0.0, 0
	if hasRank(sp, 14) {
		honours += 1
		used++
	}
	if hasRank(sp, 13) {
		used++
		if ns >= 2 {
			honours += 1
		} else {
			honours += 0.5
		}
	}
	if hasRank(sp, 12) {
		used++
		switch {
		case ns >= 3:
			honours += 0.9
		case ns == 2:
			honours += 0.4
		}
	}
	if hasRank(sp, 11) && ns >= 4 {
		honours += 0.5
	}
	est += honours
	if ns > 3 {
		est += float64(ns-3) * 0.85
	}
	// short side suits let the low spades trump in
	spare := float64(max(0, min(ns, 3)-used))
	ruff := 0.0
	for _, s := range []Suit{Hearts, Diamonds, Clubs} {
		switch len(m[s]) {
		case 0:
			ruff += 0.9
		case 1:
			ruff += 0.5
		case 2:
			ruff += 0.15
		}
	}
	est += min(ruff, spare*0.8)
	return est
}

// nilRisk scores how likely a hand is to be forced to take a trick.
func nilRisk(hand []Card) float64 {
	m := bySuit(hand)
	sp := m[Spades]
	risk := 0.0
	for _, c := range sp {
		switch {
		case c.R >= 12:
			risk += 3
		case c.R >= 10:
			risk += 0.8
		case c.R >= 8:
			risk += 0.3
		}
	}
	if len(sp) >= 4 {
		risk += float64(len(sp)-3) * 1.2
	}
	for _, s := range []Suit{Hearts, Diamonds, Clubs} {
		cs := m[s]
		n := len(cs)
		low := 0
		for _, c := range cs {
			if c.R <= 6 {
				low++
			}
		}
		for _, c := range cs {
			switch {
			case c.R == 14:
				risk += 1.6
			case c.R == 13 && low < 2:
				risk += 1
			case c.R == 13:
				risk += 0.4
			case c.R == 12 && low < 2:
				risk += 0.6
			case c.R >= 10 && n <= 2 && low == 0:
				risk += 0.3
			}
		}
	}
	return risk
}

// ChooseBid picks a computer player's bid: 0 for nil, otherwise 1 to 13.
func ChooseBid(bs BidState) int {
	est := Estimate(bs.Hand)
	p := PartnerOf(bs.Seat)
	partnerBid := -1
	if bs.Done[p] {
		partnerBid = bs.Bids[p]
	}
	if partnerBid != Nil {
		risk := nilRisk(bs.Hand)
		ok := risk < 0.9 && est < 1.7
		if partnerBid >= 4 && risk < 1.3 && est < 2.1 {
			ok = true // partner's strong enough to carry the contract
		}
		if ok {
			return Nil
		}
	}
	bid := int(est + 0.35)
	if partnerBid == Nil {
		bid = int(est + 0.6) // carrying a nil partner: bid what we'll take
	}
	// the last to bid trims the table's total back toward 13
	n := 0
	tot := 0
	for i := 0; i < 4; i++ {
		if bs.Done[i] {
			n++
			tot += bs.Bids[i]
		}
	}
	if n == 3 && tot+bid > 13 {
		bid = max(1, 13-tot)
	}
	// near the bag limit, don't underbid
	if bs.Bags[Team(bs.Seat)] >= bs.Rules.BagLimit-2 && est-float64(bid) > 0.4 {
		bid++
	}
	return min(13, max(1, bid))
}

/* ---------------- playing ---------------- */

// PlayState is what a computer player sees when it's their turn to play.
type PlayState struct {
	Seat    int
	Hand    []Card
	Trick   []Play
	History [][]Play // the tricks already finished this hand
	Broken  bool
	Bids    [4]int
	Tricks  [4]int // tricks each seat has taken this hand
}

type seen struct {
	gone map[Card]bool // played, or in our own hand
	void [4]map[Suit]bool
}

func (ps PlayState) seen() seen {
	s := seen{gone: map[Card]bool{}}
	for i := range s.void {
		s.void[i] = map[Suit]bool{}
	}
	for _, t := range append(append([][]Play(nil), ps.History...), ps.Trick) {
		if len(t) == 0 {
			continue
		}
		led := t[0].Card.S
		for _, p := range t {
			s.gone[p.Card] = true
			if p.Card.S != led {
				s.void[p.P][led] = true
				if p.Card.S != Spades {
					s.void[p.P][Spades] = true // discarded instead of trumping
				}
			}
		}
	}
	for _, c := range ps.Hand {
		s.gone[c] = true
	}
	return s
}

// higherOut counts the cards of su above r that someone else may still hold.
func (s seen) higherOut(su Suit, r int) int {
	n := 0
	for k := r + 1; k <= 14; k++ {
		if !s.gone[Card{su, k}] {
			n++
		}
	}
	return n
}

func (s seen) out(su Suit) int { return s.higherOut(su, 1) }

// contract is what a team's non-nil bidders bid, and what they've taken.
func contract(bids, tricks [4]int, t int) (bid, took int) {
	for _, i := range []int{t, t + 2} {
		if bids[i] != Nil {
			bid += bids[i]
			took += tricks[i]
		}
	}
	return
}

func lowest(cs []Card) Card {
	best := cs[0]
	for _, c := range cs[1:] {
		if c.R < best.R || c.R == best.R && best.S == Spades {
			best = c
		}
	}
	return best
}

func highest(cs []Card) Card {
	best := cs[0]
	for _, c := range cs[1:] {
		if c.R > best.R {
			best = c
		}
	}
	return best
}

func filter(cs []Card, f func(Card) bool) []Card {
	var out []Card
	for _, c := range cs {
		if f(c) {
			out = append(out, c)
		}
	}
	return out
}

// cheapest is the least valuable card in cs: side cards before spades, low before high.
func cheapest(cs []Card) Card {
	side := filter(cs, func(c Card) bool { return c.S != Spades })
	if len(side) > 0 {
		return lowest(side)
	}
	return lowest(cs)
}

// ChoosePlay picks the card a computer player plays.
func ChoosePlay(ps PlayState) Card {
	L := Legal(ps.Hand, ps.Trick, ps.Broken)
	if len(L) == 1 {
		return L[0]
	}
	sn := ps.seen()
	me, pa := ps.Seat, PartnerOf(ps.Seat)
	t := Team(me)
	alive := func(i int) bool { return ps.Bids[i] == Nil && ps.Tricks[i] == 0 }
	myNil, partnerNil := alive(me), alive(pa)
	oppNil := -1
	for _, i := range []int{(me + 1) % 4, (me + 3) % 4} {
		if alive(i) {
			oppNil = i
		}
	}
	bid, took := contract(ps.Bids, ps.Tricks, t)
	obid, otook := contract(ps.Bids, ps.Tricks, 1-t)
	left := 13 - len(ps.History)
	need, oneed := bid-took, obid-otook
	// win tricks if we still need them, or if the other side could still make theirs
	wantWin := need > 0 || (oneed > 0 && oneed <= left)

	played := func(i int) bool {
		for _, p := range ps.Trick {
			if p.P == i {
				return true
			}
		}
		return false
	}
	// a side-suit card is a sure winner if nothing higher is out and nobody
	// still to play can trump it
	boss := func(c Card) bool {
		if sn.higherOut(c.S, c.R) > 0 {
			return false
		}
		if c.S == Spades {
			return true
		}
		for k := 1; k < 4; k++ {
			i := (me + k) % 4
			if i == pa || played(i) {
				continue
			}
			if sn.void[i][c.S] && !sn.void[i][Spades] {
				return false
			}
		}
		return sn.out(c.S) >= 2 // a suit nearly gone gets trumped
	}

	/* ---------- leading ---------- */
	if len(ps.Trick) == 0 {
		if myNil {
			return cheapest(L)
		}
		if partnerNil {
			// lead winners and high cards so partner can duck under them
			best, bv := L[0], -1e9
			for _, c := range L {
				v := float64(c.R)
				if boss(c) {
					v += 20
				}
				if sn.void[pa][c.S] {
					v += 6 // partner can throw off
				}
				if c.S == Spades {
					v -= 4
				}
				if v > bv {
					best, bv = c, v
				}
			}
			return best
		}
		if oppNil >= 0 {
			// lead low in a suit the nil bidder still holds
			best, bv := L[0], -1e9
			for _, c := range L {
				v := -float64(c.R)
				if sn.void[oppNil][c.S] {
					v -= 20
				}
				if c.S == Spades {
					v -= 3
				}
				if v > bv {
					best, bv = c, v
				}
			}
			return best
		}
		if wantWin {
			for _, c := range L {
				if c.S != Spades && boss(c) {
					return c
				}
			}
			// pull trumps when holding most of the spades still out
			mySp := filter(L, func(c Card) bool { return c.S == Spades })
			if len(mySp) > 0 && ps.Broken && len(mySp) > sn.out(Spades) {
				return highest(mySp)
			}
			// lead low from a short side suit, to trump that suit later
			count := map[Suit]int{}
			for _, c := range ps.Hand {
				count[c.S]++
			}
			best, bv := L[0], -1e9
			for _, c := range L {
				v := -float64(c.R) * 0.5
				if c.S == Spades {
					v -= 10
				} else if count[Spades] > 0 {
					v += float64(6-min(6, count[c.S])) * 1.5
				}
				if v > bv {
					best, bv = c, v
				}
			}
			return best
		}
		// we're home: lead something that loses
		best, bv := L[0], -1e9
		for _, c := range L {
			v := -float64(c.R) + float64(sn.higherOut(c.S, c.R))*0.6
			if c.S == Spades {
				v -= 6
			}
			if v > bv {
				best, bv = c, v
			}
		}
		return best
	}

	/* ---------- following ---------- */
	led := ps.Trick[0].Card.S
	win := Winning(ps.Trick)
	last := len(ps.Trick) == 3
	beats := filter(L, func(c Card) bool { return Beats(c, win.Card, led) })
	under := filter(L, func(c Card) bool { return !Beats(c, win.Card, led) })
	follow := L[0].S == led
	// cheapest winning card: the lowest of the suit, or the lowest trump
	cheapWin := func() Card {
		same := filter(beats, func(c Card) bool { return c.S == led })
		if len(same) > 0 {
			return lowest(same)
		}
		return lowest(beats)
	}
	// throw away: under the winner if we can, as high as we can (or a card
	// from a short side suit when we can't follow)
	duck := func() Card {
		if len(under) > 0 {
			if follow {
				return highest(under)
			}
			side := filter(under, func(c Card) bool { return c.S != Spades })
			if len(side) > 0 {
				return highest(side)
			}
			return highest(under)
		}
		if last {
			return highest(beats) // winning anyway: spend the big card
		}
		return lowest(beats)
	}
	discard := func() Card {
		if follow {
			return lowest(L)
		}
		count := map[Suit]int{}
		for _, c := range ps.Hand {
			count[c.S]++
		}
		side := filter(L, func(c Card) bool { return c.S != Spades })
		if len(side) == 0 {
			return lowest(L)
		}
		best, bv := side[0], -1e9
		for _, c := range side {
			v := -float64(c.R) - float64(count[c.S])*1.5
			if v > bv {
				best, bv = c, v
			}
		}
		return best
	}

	if myNil {
		return duck()
	}
	if partnerNil {
		if win.P == pa && len(beats) > 0 {
			return cheapWin() // take it off partner
		}
		if !played(pa) && len(beats) > 0 {
			// partner plays after us: win as high as we can so they can duck
			same := filter(beats, func(c Card) bool { return c.S == led })
			if len(same) > 0 {
				return highest(same)
			}
			return lowest(beats)
		}
		if win.P != pa && len(beats) == 0 {
			return discard()
		}
	}
	if oppNil >= 0 {
		if win.P == oppNil {
			return duck() // let them keep it
		}
		if !played(oppNil) && win.P != pa {
			// they play after us: keep the trick low so they might take it
			if len(under) > 0 {
				return lowest(under)
			}
		}
	}
	if !wantWin {
		return duck()
	}
	if win.P == pa {
		// partner's winning: leave it unless they look like being beaten
		safe := last
		if !safe && win.Card.S == led && sn.higherOut(led, win.Card.R) == 0 {
			safe = true
			for k := len(ps.Trick) + 1; k < 4; k++ {
				i := (ps.Trick[0].P + k) % 4
				if sn.void[i][led] && !sn.void[i][Spades] {
					safe = false
				}
			}
		}
		if win.Card.S == Spades && sn.higherOut(Spades, win.Card.R) == 0 {
			safe = true
		}
		if safe || len(beats) == 0 {
			return discard()
		}
		if c := cheapWin(); boss(c) {
			return c
		}
		return discard()
	}
	if len(beats) == 0 {
		return discard()
	}
	if last {
		return cheapWin()
	}
	c := cheapWin()
	if c.S != led {
		// trumping in: use a spade big enough that the rest can't overtrump
		sp := filter(beats, func(x Card) bool { return x.S == Spades })
		sort.Slice(sp, func(i, j int) bool { return sp[i].R < sp[j].R })
		for _, x := range sp {
			if sn.higherOut(Spades, x.R) == 0 {
				return x
			}
		}
		return sp[0]
	}
	if len(ps.Trick) == 1 {
		// second hand low, unless we hold a sure winner
		for _, x := range beats {
			if x.S == led && boss(x) {
				return x
			}
		}
		return discard()
	}
	// third hand high
	same := filter(beats, func(x Card) bool { return x.S == led })
	for _, x := range same {
		if boss(x) {
			return x
		}
	}
	return highest(same)
}
