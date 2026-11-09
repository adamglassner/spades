// Bits shared by the game logic and the screen (no graphics here, so the
// headless test build can use them too).
package main

import "math"

var version = "0.0.0-dev" // set with -ldflags "-X main.version=..."

const productName = "Spades"

// Test hooks: SPADES_AUTOPILOT=1 lets the computer play your seats,
// SPADES_FAST=1 speeds everything up (the browser build reads ?autopilot&test).
var autopilot, testFast bool

// testSetup, when set by a test build, prepares the game before the window opens.
var testSetup func(g *game)

const (
	cardW, cardH = 360, 504 // baked card art size
	cardPad      = 64       // margin around card-sized effect images
	artK         = 2.4      // baked card pixels per on-table pixel at a typical size
)

func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }

func itoa(n int) string {
	if n < 0 {
		return "-" + itoa(-n)
	}
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}

// cubic-bezier easing as CSS defines it.
func bezier(x1, y1, x2, y2 float64) func(float64) float64 {
	return func(t float64) float64 {
		if t <= 0 {
			return 0
		}
		if t >= 1 {
			return 1
		}
		u := t
		for i := 0; i < 8; i++ {
			x := 3*(1-u)*(1-u)*u*x1 + 3*(1-u)*u*u*x2 + u*u*u - t
			dx := 3*(1-u)*(1-u)*x1 + 6*(1-u)*u*(x2-x1) + 3*u*u*(1-x2)
			if math.Abs(dx) < 1e-6 {
				break
			}
			u -= x / dx
			u = clamp(u, 0, 1)
		}
		return 3*(1-u)*(1-u)*u*y1 + 3*(1-u)*u*u*y2 + u*u*u
	}
}

var (
	easeCard   = bezier(.3, .7, .2, 1)
	easeInOut  = bezier(.42, 0, .58, 1)
	easeOut    = bezier(0, 0, .58, 1)
	easeBounce = bezier(.2, 1.5, .4, 1)
	easeCSS    = bezier(.25, .1, .25, 1)
)
