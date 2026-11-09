package spades

import (
	"math/rand/v2"
	"testing"
)

func c(s Suit, r int) Card { return Card{s, r} }

func TestWinning(t *testing.T) {
	tr := []Play{{0, c(Hearts, 10)}, {1, c(Hearts, 14)}, {2, c(Clubs, 14)}, {3, c(Hearts, 2)}}
	if w := Winning(tr); w.P != 1 {
		t.Fatalf("ace of the suit led should win, got %v", w)
	}
	tr[2].Card = c(Spades, 2)
	if w := Winning(tr); w.P != 2 {
		t.Fatalf("a spade trumps, got %v", w)
	}
	tr[3].Card = c(Spades, 3)
	if w := Winning(tr); w.P != 3 {
		t.Fatalf("higher spade over-trumps, got %v", w)
	}
}

func TestLegal(t *testing.T) {
	hand := []Card{c(Spades, 14), c(Hearts, 3), c(Clubs, 9)}
	if L := Legal(hand, nil, false); len(L) != 2 {
		t.Fatalf("can't lead spades before they're broken: %v", L)
	}
	if L := Legal(hand, nil, true); len(L) != 3 {
		t.Fatalf("anything leads once broken: %v", L)
	}
	if L := Legal([]Card{c(Spades, 2), c(Spades, 5)}, nil, false); len(L) != 2 {
		t.Fatalf("only spades: lead one anyway: %v", L)
	}
	if L := Legal(hand, []Play{{0, c(Hearts, 9)}}, false); len(L) != 1 || L[0] != c(Hearts, 3) {
		t.Fatalf("must follow suit: %v", L)
	}
	if L := Legal(hand, []Play{{0, c(Diamonds, 9)}}, false); len(L) != 3 {
		t.Fatalf("void: play anything: %v", L)
	}
}

func TestScore(t *testing.T) {
	r := DefaultRules
	// team 0 bids 4+3 and takes 8 (1 bag); team 1 bids 2+3 and takes 5
	h := ScoreHand([4]int{4, 2, 3, 3}, [4]int{5, 3, 3, 2}, [2]int{}, r)
	if h[0].Points != 71 || h[0].Bags != 1 || !h[0].Made {
		t.Fatalf("team 0: %+v", h[0])
	}
	if h[1].Points != 50 || h[1].Bags != 0 {
		t.Fatalf("team 1: %+v", h[1])
	}
	// set: bid 6, take 5
	h = ScoreHand([4]int{3, 4, 3, 4}, [4]int{2, 4, 3, 4}, [2]int{}, r)
	if h[0].Points != -60 || !h[0].Set {
		t.Fatalf("set: %+v", h[0])
	}
	// nil made and nil busted; the nil bidder's tricks are bags
	h = ScoreHand([4]int{Nil, 4, 5, Nil}, [4]int{0, 6, 6, 1}, [2]int{}, r)
	if h[0].Points != 150+1 || len(h[0].NilMade) != 1 {
		t.Fatalf("nil made: %+v", h[0])
	}
	if h[1].Points != -100+40+2 || len(h[1].NilBust) != 1 || h[1].Bags != 3 {
		t.Fatalf("nil bust: %+v", h[1])
	}
	// bag penalty
	h = ScoreHand([4]int{3, 3, 3, 3}, [4]int{5, 1, 4, 3}, [2]int{8, 0}, r)
	if !h[0].BagHit || h[0].Points != 63-100 || h[0].BagsAfter != 1 {
		t.Fatalf("bags: %+v", h[0])
	}
	if !GameOver([2]int{510, 300}, r) || GameOver([2]int{510, 510}, r) || !GameOver([2]int{100, -210}, r) || GameOver([2]int{499, 0}, r) {
		t.Fatal("game over")
	}
}

// playHand plays one hand with the computer at every seat and checks every play.
func playHand(t *testing.T, rng *rand.Rand, dealer int, bags [2]int) ([4]int, [4]int) {
	d := Deck()
	rng.Shuffle(len(d), func(i, j int) { d[i], d[j] = d[j], d[i] })
	var hands [4][]Card
	for i := 0; i < 52; i++ {
		hands[i%4] = append(hands[i%4], d[i])
	}
	var bids [4]int
	var done [4]bool
	for k := 1; k <= 4; k++ {
		i := (dealer + k) % 4
		b := ChooseBid(BidState{Seat: i, Hand: hands[i], Bids: bids, Done: done, Bags: bags, Rules: DefaultRules})
		if b < 0 || b > 13 {
			t.Fatalf("bid %d", b)
		}
		bids[i], done[i] = b, true
	}
	var tricks [4]int
	var hist [][]Play
	broken := false
	leader := (dealer + 1) % 4
	for n := 0; n < 13; n++ {
		var tr []Play
		for k := 0; k < 4; k++ {
			i := (leader + k) % 4
			L := Legal(hands[i], tr, broken)
			card := ChoosePlay(PlayState{Seat: i, Hand: hands[i], Trick: tr, History: hist, Broken: broken, Bids: bids, Tricks: tricks})
			if !has(L, card) {
				t.Fatalf("illegal play %v from %v (legal %v)", card, hands[i], L)
			}
			var rest []Card
			for _, x := range hands[i] {
				if x != card {
					rest = append(rest, x)
				}
			}
			hands[i] = rest
			if card.S == Spades {
				broken = true
			}
			tr = append(tr, Play{i, card})
		}
		w := Winning(tr).P
		tricks[w]++
		hist = append(hist, tr)
		leader = w
	}
	return bids, tricks
}

func TestComputerPlayers(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	made, contracts, nils, nilMade, bagsTot := 0, 0, 0, 0, 0
	for g := 0; g < 600; g++ {
		bids, tricks := playHand(t, rng, g%4, [2]int{})
		tot := 0
		for _, x := range tricks {
			tot += x
		}
		if tot != 13 {
			t.Fatalf("tricks %v", tricks)
		}
		h := ScoreHand(bids, tricks, [2]int{}, DefaultRules)
		for k := 0; k < 2; k++ {
			if h[k].Contract > 0 {
				contracts++
				if h[k].Made {
					made++
				}
			}
			nils += len(h[k].NilMade) + len(h[k].NilBust)
			nilMade += len(h[k].NilMade)
			bagsTot += h[k].Bags
		}
	}
	rate := float64(made) / float64(contracts)
	t.Logf("contracts made %d/%d (%.0f%%), nils %d made %d, bags/team-hand %.2f", made, contracts, rate*100, nils, nilMade, float64(bagsTot)/1200)
	if rate < 0.6 || rate > 0.97 {
		t.Fatalf("computer bidding looks off: %.2f made", rate)
	}
	if nils > 0 && float64(nilMade)/float64(nils) < 0.5 {
		t.Fatalf("too many nils busted: %d of %d made", nilMade, nils)
	}
}

func TestFullGames(t *testing.T) {
	rng := rand.New(rand.NewPCG(7, 9))
	for g := 0; g < 40; g++ {
		var score [2]int
		var bags [2]int
		hands := 0
		for !GameOver(score, DefaultRules) {
			bids, tricks := playHand(t, rng, hands%4, bags)
			h := ScoreHand(bids, tricks, bags, DefaultRules)
			for k := 0; k < 2; k++ {
				score[k] += h[k].Points
				bags[k] = h[k].BagsAfter
			}
			hands++
			if hands > 200 {
				t.Fatal("game never ends")
			}
		}
		if g < 5 {
			t.Logf("game %d: %v after %d hands", g, score, hands)
		}
	}
}
