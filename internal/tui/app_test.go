package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/nasoma/atcli/internal/config"
	"testing"
)

func TestPurchaseNeedsConfirmation(t *testing.T) {
	m := model{cfg: config.Default(), env: "sandbox", screen: "dashboard"}
	v, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = v.(model)
	if m.screen != "send" {
		t.Fatal(m.screen)
	}
	m.fields[0].SetValue("0712345678")
	m.fields[1].SetValue("100")
	v, c := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = v.(model)
	if m.screen != "confirm" || c != nil {
		t.Fatal("purchase before confirmation")
	}
	// The form Enter only opens confirmation; another explicit Enter submits.
	v, c = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = v.(model)
	if c == nil || !m.busy {
		t.Fatal("Enter did not submit from confirmation")
	}
	_, c = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if c != nil {
		t.Fatal("duplicate command while busy")
	}
}

func TestConfirmationEditAndCancel(t *testing.T) {
	for _, tc := range []struct {
		key    tea.KeyMsg
		screen string
	}{
		{tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}, "send"},
		{tea.KeyMsg{Type: tea.KeyEsc}, "dashboard"},
		{tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}, "confirm"},
	} {
		m := model{cfg: config.Default(), env: "production", screen: "confirm"}
		v, c := m.Update(tc.key)
		got := v.(model)
		if c != nil || got.busy || got.screen != tc.screen {
			t.Fatalf("unexpected confirmation action: %s", got.screen)
		}
	}
}
