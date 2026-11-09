//go:build !headless && !nosound

package main

import (
	"bytes"
	"io"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

// Music and sound effects, baked from the synthesized score in
// tools/bake/audio.js (three rooftop tunes: lo-fi hip-hop, soul and late-night jazz).

const sampleRate = 44100

type sound struct {
	mu      sync.Mutex
	ctx     *audio.Context
	sfx     map[string][]byte
	tunes   [][]byte
	music   *audio.Player
	tuneIdx int
	musicOn bool
	sfxOn   bool
	volume  float64
	fade    float64 // music fades in/out like the WebAudio gain did
	playing bool
	loaded  chan struct{}
}

var sfxNames = []string{"bags", "bid", "broken", "bust", "deal", "hover", "lose", "made", "nil", "nilmade", "play", "set", "shuffle", "slide", "sweep", "tick", "trick", "trump", "win"}

var snd = &sound{musicOn: true, sfxOn: true, volume: 0.6, loaded: make(chan struct{})}

func (s *sound) init() {
	s.ctx = audio.NewContext(sampleRate)
	s.sfx = map[string][]byte{}
	for _, n := range sfxNames {
		b, _ := assetFS.ReadFile("assets/audio/sfx-" + n + ".wav")
		if st, err := wav.DecodeWithSampleRate(sampleRate, bytes.NewReader(b)); err == nil {
			if pcm, err := io.ReadAll(st); err == nil {
				s.sfx[n] = pcm
			}
		}
	}
	go func() { // decoding the music takes a moment; do it off the main thread
		var tunes [][]byte
		for i := 0; i < 3; i++ {
			b, _ := assetFS.ReadFile("assets/audio/tune-" + itoa(i) + ".ogg")
			st, err := vorbis.DecodeWithSampleRate(sampleRate, bytes.NewReader(b))
			if err != nil {
				continue
			}
			pcm, _ := io.ReadAll(st)
			tunes = append(tunes, pcm)
		}
		s.mu.Lock()
		s.tunes = tunes
		s.mu.Unlock()
		close(s.loaded)
	}()
}

func (s *sound) play(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.sfxOn || s.ctx == nil || s.sfx[name] == nil {
		return
	}
	p := s.ctx.NewPlayerFromBytes(s.sfx[name])
	p.SetVolume(s.volume) // the 0.9 sfx bus gain is baked into the files
	p.Play()
}

func (s *sound) startMusic() {
	s.mu.Lock()
	s.playing = true
	s.mu.Unlock()
}

// setMusic turns the music on or off (it fades); it only starts playing
// once startMusic has been called.
func (s *sound) setMusic(on bool) {
	s.mu.Lock()
	s.musicOn = on
	s.mu.Unlock()
}

func (s *sound) setSfx(on bool)      { s.mu.Lock(); s.sfxOn = on; s.mu.Unlock() }
func (s *sound) setVolume(v float64) { s.mu.Lock(); s.volume = v; s.mu.Unlock() }

func (s *sound) tuneName() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.tuneIdx < len(art.tunes) {
		return art.tunes[s.tuneIdx]
	}
	return ""
}

// update runs once a frame: keeps the next tune rolling and eases the fade.
func (s *sound) update(dt float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ctx == nil || !s.playing || len(s.tunes) == 0 {
		return
	}
	target := 0.0
	if s.musicOn {
		target = 1
	}
	s.fade += (target - s.fade) * min(1, dt/0.15)
	if s.music == nil {
		s.music = s.ctx.NewPlayerFromBytes(s.tunes[s.tuneIdx%len(s.tunes)])
		s.music.Play()
	} else if !s.music.IsPlaying() {
		s.music.Close()
		s.tuneIdx = (s.tuneIdx + 1) % len(s.tunes)
		s.music = s.ctx.NewPlayerFromBytes(s.tunes[s.tuneIdx])
		s.music.Play()
	}
	s.music.SetVolume(s.volume * s.fade)
}
