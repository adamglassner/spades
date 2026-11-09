//go:build !headless

package main

import (
	"bytes"
	"embed"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

//go:embed assets CHANGELOG.md
var assetFS embed.FS

type assetSet struct {
	cards                     map[string]*ebiten.Image // "H11", "back"
	drink, snack, logo        *ebiten.Image
	deck, lights              *ebiten.Image
	icons                     []image.Image
	cardShadow                *ebiten.Image
	ringPick, ringWin         *ebiten.Image
	ringSel, ringFresh        *ebiten.Image
	mat, vignette, lamp       *ebiten.Image
	dot, white                *ebiten.Image
	slab, dispI, bitter, ital *text.GoTextFaceSource
	changelog                 string
	tunes                     []string
}

var art assetSet

func mustImage(name string) image.Image {
	b, err := assetFS.ReadFile("assets/" + name)
	if err != nil {
		panic(err)
	}
	im, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		panic(name + ": " + err.Error())
	}
	return im
}

func mustFont(name string) *text.GoTextFaceSource {
	b, _ := assetFS.ReadFile("assets/fonts/" + name)
	s, err := text.NewGoTextFaceSource(bytes.NewReader(b))
	if err != nil {
		panic(err)
	}
	return s
}

func loadAssets() {
	art.cards = map[string]*ebiten.Image{}
	for _, s := range "SHDC" {
		for r := 2; r <= 14; r++ {
			id := string(s) + itoa(r)
			art.cards[id] = ebiten.NewImageFromImage(mustImage("cards/" + id + ".png"))
		}
	}
	art.cards["back"] = ebiten.NewImageFromImage(mustImage("cards/back.png"))
	art.drink = ebiten.NewImageFromImage(mustImage("soda.png"))
	art.snack = ebiten.NewImageFromImage(mustImage("pizza.png"))
	art.deck = ebiten.NewImageFromImage(mustImage("deck.jpg"))
	art.lights = ebiten.NewImageFromImage(mustImage("lights.png"))
	art.logo = ebiten.NewImageFromImage(mustImage("ironite-logo.png"))
	for _, n := range []int{16, 32, 48, 64, 256} {
		art.icons = append(art.icons, mustImage("icon-"+itoa(n)+".png"))
	}
	art.slab = mustFont("Oswald.ttf")
	art.dispI = mustFont("Pacifico.ttf")
	art.bitter = mustFont("Rubik.ttf")
	art.ital = mustFont("Rubik-Italic.ttf")
	cl, _ := assetFS.ReadFile("CHANGELOG.md")
	art.changelog = string(cl)
	tn, _ := assetFS.ReadFile("assets/audio/tunes.txt")
	art.tunes = strings.Split(strings.TrimSpace(strings.ReplaceAll(string(tn), "\r", "")), "\n")

	// Card-shaped shadows and highlight rings, from a rounded-rect distance field.
	k := artK
	art.cardShadow = cardField(func(d, dy float64) color.NRGBA { // 0 2px 4px .35, 0 6px 14px .2
		a := 0.35*gauss(sdCard(d, dy-2*k), 2*k) + 0.2*gauss(sdCard(d, dy-6*k), 7*k)
		return color.NRGBA{0, 0, 0, u8(math.Min(1, a))}
	})
	art.ringPick = cardField(func(d, dy float64) color.NRGBA { // 0 0 0 3px #ffd862, 0 8px 18px .4
		sd := sdCard(d, dy)
		if sd > 0 && sd <= 3*k {
			return color.NRGBA{0xff, 0xd8, 0x62, 255}
		}
		return color.NRGBA{0, 0, 0, u8(0.4 * gauss(sdCard(d, dy-8*k), 9*k))}
	})
	art.ringWin = cardField(func(d, dy float64) color.NRGBA { // 0 0 0 4px #ffd862, 0 0 24px 6px rgba(255,214,98,.8)
		sd := sdCard(d, dy)
		if sd > 0 && sd <= 4*k {
			return color.NRGBA{0xff, 0xd8, 0x62, 255}
		}
		return color.NRGBA{255, 214, 98, u8(0.8 * gauss(sd-6*k, 12*k))}
	})
	ring := func(r, g, b uint8, a float64) func(d, dy float64) color.NRGBA {
		return func(d, dy float64) color.NRGBA {
			sd := sdCard(d, dy)
			if sd > 0 && sd <= 4*k {
				return color.NRGBA{r, g, b, 255}
			}
			return color.NRGBA{r, g, b, u8(a * gauss(sd-6*k, 12*k))}
		}
	}
	art.ringSel = cardField(ring(0x7f, 0xd8, 0xff, 0.7))
	art.ringFresh = cardField(ring(0xff, 0x9a, 0xb0, 0.75))
	art.mat = makeMat()
	art.vignette = makeVignette()
	art.lamp = makeLamp()
	dot := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			d := math.Hypot(float64(x)-31.5, float64(y)-31.5)
			dot.SetNRGBA(x, y, color.NRGBA{255, 255, 255, u8(clamp(32-d, 0, 1))})
		}
	}
	art.dot = ebiten.NewImageFromImage(dot)
	art.white = ebiten.NewImage(4, 4)
	art.white.Fill(color.White)
}

func u8(a float64) uint8 { return uint8(clamp(a, 0, 1)*255 + 0.5) }

// gauss turns a signed distance into the coverage of a blurred edge.
func gauss(sd, sigma float64) float64 { return 0.5 * math.Erfc(sd/(sigma*math.Sqrt2)) }

// cardField renders a card-sized (plus margin) image; f gets the pixel offset
// from the card centre packed as (x, y).
func cardField(f func(dx, dy float64) color.NRGBA) *ebiten.Image {
	w, h := cardW+2*cardPad, cardH+2*cardPad
	im := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			im.SetNRGBA(x, y, f(float64(x)+0.5-float64(w)/2, float64(y)+0.5-float64(h)/2))
		}
	}
	return ebiten.NewImageFromImage(im)
}

// sdCard: signed distance from the card's rounded outline (negative inside).
func sdCard(dx, dy float64) float64 {
	const r = 23
	qx := math.Abs(dx) - (cardW/2 - r)
	qy := math.Abs(dy) - (cardH/2 - r)
	return math.Hypot(math.Max(qx, 0), math.Max(qy, 0)) + math.Min(math.Max(qx, qy), 0) - r
}

func hexc(s string, a float64) color.NRGBA {
	s = strings.TrimPrefix(s, "#")
	var v [3]uint8
	for i := 0; i < 3; i++ {
		v[i] = hexByte(s[i*2])<<4 | hexByte(s[i*2+1])
	}
	return color.NRGBA{v[0], v[1], v[2], u8(a)}
}

func hexByte(c byte) uint8 {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	}
	return 0
}

func mix(a, b color.NRGBA, t float64) color.NRGBA {
	l := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t + 0.5) }
	return color.NRGBA{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), l(a.A, b.A)}
}

// The painted ring on the deck boards where the tricks are played.
const matImg = 1100

func makeMat() *ebiten.Image {
	const pad = 50
	R := float64(matImg)/2 - pad
	im := image.NewNRGBA(image.Rect(0, 0, matImg, matImg))
	c := float64(matImg) / 2
	brass1, brass2, brass3 := hexc("ffe2a0", 1), hexc("e8b860", 1), hexc("9a6a2a", 1)
	for y := 0; y < matImg; y++ {
		for x := 0; x < matImg; x++ {
			dx, dy := float64(x)+0.5-c, float64(y)+0.5-c
			r := math.Hypot(dx, dy)
			var col color.NRGBA
			// a soft dark pool inside the ring
			in := clamp((R-r)/R, 0, 1)
			col = color.NRGBA{0, 10, 6, u8(0.22 * math.Pow(in, 0.6) * clamp((R-r)/6, 0, 1))}
			band := func(r0, w float64, top, bot color.NRGBA) {
				d := math.Abs(r - r0)
				if d < w/2+1 {
					t := clamp((r-(r0-w/2))/w, 0, 1)
					cc := mix(top, bot, t)
					// light catches the top-left of the ring
					l := 0.5 + 0.5*(-dx-dy)/(r+1e-9)/math.Sqrt2
					cc = mix(cc, brass1, 0.35*l)
					cc.A = u8(clamp(w/2+1-d, 0, 1))
					col = mix(col, color.NRGBA{cc.R, cc.G, cc.B, 255}, float64(cc.A)/255)
					col.A = max(col.A, cc.A)
				}
			}
			band(R-4, 7, brass1, brass3)
			band(R-17, 2.5, brass2, brass3)
			im.SetNRGBA(x, y, col)
		}
	}
	return ebiten.NewImageFromImage(im)
}

// A dusky vignette over the deck boards.
func makeVignette() *ebiten.Image {
	const W, H = 700, 400
	im := image.NewNRGBA(image.Rect(0, 0, W, H))
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			u, v := (float64(x)+0.5)/W, (float64(y)+0.5)/H
			d := math.Hypot((u-0.5)/0.5, (v-0.46)/0.54) / math.Sqrt2
			a1 := 0.32 * clamp((d-0.3)/0.7, 0, 1)
			e := math.Min(math.Min(u*1400, (1-u)*1400), math.Min(v*800, (1-v)*800))
			a2 := 0.45 * math.Pow(clamp(1-e/120, 0, 1), 2)
			a := 1 - (1-a1)*(1-a2)
			im.SetNRGBA(x, y, color.NRGBA{4, 12, 8, u8(a * 1.2)})
		}
	}
	return ebiten.NewImageFromImage(im)
}

// The string lights' warm pool of light on the table.
func makeLamp() *ebiten.Image {
	const W, H = 560, 400
	im := image.NewNRGBA(image.Rect(0, 0, W, H))
	warm, warm2 := color.NRGBA{255, 214, 140, 0}, color.NRGBA{255, 190, 110, 0}
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			u, v := (float64(x)+0.5)/W, (float64(y)+0.5)/H
			d := math.Hypot((u-0.5)/0.5, (v-0.3)/0.7) / math.Sqrt2 * 1.25
			var c color.NRGBA
			switch {
			case d < 0.45:
				c = mix(warm, warm2, d/0.45)
				c.A = u8(0.17 - 0.13*d/0.45)
			case d < 0.85:
				t := (d - 0.45) / 0.4
				c = mix(warm2, color.NRGBA{20, 8, 0, 0}, t)
				c.A = u8(0.04 + 0.12*t)
			default:
				c = color.NRGBA{20, 8, 0, u8(0.16)}
			}
			im.SetNRGBA(x, y, c)
		}
	}
	return ebiten.NewImageFromImage(im)
}
