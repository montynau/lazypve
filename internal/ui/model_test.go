package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDrillDownFlow(t *testing.T) {
	m := New(nil)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(Model)

	updated, _ = m.Update(nodesMsg{{Node: "alpha"}, {Node: "beta"}})
	m = updated.(Model)

	allGuests := guestsMsg{
		{Node: "alpha", VMID: 100, Name: "vm-a"},
		{Node: "alpha", VMID: 101, Name: "vm-b"},
		{Node: "beta", VMID: 200, Name: "vm-c"},
	}
	updated, _ = m.Update(allGuests)
	m = updated.(Model)

	if got := len(m.guestsTable.Rows()); got != 3 {
		t.Fatalf("expected 3 guest rows before drill-down, got %d", got)
	}
	if !m.nodesTable.Focused() {
		t.Fatal("expected nodes table to be focused initially")
	}

	// Cursor starts on the first (alphabetically first) node, "alpha".
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if m.selectedNode != "alpha" {
		t.Fatalf("expected selectedNode = %q, got %q", "alpha", m.selectedNode)
	}
	if got := len(m.guestsTable.Rows()); got != 2 {
		t.Fatalf("expected 2 guest rows after drilling into alpha, got %d", got)
	}
	if !m.guestsTable.Focused() {
		t.Fatal("expected focus to move to guests table after drill-down")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	if m.selectedNode != "" {
		t.Fatalf("expected selectedNode cleared after esc, got %q", m.selectedNode)
	}
	if got := len(m.guestsTable.Rows()); got != 3 {
		t.Fatalf("expected 3 guest rows after clearing filter, got %d", got)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)

	if !m.nodesTable.Focused() {
		t.Fatal("expected tab to move focus back to nodes table")
	}
}
