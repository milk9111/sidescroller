//go:build !dialog
// +build !dialog

package main

// openBackgroundDialog opens the native file dialog and returns the selected path.
func openBackgroundDialog() (string, error) {
	panic("background dialog not available in this build")
}
