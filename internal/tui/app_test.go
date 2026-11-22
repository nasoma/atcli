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
	m.busy = true
	_, c = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if c != nil {
		t.Fatal("duplicate command while busy")
	}
}
