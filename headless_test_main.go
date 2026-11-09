//go:build headless

// A test build with no window or sound: plays whole games with the computer
// at every seat, including online games between a host and guests over
// real TCP on this computer. Run: go run -tags headless .
package main

import (
	"fmt"
	"os"
	"time"

	sp "github.com/adamglassner/spades/internal/spades"
)

type uiState struct{}

func check(ok bool, f string, a ...any) {
	if !ok {
		fmt.Printf("FAIL: "+f+"\n", a...)
		os.Exit(1)
	}
}

func waitFor(what string, d time.Duration, f func() bool) {
	end := time.Now().Add(d)
	for time.Now().Before(end) {
		if f() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	check(false, "timed out waiting for %s", what)
}

func locked[T any](g *game, f func() T) T {
	g.mu.Lock()
	defer g.mu.Unlock()
	return f()
}

// fitsFace checks a regular's name was made for their face.
func fitsFace(face, name string) bool {
	f, ok := parseFace(face)
	if !ok {
		return false
	}
	for _, n := range mannerNames[f.mannerKey()].names {
		if n == name {
			return true
		}
	}
	return false
}

func main() {
	os.Setenv("SPADES_DATA", os.TempDir()+"/spades-headless-test")
	os.RemoveAll(os.TempDir() + "/spades-headless-test")
	autopilot, testFast = true, true

	// 0. faces: names always fit, ladies are ladies
	for i := 0; i < 2000; i++ {
		f := Face{Female: i%2 == 0, Seed: uint64(i)*7919 + 3}
		n, full, ok := f.nameFor(map[string]bool{})
		check(ok && fitsFace(f.String(), n), "face %v got %q", f, full)
		l := f.look()
		check(l.Lady == f.Female, "a lady's face must look like a lady")
		if !f.Female {
			check(l.Lip.A == 0 && l.HairStyle != "bob", "a gentleman's face got a lady's look")
		}
		g2, ok := parseFace(f.String())
		check(ok && g2 == f, "face round trip")
	}

	// 1. single player: three random regulars with names that fit
	g := newGameState()
	g.mu.Lock()
	g.goPlay()
	seen := map[string]bool{}
	for i := 1; i < 4; i++ {
		p := g.players[i]
		check(p.kind == KindAI && fitsFace(p.face, p.name), "seat %d: %q doesn't fit %s", i, p.name, p.face)
		check(!seen[p.face] && !seen[p.name], "repeated face or name")
		seen[p.face], seen[p.name] = true, true
		fmt.Printf("solo seat %d: %s (%s)\n", i, p.full, p.face)
	}
	g.mu.Unlock()
	hands := 0
	waitFor("solo game", 5*time.Minute, func() bool {
		return locked(g, func() bool {
			// every card is somewhere sensible and points always add up
			if !g.showResult {
				hands = g.handNo
			}
			return g.showResult
		})
	})
	g.mu.Lock()
	tot := g.tricks[0] + g.tricks[1] + g.tricks[2] + g.tricks[3]
	check(tot == 13, "tricks %v don't add up to 13", g.tricks)
	check(sp.GameOver(g.score, g.rules), "game ended early: %v", g.score)
	check(g.save.Stats.Played == 1 && g.save.Stats.Hands == g.handNo, "record %+v after %d hands", g.save.Stats, g.handNo)
	fmt.Printf("solo: %v after %d hands, %q / %s\n", g.score, g.handNo, g.result.title, g.result.sub)
	g.mu.Unlock()
	_ = hands

	// 2. online: a host and two guests
	host := newGameState()
	host.setup.name, host.setup.face = "Hosty", pickerFaces[1]
	host.mu.Lock()
	host.hostGame()
	check(host.server != nil, "host should be listening")
	host.mu.Unlock()
	var guests []*game
	for i, nm := range []string{"Ann", "Bo"} {
		c := newGameState()
		c.save.Stats = defaultSave().Stats // everyone here shares one save folder
		c.setup.name, c.setup.face = nm, pickerFaces[2+i]
		c.mu.Lock()
		c.joinGame("127.0.0.1")
		c.mu.Unlock()
		guests = append(guests, c)
		waitFor(nm+" to join", 10*time.Second, func() bool {
			return locked(host, func() bool { return len(host.server.peers) == i+1 }) && locked(c, func() bool { return c.mode == ModeClient })
		})
	}
	host.mu.Lock()
	check(host.players[2].name == "Ann" && host.players[1].name == "Bo", "join order: across first, then left")
	host.mu.Unlock()
	time.Sleep(100 * time.Millisecond)
	for _, c := range guests {
		c.mu.Lock()
		check(len(c.lobby) == 4, "guest should have the lobby")
		c.mu.Unlock()
	}

	for round := 1; round <= 2; round++ {
		host.mu.Lock()
		if round == 1 {
			host.startHosted()
		} else {
			host.newDeal()
		}
		host.mu.Unlock()
		// while it plays, guests should always know their own cards and nobody else's
		done, passes := false, 0
		for !done {
			time.Sleep(10 * time.Millisecond)
			for _, c := range guests {
				c.mu.Lock()
				me := c.viewer
				for _, card := range c.deck {
					if card.zn.K == zHand && card.zn.P == me {
						check(card.id != "", "guest doesn't know a card in their own hand")
					}
				}
				if c.ask != nil && len(c.ask.btns) > 0 {
					passes++
				}
				for i, p := range c.players {
					if i != me {
						for _, card := range p.hand {
							check(card.id == "" || card.fa1 == 180, "guest can see someone else's card")
						}
					}
				}
				c.mu.Unlock()
			}
			done = locked(host, func() bool { return host.showResult })
		}
		check(passes > 0, "guests were never asked to bid")
		waitFor("guests' results", 10*time.Second, func() bool {
			for _, c := range guests {
				if !locked(c, func() bool { return c.showResult }) {
					return false
				}
			}
			return true
		})
		hs := host.score
		for _, c := range guests {
			c.mu.Lock()
			v := c.viewer
			check(c.score == hs, "guest scores %v vs host %v", c.score, hs)
			check(len(c.result.rows) == 2, "guest result rows")
			check(c.save.Stats.Played == round, "guest record: %d games", c.save.Stats.Played)
			fmt.Printf("round %d: %s sees %q (%s) %v\n", round, c.players[v].name, c.result.title, c.result.sub, c.score)
			c.mu.Unlock()
		}
	}

	// 3. a guest leaves mid-game: a computer player takes over and the game finishes
	host.mu.Lock()
	host.newDeal()
	host.mu.Unlock()
	time.Sleep(400 * time.Millisecond)
	bo := guests[1]
	bo.mu.Lock()
	bo.leaveOnline()
	bo.mu.Unlock()
	waitFor("game after a guest left", 5*time.Minute, func() bool { return locked(host, func() bool { return host.showResult }) })
	host.mu.Lock()
	check(host.players[1].kind == KindAI, "Bo's seat should be a computer now")
	check(len(host.server.peers) == 1, "one guest left")
	host.leaveOnline()
	host.mu.Unlock()
	waitFor("Ann to hear the host closed", 10*time.Second, func() bool {
		return locked(guests[0], func() bool { return guests[0].mode == ModeSolo && guests[0].message != "" })
	})

	// 4. same computer: three people, answered through the real pass screen,
	// bid boxes and card picks
	autopilot = false
	hs := newGameState()
	hs.mu.Lock()
	hs.save.Settings.Target = 300
	hs.setup.hotN = 3
	hs.setup.hotNames = [4]string{"Al", "Bea", "Cy", ""}
	hs.setup.hotPartner = 1
	hs.startHotseat()
	check(hs.players[0].name == "Al" && hs.players[2].name == "Bea" && hs.players[1].name == "Cy" && hs.players[3].kind == KindAI, "hot-seat seating")
	hs.mu.Unlock()
	passScreens, turns, passed := 0, map[int]int{}, map[int]int{}
	for !locked(hs, func() bool { return hs.showResult }) {
		time.Sleep(3 * time.Millisecond)
		hs.mu.Lock()
		switch {
		case hs.passCh != nil:
			for _, c := range hs.deck {
				check(c.zn.K != zHand || c.fa1 == 180, "a hand is showing behind the pass screen")
			}
			passScreens++
			hs.passCh <- struct{}{}
			hs.passCh = nil
		case hs.ask != nil:
			a := hs.ask
			hs.ask = nil
			check(hs.players[hs.viewer].kind == KindLocal && len(a.btns) == 14, "bid box for a computer seat")
			passed[hs.viewer]++
			n := 1 + hs.rng.IntN(5)
			a.ch <- askRes{btn: a.btns[n], idx: n}
		case hs.wait != nil:
			w := hs.wait
			hs.wait = nil
			for _, c := range w.all {
				check(c.zn.P == hs.viewer, "picking from someone else's hand")
			}
			turns[hs.viewer]++
			w.ch <- w.ok[hs.rng.IntN(len(w.ok))]
		}
		hs.mu.Unlock()
	}
	check(passScreens > 3 && turns[0] > 0 && turns[1] > 0 && turns[2] > 0 && turns[3] == 0 && passed[0] > 0, "hot-seat turns %v bids %v screens %d", turns, passed, passScreens)
	fmt.Printf("same computer: %d pass screens, turns %v, bids %v, %q %s\n", passScreens, turns, passed, hs.result.title, hs.result.sub)
	fmt.Println("PASS")
}
