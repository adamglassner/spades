// Package spades holds the rules of partnership Spades and the computer
// players. It has no graphics or sound, so it can be tested on its own.
package spades

import "sort"

// Suit is one of 'S', 'H', 'D', 'C'.
type Suit byte

const (
	Spades   Suit = 'S'
	Hearts   Suit = 'H'
	Diamonds Suit = 'D'
	Clubs    Suit = 'C'
	NoSuit   Suit = 0
)

var Suits = []Suit{Clubs, Diamonds, Spades, Hearts}

func (s Suit) Name() string {
	switch s {
	case Spades:
		return "Spades"
	case Hearts:
		return "Hearts"
	case Diamonds:
		return "Diamonds"
	case Clubs:
		return "Clubs"
	}
	return ""
}

func (s Suit) Symbol() string {
	switch s {
	case Spades:
		return "♠"
	case Hearts:
		return "♥"
	case Diamonds:
		return "♦"
	case Clubs:
		return "♣"
	}
	return ""
}

func (s Suit) Red() bool { return s == Hearts || s == Diamonds }

// Card ranks run 2..14 (J=11, Q=12, K=13, A=14).
type Card struct {
	S Suit
	R int
}

var AceOfSpades = Card{Spades, 14}

func (c Card) ID() string { return string(c.S) + itoa(c.R) }

func RankName(r int) string {
	switch r {
	case 11:
		return "J"
	case 12:
		return "Q"
	case 13:
		return "K"
	case 14:
		return "A"
	}
	return itoa(r)
}

func (c Card) Label() string { return RankName(c.R) + c.S.Symbol() }

func itoa(n int) string {
	if n < 0 {
		return "-" + itoa(-n)
	}
	if n >= 10 {
		return itoa(n/10) + string(rune('0'+n%10))
	}
	return string(rune('0' + n))
}

// Deck returns the 52 cards.
func Deck() []Card {
	d := make([]Card, 0, 52)
	for _, s := range Suits {
		for r := 2; r <= 14; r++ {
			d = append(d, Card{s, r})
		}
	}
	return d
}

/* ---------------- partners ---------------- */

// Team is 0 for seats 0 and 2, 1 for seats 1 and 3. Partners sit across.
func Team(i int) int { return i % 2 }

// PartnerOf is the seat across the table.
func PartnerOf(i int) int { return (i + 2) % 4 }

/* ---------------- tricks ---------------- */

// Play is one card in a trick.
type Play struct {
	P    int
	Card Card
}

func has(h []Card, c Card) bool {
	for _, x := range h {
		if x == c {
			return true
		}
	}
	return false
}

// Legal returns the cards a player may play. You must follow suit if you
// can; spades can't be led until they're broken (someone trumped with one)
// unless you hold nothing but spades.
func Legal(hand []Card, trick []Play, broken bool) []Card {
	var out []Card
	if len(trick) == 0 {
		if !broken {
			for _, c := range hand {
				if c.S != Spades {
					out = append(out, c)
				}
			}
			if len(out) > 0 {
				return out
			}
		}
		return append([]Card(nil), hand...)
	}
	led := trick[0].Card.S
	for _, c := range hand {
		if c.S == led {
			out = append(out, c)
		}
	}
	if len(out) > 0 {
		return out
	}
	return append([]Card(nil), hand...)
}

// Beats reports whether card a beats card b in a trick where led was led.
func Beats(a, b Card, led Suit) bool {
	switch {
	case a.S == b.S:
		return a.R > b.R
	case a.S == Spades:
		return true
	case b.S == Spades:
		return false
	}
	return a.S == led && b.S != led
}

// Winning returns the play currently taking the trick: the highest spade,
// or if there's none the highest card of the suit led.
func Winning(trick []Play) Play {
	if len(trick) == 0 {
		return Play{P: -1}
	}
	led := trick[0].Card.S
	best := trick[0]
	for _, t := range trick[1:] {
		if Beats(t.Card, best.Card, led) {
			best = t
		}
	}
	return best
}

/* ---------------- bidding and scoring ---------------- */

// Nil is the bid of no tricks at all.
const Nil = 0

// Rules are the table's house rules.
type Rules struct {
	Target   int // 500 or 300
	BagLimit int // every this many bags costs 100 (10)
	Floor    int // a team at or below this loses (-200)
}

var DefaultRules = Rules{Target: 500, BagLimit: 10, Floor: -200}

// TeamHand is how one team's hand went.
type TeamHand struct {
	Contract  int   // the tricks the team's non-nil bidders bid together
	Took      int   // tricks those bidders took
	Made      bool  // made the contract (always true with no contract)
	Set       bool  // fell short of it
	Bags      int   // overtricks this hand (nil bidders' tricks count too)
	NilMade   []int // seats that bid nil and took nothing
	NilBust   []int // seats that bid nil and took a trick
	BagHit    bool  // bags reached the limit: -100
	Points    int   // what the team scores this hand
	BagsAfter int   // the team's bag count after this hand
}

// ScoreHand scores a hand for both teams. bids are per seat (Nil = 0),
// tricks are the tricks each seat took, bags are the teams' bags so far.
func ScoreHand(bids, tricks [4]int, bags [2]int, r Rules) [2]TeamHand {
	if r.BagLimit <= 0 {
		r.BagLimit = 10
	}
	var out [2]TeamHand
	for t := 0; t < 2; t++ {
		h := &out[t]
		for _, i := range []int{t, t + 2} {
			if bids[i] == Nil {
				if tricks[i] == 0 {
					h.NilMade = append(h.NilMade, i)
					h.Points += 100
				} else {
					h.NilBust = append(h.NilBust, i)
					h.Points -= 100
					h.Bags += tricks[i]
				}
				continue
			}
			h.Contract += bids[i]
			h.Took += tricks[i]
		}
		switch {
		case h.Contract == 0:
			h.Made = true
			h.Bags += h.Took
		case h.Took >= h.Contract:
			h.Made = true
			h.Points += 10 * h.Contract
			over := h.Took - h.Contract
			h.Points += over
			h.Bags += over
		default:
			h.Set = true
			h.Points -= 10 * h.Contract
		}
		h.BagsAfter = bags[t] + h.Bags
		for h.BagsAfter >= r.BagLimit {
			h.BagsAfter -= r.BagLimit
			h.Points -= 100
			h.BagHit = true
		}
	}
	return out
}

// GameOver reports whether the game is decided: a team has reached the
// target or dropped to the floor, and the scores aren't tied.
func GameOver(score [2]int, r Rules) bool {
	if score[0] == score[1] {
		return false
	}
	for _, s := range score {
		if s >= r.Target || (r.Floor < 0 && s <= r.Floor) {
			return true
		}
	}
	return false
}

// Winner is the team with the higher score.
func Winner(score [2]int) int {
	if score[1] > score[0] {
		return 1
	}
	return 0
}

var suitOrder = map[Suit]int{Diamonds: 0, Clubs: 1, Hearts: 2, Spades: 3}

// SortHand orders a hand for display: diamonds, clubs, hearts, then the
// spades (trumps) at the right, each low to high.
func SortHand(h []Card) {
	sort.SliceStable(h, func(a, b int) bool {
		if h[a].S != h[b].S {
			return suitOrder[h[a].S] < suitOrder[h[b].S]
		}
		return h[a].R < h[b].R
	})
}
