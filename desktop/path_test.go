package main

import (
	"os"
	"testing"
)

func TestFixLocale(t *testing.T) {
	t.Setenv("LANG", "")
	t.Setenv("LC_ALL", "")
	fixLocale()
	if got := os.Getenv("LANG"); got != "en_US.UTF-8" {
		t.Errorf("LANG = %q", got)
	}
	t.Setenv("LANG", "de_DE.UTF-8")
	fixLocale()
	if got := os.Getenv("LANG"); got != "de_DE.UTF-8" {
		t.Errorf("an existing locale was replaced by %q", got)
	}
}
