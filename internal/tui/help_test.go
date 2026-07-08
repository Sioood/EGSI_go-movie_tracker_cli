package tui

import (
	"strings"
	"testing"
)

func TestHelpViewContainsAllShortcuts(t *testing.T) {
	m := New(Options{MovieService: newFakeMovieService()})
	view := m.helpView()

	for _, sc := range AllShortcuts() {
		if !strings.Contains(view, sc.Keys) {
			t.Errorf("help view missing keys %q", sc.Keys)
		}
		if !strings.Contains(view, sc.Action) {
			t.Errorf("help view missing action for %q: %q", sc.Keys, sc.Action)
		}
	}
}

func TestHelpSectionsNonEmpty(t *testing.T) {
	if len(helpSections) == 0 {
		t.Fatal("expected help sections")
	}
	if len(AllShortcuts()) < 20 {
		t.Fatalf("expected at least 20 shortcuts, got %d", len(AllShortcuts()))
	}
}
