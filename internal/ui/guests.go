package ui

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

// guest is a unified view over pve.VM and pve.Container, since the guest
// table displays them the same way and only needs Kind to tell them apart.
type guest struct {
	Cluster string
	Node    string
	VMID    int
	Name    string
	Kind    string // "qemu" or "lxc"
	Status  string
	CPU     float64
	Mem     int64
	MaxMem  int64
	Disk    int64
	MaxDisk int64
	NetIn   int64
	NetOut  int64
	Uptime  int64
}

type guestsMsg []guest

func (m Model) fetchGuests() tea.Cmd {
	nodes := m.nodes
	clients := m.clients
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var guests []guest
		for _, n := range nodes {
			client, ok := clients[n.Cluster]
			if !ok {
				continue
			}

			vms, err := client.GetVMs(ctx, n.Node)
			if err == nil {
				for _, vm := range vms {
					guests = append(guests, guest{
						Cluster: n.Cluster, Node: n.Node, VMID: vm.VMID, Name: vm.Name, Kind: "qemu",
						Status: vm.Status, CPU: vm.CPU, Mem: vm.Mem, MaxMem: vm.MaxMem,
						Disk: vm.Disk, MaxDisk: vm.MaxDisk, NetIn: vm.NetIn, NetOut: vm.NetOut,
						Uptime: vm.Uptime,
					})
				}
			}

			containers, err := client.GetContainers(ctx, n.Node)
			if err == nil {
				for _, c := range containers {
					guests = append(guests, guest{
						Cluster: n.Cluster, Node: n.Node, VMID: c.VMID, Name: c.Name, Kind: "lxc",
						Status: c.Status, CPU: c.CPU, Mem: c.Mem, MaxMem: c.MaxMem,
						Disk: c.Disk, MaxDisk: c.MaxDisk, NetIn: c.NetIn, NetOut: c.NetOut,
						Uptime: c.Uptime,
					})
				}
			}
		}

		// Proxmox doesn't guarantee a stable order between requests, so sort
		// here to keep rows from jumping around on every refresh.
		slices.SortFunc(guests, func(a, b guest) int {
			return cmp.Or(
				cmp.Compare(a.Cluster, b.Cluster),
				cmp.Compare(a.Node, b.Node),
				cmp.Compare(a.Kind, b.Kind),
				cmp.Compare(a.VMID, b.VMID),
			)
		})

		return guestsMsg(guests)
	}
}

func guestColumns(multiCluster bool) []table.Column {
	cols := []table.Column{}
	if multiCluster {
		cols = append(cols, table.Column{Title: "CLUSTER", Width: 12})
	}
	return append(cols,
		table.Column{Title: "NODE", Width: 14},
		table.Column{Title: "TYPE", Width: 4},
		table.Column{Title: "VMID", Width: 5},
		table.Column{Title: "NAME", Width: 16},
		table.Column{Title: "STATUS", Width: 9},
		table.Column{Title: "CPU%", Width: 5},
		table.Column{Title: "MEM", Width: 8},
		table.Column{Title: "DISK", Width: 8},
		table.Column{Title: "NET IN", Width: 9},
		table.Column{Title: "NET OUT", Width: 9},
		table.Column{Title: "UPTIME", Width: 9},
	)
}

// guestRows builds table rows from guests, optionally restricted to a single
// node (drill-down). A zero filter means "show every guest".
func guestRows(guests []guest, filter nodeKey, multiCluster bool) []table.Row {
	rows := make([]table.Row, 0, len(guests))
	for _, g := range guests {
		if filter.Node != "" && (g.Cluster != filter.Cluster || g.Node != filter.Node) {
			continue
		}
		row := table.Row{}
		if multiCluster {
			row = append(row, g.Cluster)
		}
		row = append(row,
			g.Node,
			g.Kind,
			fmt.Sprintf("%d", g.VMID),
			g.Name,
			g.Status,
			fmt.Sprintf("%.1f%%", g.CPU*100),
			formatBytes(g.Mem),
			formatBytes(g.Disk),
			formatBytes(g.NetIn),
			formatBytes(g.NetOut),
			formatUptime(g.Uptime),
		)
		rows = append(rows, row)
	}
	return rows
}
