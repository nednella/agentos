//go:build !darwin

package cli

func stdinIsTerminal() bool { return true }
