package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

// bubbles/table's own View() measures cell width with go-runewidth, which
// isn't ANSI-aware and miscounts escape codes as visible characters — so any
// pre-colored cell text corrupts column alignment. We render the tables
// ourselves with lipgloss instead (which is ANSI-aware) and keep table.Model
// around only for cursor/focus state and row-count clamping.
var (
	borderActiveColor   = lipgloss.Color("205")
	borderInactiveColor = lipgloss.Color("240")
	statusRunningColor  = lipgloss.Color("42")
	statusStoppedColor  = lipgloss.Color("244")
	headerCellStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("246"))
	filterMarkerStyle   = lipgloss.NewStyle().Foreground(borderActiveColor)
	cursorRowStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
)

// padCell pads plain (non-ANSI) text to a fixed visible width, truncating
// with an ellipsis if it overflows.
func padCell(s string, width int) string {
	r := []rune(s)
	if len(r) > width {
		if width <= 1 {
			return strings.Repeat(".", width)
		}
		return string(r[:width-1]) + "…"
	}
	return s + strings.Repeat(" ", width-len(r))
}

// padVisible pads a possibly ANSI-colored string to a fixed *visible* width,
// using lipgloss.Width so escape codes aren't counted as characters.
func padVisible(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

// statusText renders a status as a colored dot plus label, padded against
// its *unstyled* length so the embedded ANSI color codes never get counted
// as visible characters.
func statusText(status string, width int) string {
	color := statusStoppedColor
	if status == "running" || status == "online" {
		color = statusRunningColor
	}
	plain := "● " + status
	pad := width - len([]rune(plain))
	if pad < 0 {
		pad = 0
	}
	dot := lipgloss.NewStyle().Foreground(color).Render("●")
	return dot + " " + status + strings.Repeat(" ", pad)
}

// visibleWindow returns the [start,end) row bounds to display `height` rows
// out of `total`, keeping `cursor` in view and centered when possible. This
// replicates the windowing bubbles/table's own View() used to do for us.
func visibleWindow(cursor, total, height int) (start, end int) {
	if total <= height {
		return 0, total
	}
	start = cursor - height/2
	if start < 0 {
		start = 0
	}
	end = start + height
	if end > total {
		end = total
		start = end - height
	}
	return start, end
}

// renderHeaderRow marks the active sort column (if any) with ▲/▼, per
// nextSortState's linear (column, direction) cycle.
func renderHeaderRow(cols []table.Column, sortState int) string {
	parts := make([]string, len(cols))
	for i, c := range cols {
		title := c.Title
		if arrow := sortIndicator(sortState, i); arrow != "" {
			title += arrow
		}
		parts[i] = headerCellStyle.Render(padCell(title, c.Width))
	}
	return strings.Join(parts, " ")
}

// renderPanel draws a rounded border box around a header + row lines, with
// the title embedded in the top border and the border colored by focus —
// the single indicator for which panel is active.
func renderPanel(title string, active bool, header string, rows []string) string {
	borderColor := borderInactiveColor
	if active {
		borderColor = borderActiveColor
	}
	b := lipgloss.NewStyle().Foreground(borderColor)

	interior := lipgloss.Width(header)
	for _, r := range rows {
		if w := lipgloss.Width(r); w > interior {
			interior = w
		}
	}
	if min := len([]rune(title)) + 2; interior < min {
		interior = min
	}

	dashes := interior - len([]rune(title)) - 1
	var out strings.Builder
	out.WriteString(b.Render("╭─ "+title+" "+strings.Repeat("─", dashes)+"╮") + "\n")
	out.WriteString(b.Render("│") + " " + padVisible(header, interior) + " " + b.Render("│") + "\n")
	for _, r := range rows {
		out.WriteString(b.Render("│") + " " + padVisible(r, interior) + " " + b.Render("│") + "\n")
	}
	out.WriteString(b.Render("╰" + strings.Repeat("─", interior+2) + "╯"))
	return out.String()
}

func renderNodesPanel(m Model) string {
	cols := nodeColumns(m.multiCluster())
	header := renderHeaderRow(cols, m.nodesSort)

	nodes := m.displayNodes()
	active := m.focus == focusNodes
	cursor := m.nodesTable.Cursor()
	start, end := visibleWindow(cursor, len(nodes), m.nodesTable.Height())

	rows := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		rows = append(rows, renderNodeRow(nodes[i], cols, m.selected, m.multiCluster(), active && i == cursor))
	}

	return renderPanel("Nodes", active, header, rows)
}

// renderNodeRow marks the currently filtered node (if any) with a leading
// arrow, so the drill-down filter is visible in the nodes table too, not
// just the guests panel title. On the cursor row, both the arrow and the
// status text drop their own color and inherit the row's single highlight
// style instead — nesting a colored span inside another style's Render call
// lets the inner span's ANSI reset clobber the outer color for everything
// after it, so a row is either "selected" or "has colored cells", never both.
func renderNodeRow(n node, cols []table.Column, selected nodeKey, multiCluster bool, highlighted bool) string {
	filtered := n.Cluster == selected.Cluster && n.Node == selected.Node

	cells := make([]string, 0, len(cols))
	idx := 0
	if multiCluster {
		cells = append(cells, padCell(n.Cluster, cols[idx].Width))
		idx++
	}

	marker := "  "
	if filtered {
		if highlighted {
			marker = "▶ "
		} else {
			marker = filterMarkerStyle.Render("▶") + " "
		}
	}
	cells = append(cells, marker+padCell(n.Node, cols[idx].Width-2))
	idx++

	if highlighted {
		cells = append(cells, padCell(n.Status, cols[idx].Width))
	} else {
		cells = append(cells, statusText(n.Status, cols[idx].Width))
	}
	idx++

	cells = append(cells,
		padCell(fmt.Sprintf("%.1f%%", n.CPU*100), cols[idx].Width),
		padCell(formatBytes(n.Mem), cols[idx+1].Width),
		padCell(formatBytes(n.MaxMem), cols[idx+2].Width),
		padCell(formatUptime(n.Uptime), cols[idx+3].Width),
	)

	row := strings.Join(cells, " ")
	if highlighted {
		return cursorRowStyle.Render(row)
	}
	return row
}

func renderGuestsPanel(m Model) string {
	cols := guestColumns(m.multiCluster())
	header := renderHeaderRow(cols, m.guestsSort)

	active := m.focus == focusGuests
	cursor := m.guestsTable.Cursor()
	shown := len(m.guestsTable.Rows())
	start, end := visibleWindow(cursor, shown, m.guestsTable.Height())

	filtered := filterGuests(m.displayGuests(), m.selected)
	rows := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		rows = append(rows, renderGuestRow(filtered[i], cols, m.multiCluster(), active && i == cursor))
	}

	return renderPanel(guestsTitle(shown, len(m.guests), m.selected, m.multiCluster()), active, header, rows)
}

func guestsTitle(shown, total int, selected nodeKey, multiCluster bool) string {
	if selected.Node == "" {
		return fmt.Sprintf("Guests — %d", shown)
	}
	if multiCluster {
		return fmt.Sprintf("Guests — %d of %d, filtered to node %q on cluster %q", shown, total, selected.Node, selected.Cluster)
	}
	return fmt.Sprintf("Guests — %d of %d, filtered to node %q", shown, total, selected.Node)
}

func renderGuestRow(g guest, cols []table.Column, multiCluster bool, highlighted bool) string {
	cells := make([]string, 0, len(cols))
	idx := 0
	if multiCluster {
		cells = append(cells, padCell(g.Cluster, cols[idx].Width))
		idx++
	}

	cells = append(cells,
		padCell(g.Node, cols[idx].Width),
		padCell(g.Kind, cols[idx+1].Width),
		padCell(fmt.Sprintf("%d", g.VMID), cols[idx+2].Width),
		padCell(g.Name, cols[idx+3].Width),
	)
	idx += 4

	if highlighted {
		cells = append(cells, padCell(g.Status, cols[idx].Width))
	} else {
		cells = append(cells, statusText(g.Status, cols[idx].Width))
	}
	idx++

	cells = append(cells,
		padCell(fmt.Sprintf("%.1f%%", g.CPU*100), cols[idx].Width),
		padCell(formatBytes(g.Mem), cols[idx+1].Width),
		padCell(formatBytes(g.Disk), cols[idx+2].Width),
		padCell(formatRate(g.NetInRate), cols[idx+3].Width),
		padCell(formatRate(g.NetOutRate), cols[idx+4].Width),
		padCell(formatUptime(g.Uptime), cols[idx+5].Width),
	)

	row := strings.Join(cells, " ")
	if highlighted {
		return cursorRowStyle.Render(row)
	}
	return row
}
