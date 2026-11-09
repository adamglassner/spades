// The rooftop regulars who might sit down at your table. Every face is made up
// fresh from a seed, and its name, title and way of talking always fit the
// face: a lady gets a lady's name, a cook in his toque is "Chef", a grandmother
// with silver hair is "Nana". (The drawing itself is in cameo.go.)
package main

import (
	"image/color"
	"math/rand/v2"
	"strconv"
	"strings"
)

// Face is all a seat needs to share over the network: the look, the name and
// the manner are all worked out from it. As text it is "f:<seed>" or "m:<seed>".
type Face struct {
	Female bool
	Seed   uint64
}

func (f Face) String() string {
	if f.Female {
		return "f:" + strconv.FormatUint(f.Seed, 10)
	}
	return "m:" + strconv.FormatUint(f.Seed, 10)
}

func parseFace(s string) (Face, bool) {
	if len(s) < 3 || s[1] != ':' || (s[0] != 'f' && s[0] != 'm') {
		return Face{}, false
	}
	n, err := strconv.ParseUint(s[2:], 10, 64)
	if err != nil {
		return Face{}, false
	}
	return Face{Female: s[0] == 'f', Seed: n}, true
}

func faceOK(s string) bool { _, ok := parseFace(s); return ok }

/* ---------------- the look ---------------- */

type Look struct {
	Lady, Stern, Smile      bool
	Skin, Hair, Brow, Lip   color.RGBA
	HairStyle               string // bob, bun, waves, curls | short, side, bald, curly
	Top                     string // stripes, blouse, turtleneck, suit, chef, cardigan
	Coat, Dark, Accent      color.RGBA
	Stripe                  color.RGBA
	Neck                    string // scarf, pearls, bowtie, tie, kerchief, ""
	NeckColor               color.RGBA
	Hat                     string // beret, cloche, cap, fedora, toque, ""
	HatColor                color.RGBA
	Mustache                string // pencil, handlebar, walrus, ""
	Beard                   bool
	Glasses                 string // round, cateye, ""
	Mole, Earrings, Freckle bool
}

func rgb(s string) color.RGBA {
	s = strings.TrimPrefix(s, "#")
	v := func(i int) uint8 { n, _ := strconv.ParseUint(s[i:i+2], 16, 8); return uint8(n) }
	return color.RGBA{v(0), v(2), v(4), 255}
}

func shade(c color.RGBA, k float64) color.RGBA {
	f := func(v uint8) uint8 { return uint8(min(255, float64(v)*k)) }
	return color.RGBA{f(c.R), f(c.G), f(c.B), 255}
}

func choose[T any](r *rand.Rand, a ...T) T { return a[r.IntN(len(a))] }

var skinTones = []string{"#f6dbc4", "#f2d4b6", "#f1d0b5", "#eec3a2", "#e3b48e", "#d6a27a", "#c98f66", "#a8704a", "#8a5a3a", "#6e452c"}

func (f Face) rng(salt uint64) *rand.Rand { return rand.New(rand.NewPCG(f.Seed, salt)) }

// look builds a regular's appearance from their face. Ladies and gentlemen
// draw from different wardrobes, so a face always matches its name.
func (f Face) look() Look {
	r := f.rng(0xca7e)
	l := Look{Skin: rgb(choose(r, skinTones...)), Stern: r.IntN(10) < 3}
	l.Smile = !l.Stern && r.IntN(10) < 6
	l.Freckle = r.IntN(10) < 2
	grey := r.IntN(10) < 3
	if f.Female {
		l.Lady = true
		if grey {
			l.Hair = rgb(choose(r, "#b8b6bc", "#d6d2cc", "#9c9aa2", "#e8e4de"))
			l.Brow = shade(l.Hair, .7)
		} else {
			l.Hair = rgb(choose(r, "#2a1e18", "#4a2e1c", "#6b3f22", "#8a5a30", "#b5651d", "#d8b46a", "#e6c88a", "#1c1414", "#a8461e"))
		}
		l.HairStyle = choose(r, "bob", "bob", "bun", "waves", "curls")
		l.Lip = rgb(choose(r, "#b3283a", "#c0392b", "#9c2a40", "#c25a5a", "#a0404a"))
		l.Top = choose(r, "stripes", "blouse", "blouse", "turtleneck", "cardigan")
		l.Coat = rgb(choose(r, "#2a3a5c", "#6e1f2a", "#2f5a4a", "#c9a24c", "#4b2a5c", "#1e1e24", "#b3473a", "#3a6a8a", "#e8dcc8", "#7a4a6a"))
		l.Stripe = rgb(choose(r, "#1f2f55", "#1f2f55", "#a3242b"))
		l.Accent = rgb(choose(r, "#f4ecd8", "#e8b4b8", "#d9b44a", "#ffffff", "#b3283a"))
		switch n := r.IntN(10); {
		case n < 3:
			l.Neck = "scarf"
			l.NeckColor = rgb(choose(r, "#b3283a", "#e8b04a", "#3a7a9a", "#7a3a7a", "#2f7a5a", "#e07a5f"))
		case n < 6:
			l.Neck = "pearls"
		}
		switch n := r.IntN(100); {
		case n < 30:
			l.Hat = "beret"
			l.HatColor = rgb(choose(r, "#1e1e24", "#b3283a", "#2a3a5c", "#6e1f2a", "#c9a24c"))
		case n < 52:
			l.Hat = "cloche"
			l.HatColor = rgb(choose(r, "#6e1f2a", "#2a3a5c", "#3a2a2a", "#c9a27a", "#4b2a5c", "#2f5a4a"))
		}
		l.Earrings = r.IntN(10) < 5
		l.Mole = r.IntN(10) < 2
		if r.IntN(100) < 18 {
			l.Glasses = "cateye"
		}
		if grey && r.IntN(3) == 0 {
			l.Glasses = "round"
		}
		if l.Hat != "" && l.HairStyle == "bun" { // a hat would hide the bun: let the hair show
			l.HairStyle = choose(r, "bob", "waves", "curls")
		}
	} else {
		if grey {
			l.Hair = rgb(choose(r, "#e8e4dc", "#c9c4bc", "#a8a4a0"))
			l.Brow = shade(l.Hair, .75)
		} else {
			l.Hair = rgb(choose(r, "#2a1e18", "#4a2e1c", "#6b3f22", "#8a5a30", "#b5651d", "#c8a060", "#1c1414"))
		}
		l.HairStyle = choose(r, "short", "side", "side", "curly")
		if grey && r.IntN(2) == 0 {
			l.HairStyle = "bald"
		}
		l.Top = choose(r, "stripes", "suit", "suit", "turtleneck", "cardigan", "chef")
		l.Coat = rgb(choose(r, "#2a3a5c", "#3b2a1e", "#2f4a3a", "#4a2a2a", "#2a2a3a", "#5a4a3a", "#6b3a28", "#3a3a4a", "#7a6a50"))
		l.Stripe = rgb(choose(r, "#1f2f55", "#1f2f55", "#a3242b"))
		l.Accent = rgb(choose(r, "#f4ecd8", "#ffffff", "#e8e0c8"))
		switch l.Top {
		case "suit":
			l.Neck = choose(r, "tie", "bowtie", "bowtie")
			l.NeckColor = rgb(choose(r, "#b3283a", "#1e1e24", "#2a3a5c", "#c9a24c", "#2f5a4a", "#6e1f2a"))
		case "chef":
			l.Neck = "kerchief"
			l.NeckColor = rgb(choose(r, "#b3283a", "#2a3a5c"))
			l.Hat = "toque"
		case "stripes", "turtleneck":
			if r.IntN(3) == 0 {
				l.Neck = "scarf"
				l.NeckColor = rgb(choose(r, "#b3283a", "#e8b04a", "#3a7a9a", "#2a2a3a"))
			}
		}
		if l.Hat == "" {
			switch n := r.IntN(100); {
			case n < 30:
				l.Hat = "beret"
				l.HatColor = rgb(choose(r, "#1e1e24", "#2a3a5c", "#6e1f2a", "#3a3a3a"))
			case n < 45:
				l.Hat = "cap"
				l.HatColor = rgb(choose(r, "#5a4a3a", "#3a3a4a", "#6b5a3a", "#4a3a2a"))
			case n < 58:
				l.Hat = "fedora"
				l.HatColor = rgb(choose(r, "#3a3a3a", "#5a4a3a", "#2a2a3a", "#7a6a50"))
			}
		}
		switch n := r.IntN(100); {
		case n < 22:
			l.Mustache = "pencil"
		case n < 40:
			l.Mustache = "handlebar"
		case n < 52:
			l.Mustache = "walrus"
		}
		l.Beard = r.IntN(100) < 18
		if r.IntN(100) < 24 {
			l.Glasses = "round"
		}
	}
	l.Dark = shade(l.Coat, .62)
	return l
}

/* ---------------- names and manners ---------------- */

// manner is the kind of regular a face suggests; it picks the title and the turns of phrase.
func (f Face) manner() string {
	l := f.look()
	grey := l.Brow != (color.RGBA{}) // only silver hair sets the brow colour
	if f.Female {
		switch {
		case grey:
			return "mamie"
		case l.Hat == "beret" || l.Top == "stripes":
			return "artiste"
		case l.Neck == "pearls" || l.Hat == "cloche":
			return "madame"
		}
		return "mademoiselle"
	}
	switch {
	case l.Hat == "toque":
		return "chef"
	case grey:
		return "papi"
	case l.Glasses == "round":
		return "professeur"
	case l.Top == "stripes":
		return "capitaine"
	case l.Hat == "beret":
		return "artiste"
	}
	return "monsieur"
}

var mannerNames = map[string]struct {
	titles []string
	names  []string
}{
	"mamie":        {[]string{"Nana", "Auntie"}, []string{"Rose", "Dot", "Fran", "Bev", "Gloria", "Marge", "Connie", "Ida"}},
	"artiste":      {[]string{"DJ", "Lil'"}, []string{"Kiki", "Jazz", "Luna", "Roxy", "Nova", "Bree", "Zoe", "Indie"}},
	"madame":       {[]string{"Miss", "Ms."}, []string{"Denise", "Carmen", "Yolanda", "Theresa", "Loretta", "Angela", "Diane", "Renee"}},
	"mademoiselle": {[]string{"Cousin", "Lil'"}, []string{"Jess", "Tasha", "Nikki", "Bianca", "Gina", "Maria", "Keisha", "Dani"}},
	"chef":         {[]string{"Chef"}, []string{"Sal", "Vinny", "Manny", "Rocco", "Tony", "Ray", "Frankie"}},
	"papi":         {[]string{"Pop", "Uncle"}, []string{"Morty", "Lou", "Abe", "Murray", "Herb", "Earl", "Benny"}},
	"professeur":   {[]string{"Doc", "Professor"}, []string{"Goldberg", "Russo", "Kaplan", "Esposito", "Washington", "Fischer", "Rosen"}},
	"capitaine":    {[]string{"Cap"}, []string{"Mike", "Jimmy", "Eddie", "Pat", "Dom", "Ricky"}},
	"artiste_m":    {[]string{"MC", "Slim"}, []string{"Marcus", "Andre", "Dre", "Jay", "Malik", "Theo", "Ty"}},
	"monsieur":     {[]string{"Big", "Little"}, []string{"Joey", "Nicky", "Mikey", "Carlos", "Danny", "Kevin", "Luis", "Tommy"}},
}

func (f Face) mannerKey() string {
	m := f.manner()
	if m == "artiste" && !f.Female {
		return "artiste_m"
	}
	return m
}

// nameFor gives the face a name (first name, and the full name with a title),
// avoiding names already at the table.
func (f Face) nameFor(avoid map[string]bool) (name, full string, ok bool) {
	mn := mannerNames[f.mannerKey()]
	r := f.rng(0x9a3e5)
	names := append([]string(nil), mn.names...)
	r.Shuffle(len(names), func(i, j int) { names[i], names[j] = names[j], names[i] })
	for _, n := range names {
		if !avoid[n] {
			return n, choose(r, mn.titles...) + " " + n, true
		}
	}
	return "", "", false
}

/* ---------------- table talk ---------------- */

// {p}: whoever the line is about; {n}: a bid.
var mannerLines = map[string]map[string][]string{
	"mamie": {
		"bid":     {"I'll take {n}, sweetheart.", "{n}. Don't rush me.", "Put me down for {n}, hon."},
		"nil":     {"Nil. Nana's feeling lucky.", "Nothing for me, dear. Nil."},
		"broken":  {"Oh, there go the spades.", "Spades are out, kids."},
		"trump":   {"Nana cuts you, dear.", "Sorry, hon. Spade."},
		"trick":   {"That's Nana's.", "Mine, sweetie."},
		"trumped": {"Fresh!", "Oh, you little stinker."},
		"made":    {"Made it, like always.", "See? Nana knows."},
		"set":     {"Oy. Set again.", "Who dealt me this?"},
		"bust":    {"Oh, phooey. I took one.", "There goes my nil, dear."},
		"nilmade": {"Nil! Nana's still got it.", "Not one trick. Told you."},
		"setThem": {"Set! Serves you right, dears."},
		"gotcha":  {"Gotcha, {p}!", "Sorry, {p}, dear."},
		"bags":    {"Bags? Don't talk to me about bags."},
		"win":     {"Now who wants cannoli?"},
		"lose":    {"Next week, I'm bringing my lucky deck."},
	},
	"artiste": {
		"bid":     {"{n}. Feeling it.", "Put me down for {n}.", "{n}, no cap."},
		"nil":     {"Nil. Watch this.", "Going nil. Trust the vibe."},
		"broken":  {"Spades are live!", "Okay, spades are out."},
		"trump":   {"Drop the beat. Spade.", "Cut!"},
		"trick":   {"Mine.", "That's a vibe.", "Run it."},
		"trumped": {"Rude!", "Okay, I see you."},
		"made":    {"Called it.", "Made it, easy."},
		"set":     {"Off-beat. Set.", "Ugh, set."},
		"bust":    {"Nooo, my nil!", "Took one. Embarrassing."},
		"nilmade": {"Nil made! Encore!", "Zero tricks. Art."},
		"setThem": {"Ooh, they got set!"},
		"gotcha":  {"Busted, {p}!", "Sorry not sorry, {p}."},
		"bags":    {"Bags! Stop feeding us."},
		"win":     {"That was a whole set!"},
		"lose":    {"Rematch on the roof next week."},
	},
	"madame": {
		"bid":     {"{n}, please.", "I'll bid {n}.", "{n}. Thank you."},
		"nil":     {"Nil. Don't look at me like that.", "I'll go nil, thank you."},
		"broken":  {"Spades are broken.", "There they go."},
		"trump":   {"I'll trump that, thank you.", "Spade. Sorry, not sorry."},
		"trick":   {"Thank you.", "I'll take that.", "Lovely."},
		"trumped": {"Excuse me?", "Well, that was rude."},
		"made":    {"Made it. Naturally.", "Exactly as bid."},
		"set":     {"Well. That's unfortunate.", "Set? How embarrassing."},
		"bust":    {"I took one. Wonderful.", "So much for nil."},
		"nilmade": {"Nil, made. Of course.", "Not a single trick."},
		"setThem": {"Set? What a pity. For them."},
		"gotcha":  {"Caught you, {p}.", "Oh dear, {p}."},
		"bags":    {"Ten bags. How tacky."},
		"win":     {"Somebody pour me a spritz."},
		"lose":    {"Well played, all of you."},
	},
	"mademoiselle": {
		"bid":     {"{n}!", "Uh, {n}? Yeah, {n}.", "{n}, let's go!"},
		"nil":     {"Nil! I'm doing it!", "Going nil, don't laugh."},
		"broken":  {"Spades are broken!", "Ooh, spades!"},
		"trump":   {"Spade! Sorry!", "Chop!"},
		"trick":   {"Mine!", "Yes!", "Let's go!"},
		"trumped": {"No way!", "Seriously?"},
		"made":    {"We made it!", "Yes! Made it!"},
		"set":     {"Ugh, set!", "No no no."},
		"bust":    {"Nooo, I took one!", "My nil! Ugh."},
		"nilmade": {"Nil made! Did you see?", "Zero! Yes!"},
		"setThem": {"They got set! Ha!"},
		"gotcha":  {"Got you, {p}!", "Oops, {p}!"},
		"bags":    {"Bags?! Again?"},
		"win":     {"We won! Selfie!"},
		"lose":    {"Okay, rematch. Now."},
	},
	"chef": {
		"bid":     {"{n}, order up.", "Gimme {n}.", "{n}. Fresh."},
		"nil":     {"Nil. Nothing on my plate.", "Nil, capisce?"},
		"broken":  {"Spades are cookin' now!", "Spades are out!"},
		"trump":   {"Chop chop! Spade!", "Into the fryer."},
		"trick":   {"Order up!", "Mine.", "That's a slice."},
		"trumped": {"Ay, you burned me!", "Whaddaya doin'?"},
		"made":    {"Perfect, like my gravy.", "Made it, no sweat."},
		"set":     {"Burnt. Completely burnt.", "Ay, set!"},
		"bust":    {"Ay! I took one.", "There goes the nil."},
		"nilmade": {"Nil! Chef's kiss.", "Not a crumb on my plate."},
		"setThem": {"Ha! Burnt 'em!"},
		"gotcha":  {"Gotcha, {p}!", "Sorry, {p}, kitchen's closed."},
		"bags":    {"Too many bags! Ay!"},
		"win":     {"Pizza's on me!"},
		"lose":    {"Next time I cook, you lose."},
	},
	"papi": {
		"bid":     {"{n}.", "Gimme {n}, kid.", "{n}. Next."},
		"nil":     {"Nil. Don't argue with me.", "Nil, and I mean it."},
		"broken":  {"Spades are out.", "Here we go, spades."},
		"trump":   {"Spade. Learn something.", "I'll cut that."},
		"trick":   {"Mine.", "Heh.", "Yeah, yeah."},
		"trumped": {"Wise guy.", "Hey! Respect your elders."},
		"made":    {"Made it. Like '78.", "Still got it."},
		"set":     {"Aw, nuts.", "Set. Feh."},
		"bust":    {"Feh. I took one.", "So much for nil."},
		"nilmade": {"Nil! Not bad for an old man.", "Zero. Like I said."},
		"setThem": {"Set! Heh. Kids."},
		"gotcha":  {"Gotcha, {p}.", "Sorry, kid."},
		"bags":    {"Bags. In my day we counted 'em."},
		"win":     {"Not bad for an old-timer!"},
		"lose":    {"Eh. Next Sunday."},
	},
	"professeur": {
		"bid":     {"{n}, by my estimate.", "{n}. Probably.", "I calculate {n}."},
		"nil":     {"Nil. The math says yes.", "Nil. A calculated risk."},
		"broken":  {"Spades are now in play.", "Ah, spades are broken."},
		"trump":   {"Trump. Elementary.", "A spade, logically."},
		"trick":   {"Indeed.", "Logical.", "Mine, I believe."},
		"trumped": {"Hm. Unexpected.", "Curious."},
		"made":    {"Contract fulfilled.", "As predicted."},
		"set":     {"A miscalculation.", "Hm. Set."},
		"bust":    {"An outlier. I took one.", "Nil, falsified."},
		"nilmade": {"Nil, as theorized.", "Zero tricks. Q.E.D."},
		"setThem": {"Their contract failed. As expected."},
		"gotcha":  {"Disproven, {p}.", "Sorry, {p}. Math."},
		"bags":    {"The bags compound. Alarming."},
		"win":     {"A tidy result."},
		"lose":    {"I shall review my notes."},
	},
	"capitaine": {
		"bid":     {"{n}, aye.", "Steady at {n}.", "{n}. Full speed."},
		"nil":     {"Nil. Running silent.", "Going nil. Batten down."},
		"broken":  {"Spades on the horizon!", "Spades are out!"},
		"trump":   {"Broadside! Spade!", "Ram it!"},
		"trick":   {"Aye.", "Mine!", "Steady."},
		"trumped": {"Torpedoed!", "Mutiny!"},
		"made":    {"Made port!", "Right on course."},
		"set":     {"Ran aground.", "Set. Rough seas."},
		"bust":    {"Sprung a leak. Took one.", "Nil's sunk."},
		"nilmade": {"Nil! Smooth sailing.", "Not a scratch on the hull."},
		"setThem": {"They ran aground!"},
		"gotcha":  {"Sunk you, {p}!", "Gotcha, {p}."},
		"bags":    {"Bags! We're taking on water."},
		"win":     {"Land ho! We won!"},
		"lose":    {"We'll sail again."},
	},
	"artiste_m": {
		"bid":     {"{n}. Easy.", "Put me down for {n}.", "{n}, no doubt."},
		"nil":     {"Nil. Watch the master.", "Nil. Trust me."},
		"broken":  {"Spades are live!", "Spades in the building!"},
		"trump":   {"Spade. Mic drop.", "Cut it."},
		"trick":   {"Mine.", "Run it back.", "That's me."},
		"trumped": {"Oh, it's like that?", "Cold."},
		"made":    {"Made it, easy.", "Told you."},
		"set":     {"Set. Brutal.", "Ugh. Set."},
		"bust":    {"Took one. My bad.", "Nil's gone."},
		"nilmade": {"Nil made! Legendary.", "Zero. Like a boss."},
		"setThem": {"Set! That's cold."},
		"gotcha":  {"Gotcha, {p}!", "Bye, {p}!"},
		"bags":    {"Bags! Chill with that."},
		"win":     {"That's the whole album!"},
		"lose":    {"Rematch. Same roof, same time."},
	},
	"monsieur": {
		"bid":     {"{n}.", "Gimme {n}.", "{n}, easy."},
		"nil":     {"Nil. Fuhgeddaboudit.", "Going nil, watch."},
		"broken":  {"Spades are out!", "Spades are broken."},
		"trump":   {"Spade! Bada bing.", "Cut!"},
		"trick":   {"Mine.", "That's me.", "Yeah!"},
		"trumped": {"Ay, come on!", "You gotta be kiddin'."},
		"made":    {"Made it!", "Right on the money."},
		"set":     {"Ay, set!", "Who dealt this?"},
		"bust":    {"Ay, I took one.", "There goes the nil."},
		"nilmade": {"Nil made! Ay-oh!", "Zero! Fuhgeddaboudit."},
		"setThem": {"Set! Ay-oh!"},
		"gotcha":  {"Gotcha, {p}!", "Sorry, {p}!"},
		"bags":    {"Ten bags? Ay!"},
		"win":     {"Drinks on the roof!"},
		"lose":    {"Next week, you're goin' down."},
	},
}

func (f Face) lines(kind string) []string { return mannerLines[f.mannerKey()][kind] }

/* ---------------- picking regulars ---------------- */

// folk is one computer player picked for a game.
type folk struct{ face, name, full string }

// randomFolks makes up n regulars with different names (a mix of ladies and gentlemen).
func randomFolks(rng *rand.Rand, n int, avoidFaces, avoidNames map[string]bool) []folk {
	var out []folk
	men := 0
	for tries := 0; len(out) < n && tries < 500; tries++ {
		f := Face{Female: rng.IntN(2) == 0, Seed: rng.Uint64()}
		if len(out) == n-1 && n > 1 && men == 0 {
			f.Female = false // always at least one gentleman and one lady
		} else if len(out) == n-1 && n > 1 && men == len(out) {
			f.Female = true
		}
		if avoidFaces[f.String()] {
			continue
		}
		name, full, ok := f.nameFor(avoidNames)
		if !ok {
			continue
		}
		avoidNames[name] = true
		if !f.Female {
			men++
		}
		out = append(out, folk{face: f.String(), name: name, full: full})
	}
	return out
}

// pickerFaces are the faces offered for your own player online and on a shared computer.
var pickerFaces = func() []string {
	var out []string
	r := rand.New(rand.NewPCG(2026, 11))
	for len(out) < 10 {
		f := Face{Female: len(out)%2 == 0, Seed: r.Uint64()}
		if f.mannerKey() == "chef" && len(out) > 6 {
			continue
		}
		out = append(out, f.String())
	}
	return out
}()
