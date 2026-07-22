package ui

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"

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

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("245"))

	statusOnlineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	statusOtherStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

type nodesMsg []pve.Node
type errMsg struct{ err error }
type tickMsg time.Time

type Model struct {
	client *pve.Client

	width, height int
	nodes         []pve.Node
	guests        []guest
	err           error
	loading       bool
}

func New(client *pve.Client) Model {
	return Model{client: client, loading: true}
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

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case nodesMsg:
		m.nodes = msg
		m.err = nil
		m.loading = false
		return m, m.fetchGuests()

	case guestsMsg:
		m.guests = msg
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

func (m Model) View() string {
	title := titleStyle.Render("lazypve")
	help := helpStyle.Render("q: quit")

	var body string
	switch {
	case m.loading:
		body = "connecting to Proxmox..."
	case m.err != nil:
		body = errStyle.Render("error: " + m.err.Error())
	default:
		body = m.renderNodes() + "\n\n" + m.renderGuests()
	}

	return title + "\n\n" + body + "\n\n" + help
}

func (m Model) renderNodes() string {
	if len(m.nodes) == 0 {
		return "no nodes found"
	}

	header := headerStyle.Render(fmt.Sprintf("%-16s %-10s %6s %8s %8s %10s", "NODE", "STATUS", "CPU%", "MEM", "MAXMEM", "UPTIME"))
	lines := []string{header}

	for _, n := range m.nodes {
		statusStyle := statusOtherStyle
		if n.Status == "online" {
			statusStyle = statusOnlineStyle
		}
		line := fmt.Sprintf("%-16s %s %5.1f%% %8s %8s %10s",
			n.Node,
			statusStyle.Render(fmt.Sprintf("%-10s", n.Status)),
			n.CPU*100,
			formatBytes(n.Mem),
			formatBytes(n.MaxMem),
			formatUptime(n.Uptime),
		)
		lines = append(lines, line)
	}

	out := lines[0]
	for _, l := range lines[1:] {
		out += "\n" + l
	}
	return out
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
