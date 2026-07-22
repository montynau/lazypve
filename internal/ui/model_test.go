package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MontyNau/lazypve/internal/pve"
)

func TestDrillDownFlow(t *testing.T) {
	m := New(map[string]*pve.Client{"default": nil})

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(Model)

	updated, _ = m.Update(nodesMsg{
		{Cluster: "default", Node: "alpha"},
		{Cluster: "default", Node: "beta"},
	})
	m = updated.(Model)

	allGuests := guestsMsg{
		{Cluster: "default", Node: "alpha", VMID: 100, Name: "vm-a"},
		{Cluster: "default", Node: "alpha", VMID: 101, Name: "vm-b"},
		{Cluster: "default", Node: "beta", VMID: 200, Name: "vm-c"},
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

	if m.selected.Node != "alpha" {
		t.Fatalf("expected selected.Node = %q, got %q", "alpha", m.selected.Node)
	}
	if got := len(m.guestsTable.Rows()); got != 2 {
		t.Fatalf("expected 2 guest rows after drilling into alpha, got %d", got)
	}
	if !m.guestsTable.Focused() {
		t.Fatal("expected focus to move to guests table after drill-down")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	if m.selected.Node != "" {
		t.Fatalf("expected selected cleared after esc, got %+v", m.selected)
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

// TestDrillDownMultiCluster checks that filtering keys off (cluster, node)
// together, not node name alone — two clusters can have a same-named node.
func TestDrillDownMultiCluster(t *testing.T) {
	m := New(map[string]*pve.Client{"prod": nil, "lab": nil})

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(Model)

	// Both clusters happen to have a node called "alpha".
	updated, _ = m.Update(nodesMsg{
		{Cluster: "lab", Node: "alpha"},
		{Cluster: "prod", Node: "alpha"},
	})
	m = updated.(Model)

	updated, _ = m.Update(guestsMsg{
		{Cluster: "lab", Node: "alpha", VMID: 100, Name: "lab-vm"},
		{Cluster: "prod", Node: "alpha", VMID: 200, Name: "prod-vm"},
	})
	m = updated.(Model)

	// Nodes are sorted by (Cluster, Node), so "lab" sorts before "prod";
	// the cursor starts on the "lab" row.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if m.selected != (nodeKey{Cluster: "lab", Node: "alpha"}) {
		t.Fatalf("expected selected = {lab alpha}, got %+v", m.selected)
	}

	rows := m.guestsTable.Rows()
	if len(rows) != 1 {
		t.Fatalf("expected exactly 1 guest row for lab/alpha, got %d: %+v", len(rows), rows)
	}
}
