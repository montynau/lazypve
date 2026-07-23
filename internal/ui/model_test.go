package ui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MontyNau/lazypve/internal/pve"
)

func TestDrillDownFlow(t *testing.T) {
	m := New(map[string]*pve.Client{"default": nil})

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(Model)

	updated, _ = m.Update(nodesMsg{nodes: []node{
		{Cluster: "default", Node: "alpha"},
		{Cluster: "default", Node: "beta"},
	}})
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
	if !m.nodesTable.Focused() {
		t.Fatal("expected esc to move focus back to nodes table, mirroring enter")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)

	if !m.guestsTable.Focused() {
		t.Fatal("expected tab to move focus to guests table")
	}
}

// TestEscClearsSort checks that esc resets the focused table's sort back to
// natural (fetch) order, independent of whether a drill-down filter is
// active.
func TestEscClearsSort(t *testing.T) {
	m := New(map[string]*pve.Client{"default": nil})

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(Model)

	updated, _ = m.Update(nodesMsg{nodes: []node{
		{Cluster: "default", Node: "beta"},
		{Cluster: "default", Node: "alpha"},
	}})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("]")})
	m = updated.(Model)

	if m.nodesSort == -1 {
		t.Fatal("expected nodesSort to be set after pressing ']'")
	}
	if got := m.nodesTable.Rows()[0][0]; got != "  alpha" {
		t.Fatalf("expected sorted order to put alpha first, got row %q", got)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	if m.nodesSort != -1 {
		t.Fatalf("expected esc to clear nodesSort, got %d", m.nodesSort)
	}
	if got := m.nodesTable.Rows()[0][0]; got != "  beta" {
		t.Fatalf("expected esc to restore fetch order (beta first), got row %q", got)
	}
}

// TestPartialClusterFailure checks that one cluster failing to fetch doesn't
// blank the UI or silently drop the warning — the other cluster's nodes
// still show, and the failure is surfaced instead of just vanishing.
func TestPartialClusterFailure(t *testing.T) {
	m := New(map[string]*pve.Client{"prod": nil, "lab": nil})

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(Model)

	updated, _ = m.Update(nodesMsg{
		nodes:       []node{{Cluster: "prod", Node: "pve1"}},
		clusterErrs: map[string]error{"lab": errors.New("connection refused")},
	})
	m = updated.(Model)

	if m.err != nil {
		t.Fatalf("expected no top-level error when one of several clusters is still reachable, got %v", m.err)
	}
	if len(m.nodes) != 1 {
		t.Fatalf("expected the reachable cluster's node to still be shown, got %+v", m.nodes)
	}

	warn := m.clusterWarning()
	if !strings.Contains(warn, "lab") {
		t.Fatalf("expected cluster warning to mention the failing cluster, got %q", warn)
	}
	if !strings.Contains(m.View(), "lab") {
		t.Fatal("expected the rendered view to surface the cluster warning")
	}

	// Once "lab" recovers, the next nodesMsg carries no error for it, and the
	// warning must clear rather than sticking around from the last failure.
	updated, _ = m.Update(nodesMsg{
		nodes: []node{{Cluster: "prod", Node: "pve1"}, {Cluster: "lab", Node: "pve2"}},
	})
	m = updated.(Model)

	if warn := m.clusterWarning(); warn != "" {
		t.Fatalf("expected warning to clear once the cluster recovers, got %q", warn)
	}
}

// TestDrillDownMultiCluster checks that filtering keys off (cluster, node)
// together, not node name alone — two clusters can have a same-named node.
func TestDrillDownMultiCluster(t *testing.T) {
	m := New(map[string]*pve.Client{"prod": nil, "lab": nil})

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(Model)

	// Both clusters happen to have a node called "alpha".
	updated, _ = m.Update(nodesMsg{nodes: []node{
		{Cluster: "lab", Node: "alpha"},
		{Cluster: "prod", Node: "alpha"},
	}})
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
