// Play Together: who sits where on a shared computer, and the name and face
// you use online. (The screens for these are in setup_ui.go.)
package main

import (
	"fmt"
	"strings"
)

type setupState struct {
	hotN       int // people sharing this computer (2-4)
	hotNames   [4]string
	hotFaces   [4]string
	hotPartner int // who partners Player 1: 1-3 for Player 2-4, 0 for a computer
	name, face string
	addr       string
}

func newSetup(sv *saveData) setupState {
	s := sv.Settings
	// saved partner: 0 = not chosen yet (Player 2), 4 = a computer
	st := setupState{hotN: s.HotN, hotPartner: s.HotPartner, name: s.Name, face: s.Face, addr: s.Host}
	switch s.HotPartner {
	case 0:
		st.hotPartner = 1
	case 4:
		st.hotPartner = 0
	}
	if st.hotN < 2 || st.hotN > 4 {
		st.hotN = 2
	}
	for i := 0; i < 4; i++ {
		st.hotNames[i] = fmt.Sprintf("Player %d", i+1)
		st.hotFaces[i] = pickerFaces[i]
		if i < len(s.HotNames) && strings.TrimSpace(s.HotNames[i]) != "" {
			st.hotNames[i] = s.HotNames[i]
		}
		if i < len(s.HotFaces) && faceOK(s.HotFaces[i]) {
			st.hotFaces[i] = s.HotFaces[i]
		}
	}
	st.fixPartner()
	if strings.TrimSpace(st.name) == "" {
		st.name = "Player"
	}
	if !faceOK(st.face) {
		st.face = pickerFaces[0]
	}
	return st
}

// fixPartner keeps the partner choice valid for the number of people.
func (st *setupState) fixPartner() {
	if st.hotPartner >= st.hotN || st.hotPartner < 0 || (st.hotPartner == 0 && st.hotN == 4) {
		st.hotPartner = 1
	}
}

// keepSetup saves the setup choices with the settings.
func (g *game) keepSetup() {
	s := &g.save.Settings
	st := &g.setup
	s.HotN, s.HotPartner, s.Name, s.Face, s.Host = st.hotN, st.hotPartner, st.name, st.face, st.addr
	if st.hotPartner == 0 {
		s.HotPartner = 4
	}
	s.HotNames, s.HotFaces = st.hotNames[:], st.hotFaces[:]
	g.persist()
}

// hotSeats says which seat each person sharing the computer takes:
// Player 1 in seat 0, their partner across in seat 2, the rest at the sides.
func (st *setupState) hotSeats() []int {
	seats := make([]int, st.hotN)
	taken := map[int]bool{0: true}
	if st.hotPartner > 0 {
		seats[st.hotPartner] = 2
		taken[2] = true
	}
	order := []int{1, 3, 2}
	for i := 1; i < st.hotN; i++ {
		if i == st.hotPartner {
			continue
		}
		for _, s := range order {
			if !taken[s] {
				seats[i], taken[s] = s, true
				break
			}
		}
	}
	return seats
}

func (g *game) startHotseat() {
	g.leaveOnline()
	g.keepSetup()
	st := &g.setup
	g.mode, g.viewer = ModeHotseat, 0
	for i := range g.players {
		g.players[i] = nil
	}
	for k, seat := range st.hotSeats() {
		name := cleanName(st.hotNames[k], fmt.Sprintf("Player %d", k+1))
		for _, p := range g.players { // two people with the same name get told apart
			if p != nil && strings.EqualFold(p.name, name) {
				name = cleanName(name, "Player") + " " + itoa(k+1)
			}
		}
		g.setPlayer(seat, KindLocal, st.hotFaces[k], name)
	}
	g.fillFolks()
	g.closePanels()
	g.showTitle = false
	g.startAudio()
	g.newGame()
}

func (g *game) startAudio() {
	snd.setVolume(float64(g.save.Settings.Vol) / 100)
	snd.setSfx(g.save.Settings.Sfx)
	snd.setMusic(g.save.Settings.Music)
	snd.startMusic()
}

// goPlay is "Deal" on the title card: you and a computer partner against two computer players.
func (g *game) goPlay() {
	g.leaveOnline()
	g.mode = ModeSolo
	g.closePanels()
	g.showTitle = false
	g.startAudio()
	g.newGame()
}

// newDeal is New Game / F2 / Play Again: the same table, dealt again.
func (g *game) newDeal() {
	switch g.mode {
	case ModeClient:
		g.showMsg("Only the host can deal a new game.")
	case ModeHost:
		g.showTitle = false
		g.startHosted()
	case ModeHotseat:
		g.closePanels()
		g.showTitle, g.showResult = false, false
		g.newGame()
	default:
		g.goPlay()
	}
}

func (g *game) toMenu() {
	g.showResult = false
	g.showTitle = true
}

func (g *game) closePanels() {
	g.showTogether, g.showHotseat, g.showOnline, g.showLobby = false, false, false, false
	g.message, g.focus = "", ""
}

func (g *game) showMsg(s string) { g.message = s }
