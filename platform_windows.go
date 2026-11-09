//go:build windows && !headless

package main

import (
	"os"
	"syscall"
	"unsafe"
)

var (
	user32         = syscall.NewLazyDLL("user32.dll")
	kernel32       = syscall.NewLazyDLL("kernel32.dll")
	pMessageBox    = user32.NewProc("MessageBoxW")
	pFindWindow    = user32.NewProc("FindWindowW")
	pShowWindow    = user32.NewProc("ShowWindow")
	pSetForeground = user32.NewProc("SetForegroundWindow")
	pCreateMutex   = kernel32.NewProc("CreateMutexW")
)

func u16(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }

func fatal(text string) {
	pMessageBox.Call(0, uintptr(unsafe.Pointer(u16(text))), uintptr(unsafe.Pointer(u16(productName))), 0x10)
	os.Exit(1)
}

// singleInstance brings an already-open Spades window forward instead of
// opening a second one.
func singleInstance() bool {
	_, _, err := pCreateMutex.Call(0, 0, uintptr(unsafe.Pointer(u16(`Local\IroniteSpades`))))
	if err == syscall.Errno(183) { // ERROR_ALREADY_EXISTS
		if h, _, _ := pFindWindow.Call(0, uintptr(unsafe.Pointer(u16(productName+" "+version)))); h != 0 {
			pShowWindow.Call(h, 9)
			pSetForeground.Call(h)
		}
		return false
	}
	return true
}

func testFlags() (bool, bool) {
	return os.Getenv("SPADES_AUTOPILOT") != "", os.Getenv("SPADES_FAST") != ""
}
