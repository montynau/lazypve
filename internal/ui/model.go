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

	warnStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))
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

// nodesMsg carries the nodes that were fetched successfully plus, per
// cluster, the error from any cluster that failed this round — a cluster
// timing out shouldn't blank the whole UI when others are still reachable.
// clusterErrs only holds entries for clusters that failed; a cluster that
// recovers simply stops appearing in it on the next poll.
type nodesMsg struct {
	nodes       []node
	clusterErrs map[string]error
}

type errMsg struct{ err error }
type tickMsg time.Time

type Model struct {
	clients map[string]*pve.Client

	width, height int

	nodes    []node
	guests   []guest
	selected nodeKey // zero value = no drill-down filter

	// clusterErrors holds the error from any cluster that failed its most
	// recent node fetch, so a down cluster can be flagged in the UI instead
	// of just silently vanishing from the aggregated node/guest lists.
	clusterErrors map[string]error

	// prevGuestSample holds each guest's cumulative NetIn/NetOut from the
	// last poll, so the next poll can diff against it to show live
	// throughput instead of Proxmox's cumulative-since-boot counters.
	prevGuestSample map[guestSampleKey]guestSample

	// nodesSort/guestsSort are indices into a linear (column, direction)
	// cycle — see nextSortState. -1 means no explicit sort is active.
	nodesSort  int
	guestsSort int

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
		nodesSort:   -1,
		guestsSort:  -1,
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
		clusterErrs := map[string]error{}
		for name, client := range clients {
			raw, err := client.GetNodes(ctx)
			if err != nil {
				clusterErrs[name] = err
				continue
			}
			for _, n := range raw {
				nodes = append(nodes, node{
					Cluster: name, Node: n.Node, Status: n.Status,
					CPU: n.CPU, Mem: n.Mem, MaxMem: n.MaxMem, Uptime: n.Uptime,
				})
			}
		}

		if len(nodes) == 0 && len(clusterErrs) > 0 {
			errs := make([]error, 0, len(clusterErrs))
			for name, err := range clusterErrs {
				errs = append(errs, fmt.Errorf("%s: %w", name, err))
			}
			return errMsg{errors.Join(errs...)}
		}

		slices.SortFunc(nodes, func(a, b node) int {
			return cmp.Or(
				cmp.Compare(a.Cluster, b.Cluster),
				cmp.Compare(a.Node, b.Node),
			)
		})
		return nodesMsg{nodes: nodes, clusterErrs: clusterErrs}
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
				nodes := m.displayNodes()
				if idx := m.nodesTable.Cursor(); idx >= 0 && idx < len(nodes) {
					n := nodes[idx]
					m.selected = nodeKey{Cluster: n.Cluster, Node: n.Node}
					m.nodesTable.SetRows(nodeRows(nodes, m.selected, m.multiCluster()))
					m.guestsTable.SetRows(guestRows(m.displayGuests(), m.selected, m.multiCluster()))
					m.toggleFocus()
				}
			}
			return m, nil

		case "esc":
			focus := m.focus
			if m.selected.Node != "" {
				m.selected = nodeKey{}
				m.nodesTable.SetRows(nodeRows(m.displayNodes(), m.selected, m.multiCluster()))
				m.guestsTable.SetRows(guestRows(m.displayGuests(), m.selected, m.multiCluster()))
				if m.focus == focusGuests {
					m.toggleFocus()
				}
			}
			// Reset whichever table the user was looking at when they hit
			// esc, not m.focus — clearing the filter above may have already
			// flipped focus back to the nodes table.
			switch {
			case focus == focusNodes && m.nodesSort != -1:
				m.nodesSort = -1
				m.nodesTable.SetRows(nodeRows(m.displayNodes(), m.selected, m.multiCluster()))
			case focus == focusGuests && m.guestsSort != -1:
				m.guestsSort = -1
				m.guestsTable.SetRows(guestRows(m.displayGuests(), m.selected, m.multiCluster()))
			}
			return m, nil

		case "]":
			m.cycleSort(1)
			return m, nil

		case "[":
			m.cycleSort(-1)
			return m, nil
		}

		var cmd tea.Cmd
		m.nodesTable, cmd = m.nodesTable.Update(msg)
		var cmd2 tea.Cmd
		m.guestsTable, cmd2 = m.guestsTable.Update(msg)
		return m, tea.Batch(cmd, cmd2)

	case nodesMsg:
		m.nodes = msg.nodes
		m.clusterErrors = msg.clusterErrs
		m.err = nil
		m.loading = false
		m.nodesTable.SetRows(nodeRows(m.displayNodes(), m.selected, m.multiCluster()))
		m.resizeTables()
		return m, m.fetchGuests()

	case guestsMsg:
		m.guests = msg
		m.prevGuestSample = applyNetRates(m.guests, m.prevGuestSample, time.Now())
		m.guestsTable.SetRows(guestRows(m.displayGuests(), m.selected, m.multiCluster()))
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

// cycleSort advances the sort state of whichever table is focused and
// resorts+redisplays it immediately. Fresh data from the next poll is
// resorted the same way in Update's nodesMsg/guestsMsg handling, so the
// chosen order survives refreshes instead of reverting to fetch order.
func (m *Model) cycleSort(dir int) {
	if m.focus == focusNodes {
		m.nodesSort = nextSortState(m.nodesSort, dir, len(nodeSortFields(m.multiCluster())))
		m.nodesTable.SetRows(nodeRows(m.displayNodes(), m.selected, m.multiCluster()))
	} else {
		m.guestsSort = nextSortState(m.guestsSort, dir, len(guestSortFields(m.multiCluster())))
		m.guestsTable.SetRows(guestRows(m.displayGuests(), m.selected, m.multiCluster()))
	}
}

// displayNodes/displayGuests return m.nodes/m.guests sorted per the current
// sort state, without mutating the underlying slice — m.nodes/m.guests
// always stay in natural fetch order so clearing the sort (state -1) has an
// original order to fall back to instead of being stuck however it was last
// left.
func (m Model) displayNodes() []node {
	return sortedNodes(m.nodes, m.nodesSort, m.multiCluster())
}

func (m Model) displayGuests() []guest {
	return sortedGuests(m.guests, m.guestsSort, m.multiCluster())
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
	available := m.height - 12 // title, help, spacing, and each panel's border+header lines
	if len(m.clusterErrors) > 0 {
		available-- // room for the cluster-unreachable warning line
	}
	if available < 6 {
		available = 6
	}

	nodesHeight := clamp(len(m.nodes), 3, available/3)
	guestsHeight := clamp(len(m.guests), 3, available-nodesHeight)

	// table.Model.SetHeight(h) reserves one line for its own header internally
	// (viewport.Height = h - headerHeight), and Height() returns that already-
	// reduced value. We read Height() back in render.go as "how many data rows
	// to draw," so pad by 1 here to cancel that reservation out — otherwise
	// every panel silently renders one row short of what resizeTables intended.
	m.nodesTable.SetHeight(nodesHeight + 1)
	m.guestsTable.SetHeight(guestsHeight + 1)
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

	body := renderNodesPanel(m) + "\n\n" + renderGuestsPanel(m)

	if warn := m.clusterWarning(); warn != "" {
		return title + "\n" + warn + "\n\n" + body + "\n\n" + help
	}
	return title + "\n\n" + body + "\n\n" + help
}

// clusterWarning summarizes any cluster that failed its last node fetch, so
// a down cluster is visible instead of just silently missing from the
// aggregated node/guest lists. Empty when every configured cluster is
// reachable (which is also the only possible state for a single-cluster
// setup — see fetchNodes: an all-clusters failure is reported as m.err
// instead, since there'd be nothing left to show).
func (m Model) clusterWarning() string {
	if len(m.clusterErrors) == 0 {
		return ""
	}
	names := make([]string, 0, len(m.clusterErrors))
	for name := range m.clusterErrors {
		names = append(names, name)
	}
	slices.Sort(names)

	if len(names) == 1 {
		return warnStyle.Render(fmt.Sprintf("⚠ cluster %q unreachable", names[0]))
	}
	return warnStyle.Render(fmt.Sprintf("⚠ %d clusters unreachable: %s", len(names), strings.Join(names, ", ")))
}

// helpText only lists actions that would actually do something right now:
// "enter" only applies while the nodes table is focused, "esc" only applies
// while a node filter and/or the focused table's sort is active.
func (m Model) helpText() string {
	parts := []string{"tab: switch", "[ ]: sort"}
	if m.focus == focusNodes {
		parts = append(parts, "enter: filter by node")
	}

	filterActive := m.selected.Node != ""
	sortActive := (m.focus == focusNodes && m.nodesSort != -1) || (m.focus == focusGuests && m.guestsSort != -1)
	switch {
	case filterActive && sortActive:
		parts = append(parts, "esc: reset filter & sort")
	case filterActive:
		parts = append(parts, "esc: show all guests")
	case sortActive:
		parts = append(parts, "esc: clear sort")
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
