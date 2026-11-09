//go:build js && !headless

package main

// The browser build is only used for automated visual checks.

import (
	"strings"
	"syscall/js"
)

func fatal(text string) { panic(text) }

func singleInstance() bool { return true }

func init() {
	// test hooks for screenshots (the browser can't host a game, so ?lobby
	// shows a made-up one): ?panel=together|hotseat|online|rules|credits|faces,
	// ?lobby, ?hot (a shared-computer game), ?deal (a single-player game)
	testSetup = func(g *game) {
		q := js.Global().Get("location").Get("search").String()
		g.mu.Lock()
		defer g.mu.Unlock()
		switch {
		case strings.Contains(q, "panel=together"):
			g.showTitle, g.showTogether = false, true
		case strings.Contains(q, "panel=hotseat"):
			g.showTitle, g.showHotseat = false, true
			g.setup.hotN, g.setup.hotNames = 3, [4]string{"Adam", "Jess", "Sam", ""}
		case strings.Contains(q, "panel=online"):
			g.showTitle, g.showOnline = false, true
			g.setup.name, g.setup.addr = "Adam", "192.168.1.20"
		case strings.Contains(q, "panel=rules"):
			g.showRules = true
		case strings.Contains(q, "panel=faces"):
			g.showTitle, showFaces = false, true
		case strings.Contains(q, "panel=credits"):
			g.showTitle, g.showCredits = false, true
		case strings.Contains(q, "lobby"):
			g.showTitle, g.mode, g.showLobby = false, ModeHost, true
			g.players[0] = &player{i: 0, kind: KindLocal, face: pickerFaces[1], name: "Adam", full: "Adam"}
			g.players[2] = &player{i: 2, kind: KindRemote, face: pickerFaces[2], name: "Jess", full: "Jess"}
		case strings.Contains(q, "hot"):
			g.setup.hotN, g.setup.hotNames = 3, [4]string{"Adam", "Jess", "Sam", ""}
			g.startHotseat()
		case strings.Contains(q, "deal"):
			g.goPlay()
		}
	}
}

func testFlags() (bool, bool) {
	q := js.Global().Get("location").Get("search").String()
	return strings.Contains(q, "autopilot"), strings.Contains(q, "test")
}
