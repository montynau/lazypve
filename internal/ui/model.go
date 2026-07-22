package ui

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/MontyNau/lazypve/internal/pve"
)

const refreshInterval = 3 * time.Second

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	errStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	activeLabelStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205"))

	inactiveLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240"))
)

type focusArea int

const (
	focusNodes focusArea = iota
	focusGuests
)

// node is a pve.Node tagged with which cluster it came from, since lazypve
// can poll several clusters at once.
type node struct {
	Cluster string
	Node    string
	Status  string
	CPU     float64
	Mem     int64
	MaxMem  int64
	Uptime  int64
}

// nodeKey identifies a single node across clusters (node names are only
// unique within a cluster, not globally).
type nodeKey struct {
	Cluster string
	Node    string
}

type nodesMsg []node
type errMsg struct{ err error }
type tickMsg time.Time

type Model struct {
	clients map[string]*pve.Client

	width, height int

	nodes    []node
	guests   []guest
	selected nodeKey // zero value = no drill-down filter

	nodesTable  table.Model
	guestsTable table.Model
	focus       focusArea

	err     error
	loading bool
}

func New(clients map[string]*pve.Client) Model {
	multi := len(clients) > 1
	nodesTable := table.New(
		table.WithColumns(nodeColumns(multi)),
		table.WithFocused(true),
		table.WithHeight(6),
	)
	guestsTable := table.New(
		table.WithColumns(guestColumns(multi)),
		table.WithHeight(10),
	)

	return Model{
		clients:     clients,
		loading:     true,
		nodesTable:  nodesTable,
		guestsTable: guestsTable,
		focus:       focusNodes,
	}
}

func (m Model) multiCluster() bool {
	return len(m.clients) > 1
}

func (m Model) Init() tea.Cmd {
	return m.fetchNodes()
}

func (m Model) fetchNodes() tea.Cmd {
	clients := m.clients
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var nodes []node
		var errs []error
		for name, client := range clients {
			raw, err := client.GetNodes(ctx)
			if err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", name, err))
				continue
			}
			for _, n := range raw {
				nodes = append(nodes, node{
					Cluster: name, Node: n.Node, Status: n.Status,
					CPU: n.CPU, Mem: n.Mem, MaxMem: n.MaxMem, Uptime: n.Uptime,
				})
			}
		}

		if len(nodes) == 0 && len(errs) > 0 {
			return errMsg{errors.Join(errs...)}
		}

		slices.SortFunc(nodes, func(a, b node) int {
			return cmp.Or(
				cmp.Compare(a.Cluster, b.Cluster),
				cmp.Compare(a.Node, b.Node),
			)
		})
		return nodesMsg(nodes)
	}
}

func tick() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeTables()

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "tab":
			m.toggleFocus()
			return m, nil

		case "enter":
			if m.focus == focusNodes {
				if idx := m.nodesTable.Cursor(); idx >= 0 && idx < len(m.nodes) {
					n := m.nodes[idx]
					m.selected = nodeKey{Cluster: n.Cluster, Node: n.Node}
					m.nodesTable.SetRows(nodeRows(m.nodes, m.selected, m.multiCluster()))
					m.guestsTable.SetRows(guestRows(m.guests, m.selected, m.multiCluster()))
					m.toggleFocus()
				}
			}
			return m, nil

		case "esc":
			if m.selected.Node != "" {
				m.selected = nodeKey{}
				m.nodesTable.SetRows(nodeRows(m.nodes, m.selected, m.multiCluster()))
				m.guestsTable.SetRows(guestRows(m.guests, m.selected, m.multiCluster()))
				if m.focus == focusGuests {
					m.toggleFocus()
				}
			}
			return m, nil
		}

		var cmd tea.Cmd
		m.nodesTable, cmd = m.nodesTable.Update(msg)
		var cmd2 tea.Cmd
		m.guestsTable, cmd2 = m.guestsTable.Update(msg)
		return m, tea.Batch(cmd, cmd2)

	case nodesMsg:
		m.nodes = msg
		m.err = nil
		m.loading = false
		m.nodesTable.SetRows(nodeRows(m.nodes, m.selected, m.multiCluster()))
		m.resizeTables()
		return m, m.fetchGuests()

	case guestsMsg:
		m.guests = msg
		m.guestsTable.SetRows(guestRows(m.guests, m.selected, m.multiCluster()))
		m.resizeTables()
		return m, tick()

	case errMsg:
		m.err = msg.err
		m.loading = false
		return m, tick()

	case tickMsg:
		return m, m.fetchNodes()
	}

	return m, nil
}

func (m *Model) toggleFocus() {
	if m.focus == focusNodes {
		m.focus = focusGuests
		m.nodesTable.Blur()
		m.guestsTable.Focus()
	} else {
		m.focus = focusNodes
		m.guestsTable.Blur()
		m.nodesTable.Focus()
	}
}

func (m *Model) resizeTables() {
	available := m.height - 10 // title, help, section labels, spacing
	if available < 6 {
		available = 6
	}

	nodesHeight := clamp(len(m.nodes), 3, available/3)
	guestsHeight := clamp(len(m.guests), 3, available-nodesHeight)

	m.nodesTable.SetHeight(nodesHeight)
	m.guestsTable.SetHeight(guestsHeight)
}

func clamp(v, lo, hi int) int {
	if hi < lo {
		hi = lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (m Model) View() string {
	title := titleStyle.Render("lazypve")
	help := helpStyle.Render(m.helpText())

	if m.loading {
		return title + "\n\nconnecting to Proxmox...\n\n" + help
	}
	if m.err != nil {
		return title + "\n\n" + errStyle.Render("error: "+m.err.Error()) + "\n\n" + help
	}

	nodesLabel := "Nodes"
	shown := len(m.guestsTable.Rows())
	guestsLabel := fmt.Sprintf("Guests — %d", shown)
	if m.selected.Node != "" {
		if m.multiCluster() {
			guestsLabel = fmt.Sprintf("Guests — %d of %d, filtered to node %q on cluster %q", shown, len(m.guests), m.selected.Node, m.selected.Cluster)
		} else {
			guestsLabel = fmt.Sprintf("Guests — %d of %d, filtered to node %q", shown, len(m.guests), m.selected.Node)
		}
	}
	if m.focus == focusNodes {
		nodesLabel = activeLabelStyle.Render(nodesLabel)
		guestsLabel = inactiveLabelStyle.Render(guestsLabel)
	} else {
		nodesLabel = inactiveLabelStyle.Render(nodesLabel)
		guestsLabel = activeLabelStyle.Render(guestsLabel)
	}

	body := nodesLabel + "\n" + m.nodesTable.View() + "\n\n" + guestsLabel + "\n" + m.guestsTable.View()

	return title + "\n\n" + body + "\n\n" + help
}

// helpText only lists actions that would actually do something right now:
// "enter" only applies while the nodes table is focused, "esc" only applies
// while a node filter is active.
func (m Model) helpText() string {
	parts := []string{"tab: switch"}
	if m.focus == focusNodes {
		parts = append(parts, "enter: filter by node")
	}
	if m.selected.Node != "" {
		parts = append(parts, "esc: show all guests")
	}
	parts = append(parts, "q: quit")
	return strings.Join(parts, "  ")
}

func nodeColumns(multiCluster bool) []table.Column {
	cols := []table.Column{}
	if multiCluster {
		cols = append(cols, table.Column{Title: "CLUSTER", Width: 12})
	}
	return append(cols,
		table.Column{Title: "NODE", Width: 16},
		table.Column{Title: "STATUS", Width: 9},
		table.Column{Title: "CPU%", Width: 5},
		table.Column{Title: "MEM", Width: 8},
		table.Column{Title: "MAXMEM", Width: 8},
		table.Column{Title: "UPTIME", Width: 9},
	)
}

// nodeRows marks the currently filtered node (if any) with a leading arrow,
// so the drill-down filter is visible in the nodes table too, not just the
// guests section label.
func nodeRows(nodes []node, selected nodeKey, multiCluster bool) []table.Row {
	rows := make([]table.Row, 0, len(nodes))
	for _, n := range nodes {
		name := "  " + n.Node
		if n.Cluster == selected.Cluster && n.Node == selected.Node {
			name = "▶ " + n.Node
		}
		row := table.Row{}
		if multiCluster {
			row = append(row, n.Cluster)
		}
		row = append(row,
			name,
			n.Status,
			fmt.Sprintf("%.1f%%", n.CPU*100),
			formatBytes(n.Mem),
			formatBytes(n.MaxMem),
			formatUptime(n.Uptime),
		)
		rows = append(rows, row)
	}
	return rows
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
