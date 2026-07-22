package ui

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// guest is a unified view over pve.VM and pve.Container, since the node
// table displays them the same way and only needs to know Kind to tell
// them apart.
type guest struct {
	Node   string
	VMID   int
	Name   string
	Kind   string // "qemu" or "lxc"
	Status string
	CPU    float64
	Mem    int64
	MaxMem int64
	Uptime int64
}

type guestsMsg []guest

func (m Model) fetchGuests() tea.Cmd {
	nodes := m.nodes
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var guests []guest
		for _, n := range nodes {
			vms, err := m.client.GetVMs(ctx, n.Node)
			if err == nil {
				for _, vm := range vms {
					guests = append(guests, guest{
						Node: n.Node, VMID: vm.VMID, Name: vm.Name, Kind: "qemu",
						Status: vm.Status, CPU: vm.CPU, Mem: vm.Mem, MaxMem: vm.MaxMem, Uptime: vm.Uptime,
					})
				}
			}

			containers, err := m.client.GetContainers(ctx, n.Node)
			if err == nil {
				for _, c := range containers {
					guests = append(guests, guest{
						Node: n.Node, VMID: c.VMID, Name: c.Name, Kind: "lxc",
						Status: c.Status, CPU: c.CPU, Mem: c.Mem, MaxMem: c.MaxMem, Uptime: c.Uptime,
					})
				}
			}
		}

		// Proxmox doesn't guarantee a stable order between requests, so sort
		// here to keep rows from jumping around on every refresh.
		slices.SortFunc(guests, func(a, b guest) int {
			return cmp.Or(
				cmp.Compare(a.Node, b.Node),
				cmp.Compare(a.Kind, b.Kind),
				cmp.Compare(a.VMID, b.VMID),
			)
		})

		return guestsMsg(guests)
	}
}

func (m Model) renderGuests() string {
	if len(m.guests) == 0 {
		return "no VMs or containers found"
	}

	header := headerStyle.Render(fmt.Sprintf("%-16s %-6s %-6s %-16s %-10s %6s %8s %10s",
		"NODE", "TYPE", "VMID", "NAME", "STATUS", "CPU%", "MEM", "UPTIME"))
	lines := []string{header}

	for _, g := range m.guests {
		statusStyle := statusOtherStyle
		if g.Status == "running" {
			statusStyle = statusOnlineStyle
		}
		line := fmt.Sprintf("%-16s %-6s %-6d %-16s %s %5.1f%% %8s %10s",
			g.Node,
			g.Kind,
			g.VMID,
			g.Name,
			statusStyle.Render(fmt.Sprintf("%-10s", g.Status)),
			g.CPU*100,
			formatBytes(g.Mem),
			formatUptime(g.Uptime),
		)
		lines = append(lines, line)
	}

	out := lines[0]
	for _, l := range lines[1:] {
		out += "\n" + l
	}
	return out
}
