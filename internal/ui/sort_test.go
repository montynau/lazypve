package ui

import "testing"

func TestNextSortState(t *testing.T) {
	const numFields = 3 // states: 0=(col0,asc) 1=(col0,desc) 2=(col1,asc) 3=(col1,desc) 4=(col2,asc) 5=(col2,desc)

	// From unsorted, "]" lands on the first column ascending.
	if got := nextSortState(-1, 1, numFields); got != 0 {
		t.Fatalf("nextSortState(-1, +1) = %d, want 0", got)
	}
	// From unsorted, "[" wraps to the last column descending.
	if got := nextSortState(-1, -1, numFields); got != 5 {
		t.Fatalf("nextSortState(-1, -1) = %d, want 5", got)
	}

	// Repeated "]" on the same column toggles to descending before moving on.
	if got := nextSortState(0, 1, numFields); got != 1 {
		t.Fatalf("nextSortState(0, +1) = %d, want 1 (same column, desc)", got)
	}
	if got := nextSortState(1, 1, numFields); got != 2 {
		t.Fatalf("nextSortState(1, +1) = %d, want 2 (next column, asc)", got)
	}

	// Wraps around at the ends.
	if got := nextSortState(5, 1, numFields); got != 0 {
		t.Fatalf("nextSortState(5, +1) = %d, want 0 (wrap forward)", got)
	}
	if got := nextSortState(0, -1, numFields); got != 5 {
		t.Fatalf("nextSortState(0, -1) = %d, want 5 (wrap backward)", got)
	}
}

func TestSortedNodesAscDesc(t *testing.T) {
	original := []node{
		{Node: "c", CPU: 0.3},
		{Node: "a", CPU: 0.1},
		{Node: "b", CPU: 0.2},
	}

	// State 0 = column 0 (NODE, since single-cluster), ascending.
	asc := sortedNodes(original, 0, false)
	if got := []string{asc[0].Node, asc[1].Node, asc[2].Node}; got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Fatalf("expected nodes sorted ascending by name, got %v", got)
	}

	// State 1 = column 0 descending.
	desc := sortedNodes(original, 1, false)
	if got := []string{desc[0].Node, desc[1].Node, desc[2].Node}; got[0] != "c" || got[1] != "b" || got[2] != "a" {
		t.Fatalf("expected nodes sorted descending by name, got %v", got)
	}

	// The input itself must never be mutated — state -1 (and any future
	// caller relying on fetch order) depends on it staying untouched.
	if original[0].Node != "c" || original[1].Node != "a" || original[2].Node != "b" {
		t.Fatalf("expected original slice to be left untouched, got %+v", original)
	}

	// -1 returns the original order as-is.
	if unsorted := sortedNodes(original, -1, false); unsorted[0].Node != "c" {
		t.Fatalf("expected state -1 to return natural fetch order, got %v", unsorted)
	}
}
