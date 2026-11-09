//go:build !headless

// Spades - A Brooklyn Rooftop Card Game (c) 2026 Ironite.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "/version") {
		fmt.Println(version)
		return
	}
	if logf, err := os.OpenFile(filepath.Join(os.TempDir(), "IroniteSpades.log"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644); err == nil {
		log.SetOutput(logf)
	}
	log.Printf("%s %s starting", productName, version)
	autopilot, testFast = testFlags()
	if !singleInstance() {
		return
	}
	loadAssets()
	if runtime.GOOS != "js" { // the browser test build has no preemption; skip audio decoding
		snd.init()
	}
	g := newGameState()
	snd.setVolume(float64(g.save.Settings.Vol) / 100)
	snd.setSfx(g.save.Settings.Sfx)
	snd.setMusic(g.save.Settings.Music)

	ebiten.SetWindowTitle(productName + " " + version)
	ebiten.SetWindowIcon(art.icons)
	ebiten.SetWindowSize(1440, 900)
	ebiten.SetWindowSizeLimits(1024, 640, -1, -1)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetVsyncEnabled(true)
	ebiten.SetScreenClearedEveryFrame(true)
	if autopilot {
		g.mu.Lock()
		g.goPlay()
		g.mu.Unlock()
	}
	if testSetup != nil {
		testSetup(g)
	}
	if err := ebiten.RunGameWithOptions(g, &ebiten.RunGameOptions{}); err != nil {
		fatal("Spades could not start: " + err.Error())
	}
}
