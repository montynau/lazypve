package ui

import "slices"

// nodeSortFields lists the node comparators in the same order (and with the
// same conditional CLUSTER column) as nodeColumns, so a column's position in
// the rendered header matches its position here.
func nodeSortFields(multiCluster bool) []func(a, b node) bool {
	fields := []func(a, b node) bool{}
	if multiCluster {
		fields = append(fields, func(a, b node) bool { return a.Cluster < b.Cluster })
	}
	return append(fields,
		func(a, b node) bool { return a.Node < b.Node },
		func(a, b node) bool { return a.Status < b.Status },
		func(a, b node) bool { return a.CPU < b.CPU },
		func(a, b node) bool { return a.Mem < b.Mem },
		func(a, b node) bool { return a.MaxMem < b.MaxMem },
		func(a, b node) bool { return a.Uptime < b.Uptime },
	)
}

// guestSortFields mirrors guestColumns the same way nodeSortFields mirrors
// nodeColumns.
func guestSortFields(multiCluster bool) []func(a, b guest) bool {
	fields := []func(a, b guest) bool{}
	if multiCluster {
		fields = append(fields, func(a, b guest) bool { return a.Cluster < b.Cluster })
	}
	return append(fields,
		func(a, b guest) bool { return a.Node < b.Node },
		func(a, b guest) bool { return a.Kind < b.Kind },
		func(a, b guest) bool { return a.VMID < b.VMID },
		func(a, b guest) bool { return a.Name < b.Name },
		func(a, b guest) bool { return a.Status < b.Status },
		func(a, b guest) bool { return a.CPU < b.CPU },
		func(a, b guest) bool { return a.Mem < b.Mem },
		func(a, b guest) bool { return a.Disk < b.Disk },
		func(a, b guest) bool { return a.NetInRate < b.NetInRate },
		func(a, b guest) bool { return a.NetOutRate < b.NetOutRate },
		func(a, b guest) bool { return a.Uptime < b.Uptime },
	)
}

// nextSortState advances a linear cursor over (column, direction) pairs: each
// column is visited ascending then descending before moving to the next, so
// repeated presses of the same key ("]" or "[") predictably toggle direction
// before advancing further, matching a spreadsheet's "click header again to
// reverse" convention. -1 means "no explicit sort" — the table's natural
// fetch order (already sorted by cluster/node).
func nextSortState(state, dir, numFields int) int {
	total := numFields * 2
	if state < 0 {
		if dir > 0 {
			return 0
		}
		return total - 1
	}
	return (state + dir + total) % total
}

// sortedNodes returns a sorted copy of nodes per state, leaving the input
// untouched — nodes always stays in its natural fetch order so state -1
// ("no explicit sort") has an original order to fall back to.
func sortedNodes(nodes []node, state int, multiCluster bool) []node {
	if state < 0 {
		return nodes
	}
	sorted := slices.Clone(nodes)
	fields := nodeSortFields(multiCluster)
	col, asc := state/2, state%2 == 0
	less := fields[col]
	slices.SortStableFunc(sorted, func(a, b node) int {
		switch {
		case less(a, b):
			return -1
		case less(b, a):
			return 1
		default:
			return 0
		}
	})
	if !asc {
		slices.Reverse(sorted)
	}
	return sorted
}

// sortedGuests mirrors sortedNodes for the guests table.
func sortedGuests(guests []guest, state int, multiCluster bool) []guest {
	if state < 0 {
		return guests
	}
	sorted := slices.Clone(guests)
	fields := guestSortFields(multiCluster)
	col, asc := state/2, state%2 == 0
	less := fields[col]
	slices.SortStableFunc(sorted, func(a, b guest) int {
		switch {
		case less(a, b):
			return -1
		case less(b, a):
			return 1
		default:
			return 0
		}
	})
	if !asc {
		slices.Reverse(sorted)
	}
	return sorted
}

// sortIndicator returns an arrow marking colIdx as the active sort column
// (▲ ascending, ▼ descending), or "" if it isn't the active one.
func sortIndicator(state, colIdx int) string {
	if state < 0 || state/2 != colIdx {
		return ""
	}
	if state%2 == 0 {
		return "▲"
	}
	return "▼"
}
