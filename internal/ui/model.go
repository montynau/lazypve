package ui

import (
	"cmp"
	"context"
	"fmt"
	"slices"
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

type nodesMsg []pve.Node
type errMsg struct{ err error }
type tickMsg time.Time

type Model struct {
	client *pve.Client

	width, height int

	nodes        []pve.Node
	guests       []guest
	selectedNode string // "" = no filter, guests table shows every guest

	nodesTable  table.Model
	guestsTable table.Model
	focus       focusArea

	err     error
	loading bool
}

func New(client *pve.Client) Model {
	nodesTable := table.New(
		table.WithColumns(nodeColumns()),
		table.WithFocused(true),
		table.WithHeight(6),
	)
	guestsTable := table.New(
		table.WithColumns(guestColumns()),
		table.WithHeight(10),
	)

	return Model{
		client:      client,
		loading:     true,
		nodesTable:  nodesTable,
		guestsTable: guestsTable,
		focus:       focusNodes,
	}
}

func (m Model) Init() tea.Cmd {
	return m.fetchNodes()
}

func (m Model) fetchNodes() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		nodes, err := m.client.GetNodes(ctx)
		if err != nil {
			return errMsg{err}
		}
		slices.SortFunc(nodes, func(a, b pve.Node) int {
			return cmp.Compare(a.Node, b.Node)
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
					m.selectedNode = m.nodes[idx].Node
					m.nodesTable.SetRows(nodeRows(m.nodes, m.selectedNode))
					m.guestsTable.SetRows(guestRows(m.guests, m.selectedNode))
					m.toggleFocus()
				}
			}
			return m, nil

		case "esc":
			if m.selectedNode != "" {
				m.selectedNode = ""
				m.nodesTable.SetRows(nodeRows(m.nodes, m.selectedNode))
				m.guestsTable.SetRows(guestRows(m.guests, m.selectedNode))
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
		m.nodesTable.SetRows(nodeRows(m.nodes, m.selectedNode))
		m.resizeTables()
		return m, m.fetchGuests()

	case guestsMsg:
		m.guests = msg
		m.guestsTable.SetRows(guestRows(m.guests, m.selectedNode))
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
	help := helpStyle.Render("tab: switch  enter: filter by node  esc: show all guests  q: quit")

	if m.loading {
		return title + "\n\nconnecting to Proxmox...\n\n" + help
	}
	if m.err != nil {
		return title + "\n\n" + errStyle.Render("error: "+m.err.Error()) + "\n\n" + help
	}

	nodesLabel := "Nodes"
	shown := len(m.guestsTable.Rows())
	guestsLabel := fmt.Sprintf("Guests — %d", shown)
	if m.selectedNode != "" {
		guestsLabel = fmt.Sprintf("Guests — %d of %d, filtered to node %q", shown, len(m.guests), m.selectedNode)
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

func nodeColumns() []table.Column {
	return []table.Column{
		{Title: "NODE", Width: 16},
		{Title: "STATUS", Width: 9},
		{Title: "CPU%", Width: 5},
		{Title: "MEM", Width: 8},
		{Title: "MAXMEM", Width: 8},
		{Title: "UPTIME", Width: 9},
	}
}

// nodeRows marks the currently filtered node (if any) with a leading arrow,
// so the drill-down filter is visible in the nodes table too, not just the
// guests section label.
func nodeRows(nodes []pve.Node, selectedNode string) []table.Row {
	rows := make([]table.Row, 0, len(nodes))
	for _, n := range nodes {
		name := "  " + n.Node
		if n.Node == selectedNode {
			name = "▶ " + n.Node
		}
		rows = append(rows, table.Row{
			name,
			n.Status,
			fmt.Sprintf("%.1f%%", n.CPU*100),
			formatBytes(n.Mem),
			formatBytes(n.MaxMem),
			formatUptime(n.Uptime),
		})
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
