# Spades

A Brooklyn rooftop Spades game by Ironite, written in Go with
[Ebitengine](https://ebitengine.org). Partner up against the rooftop regulars,
share one computer with up to three friends, or play together online. It ships
on its own and inside the [Ironite Launcher](https://github.com/adamglassner/launcher).

## How to play

- Four players in two teams of two; partners sit across. All 52 cards are dealt.
- Everyone bids how many tricks they'll take; partners' bids make the team's contract. Bid Nil to promise no tricks at all.
- Follow suit if you can. Spades are always trumps; the highest spade wins, or else the highest card of the suit led.
- No leading spades until they're broken (unless you hold nothing else).
- Make your contract: 10 per trick bid plus 1 per extra trick (a bag). Miss it: −10 per trick bid. Every 10 bags costs 100. Nil is worth ±100.
- First team to 500 (or 300) wins; a team at −200 loses.
- F2 starts a new game, M toggles music, F11 toggles full screen.

### Ways to play

- **Deal** – you and a rooftop regular against two more. Every game makes up new ones, each with a face, and a name and manner that fit it.
- **Play Together → Same Computer** – 2 to 4 people take turns at one PC, choose who partners whom, and get a "pass the computer" screen between turns.
- **Play Together → Online** – one player hosts, friends join by IP address (TCP port 47240). The host runs the table and sends each guest only what they can see. A guest who leaves is replaced by a computer player.

Saves and settings are kept in `%APPDATA%\Ironite\Spades\save.json`, outside the install folder.

## Play

The easiest way is **[Ironite Launcher](https://github.com/adamglassner/launcher)**: it downloads
Spades from this repository's releases, installs it and keeps it up to date.

Or download from the [latest release](../../releases/latest): `Spades-Setup-v<version>.exe`
installs it with Start menu and desktop shortcuts, or unzip `Spades-<version>-windows.zip`
anywhere and run `Spades.exe`.

## Code

| File | What it is |
|---|---|
| `main.go` | Window setup and the entry point. |
| `game.go` | Game state, card zones laid out from each player's seat, the hand script (dealing, bidding, tricks, scoring), results. |
| `internal/spades/` | The rules and the computer players (no graphics; has tests). |
| `net.go` | Online play: host/guest protocol (JSON lines over TCP). |
| `setup.go`, `setup_ui.go` | Play Together: seating and partners, lobby, pass screen, text fields, face picker. |
| `faces.go`, `cameo.go` | The rooftop regulars: random faces, fitting names and table talk, and their portraits drawn in Go. |
| `ui.go`, `panels.go` | The table, brick bar, bid box, hand and score boxes; title, rules, credits and result panels. |
| `draw.go`, `glyphs.go`, `assets.go` | Vector drawing, text and suit symbols, embedded art. |
| `sound.go`, `save.go`, `common.go` | Music and sound effects, the save file, shared bits. |
| `assets/` | Card faces, props, deck boards, string lights, fonts, music and sound effects. |
| `tools/bake/` | Draws the card art and store art and synthesizes the music and sounds into `assets/` (`node tools/bake/bake.mjs`, needs Playwright and ffmpeg). |
| `tools/mkres/` | Writes the Windows icon and version info into the .exe. |
| `installer/` | Inno Setup scripts for the standalone `Spades-Setup-v<version>.exe`. |

`headless_test_main.go` and `sound_stub.go` are only used by test builds:
`go run -tags headless .` plays a solo game, two online games between a host
and two guests over TCP on this computer (including a guest leaving mid-game),
and a shared-computer game through the pass screen. `go test ./internal/spades`
checks the rules and plays a few hundred computer games.

## Building

You need Go 1.24 or newer and bash (Git Bash works on Windows). No C compiler is needed.

```bash
./build.sh
```

This writes `dist/Spades/Spades.exe` with its README, CREDITS, CHANGELOG and license files.
The version comes from the top entry of `CHANGELOG.md`. For the installer, install
Inno Setup 6.3 or newer, then: `ISCC.exe /DGameVersion=<version> installer\SpadesSetup.iss`

## Releasing

Add a new version section at the top of `CHANGELOG.md` and push to `main`. The
release workflow builds the game, publishes `Spades-<version>-windows.zip` as a
GitHub release (Ironite Launcher installs and updates from it), then builds and
attaches the standalone `Spades-Setup-v<version>.exe`.

(c) 2026 Ironite
