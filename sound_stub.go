//go:build headless || nosound

// Silent stand-in for test builds on machines without audio libraries.
package main

type sound struct{}

var snd = &sound{}

func (s *sound) init()               {}
func (s *sound) play(name string)    {}
func (s *sound) startMusic()         {}
func (s *sound) setMusic(on bool)    {}
func (s *sound) setSfx(on bool)      {}
func (s *sound) setVolume(v float64) {}
func (s *sound) update(dt float64)   {}
func (s *sound) tuneName() string    { return "" }
