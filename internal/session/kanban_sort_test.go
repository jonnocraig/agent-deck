package session

import "testing"

func TestInitialSortOrder(t *testing.T) {
	// First card in each column gets column_base
	tests := []struct {
		column KanbanColumn
		want   int
	}{
		{KanbanBacklog, 0},
		{KanbanDesign, 1000},
		{KanbanPlan, 2000},
		{KanbanImplement, 3000},
		{KanbanReview, 4000},
		{KanbanDone, 5000},
	}
	for _, tt := range tests {
		t.Run(string(tt.column), func(t *testing.T) {
			got := CalculateInitialSortOrder(tt.column, nil)
			if got != tt.want {
				t.Errorf("CalculateInitialSortOrder(%s, nil) = %d, want %d", tt.column, got, tt.want)
			}
		})
	}
}

func TestAppendSortOrder(t *testing.T) {
	// Appending cards gets column_base + position * 100
	existing := []*Instance{
		{KanbanSortOrder: 0},
		{KanbanSortOrder: 100},
	}
	got := CalculateInitialSortOrder(KanbanBacklog, existing)
	if got != 200 {
		t.Errorf("expected 200 (3rd card in backlog), got %d", got)
	}
}

func TestInsertBetween(t *testing.T) {
	got := CalculateInsertBetween(100, 200)
	if got != 150 {
		t.Errorf("CalculateInsertBetween(100, 200) = %d, want 150", got)
	}
}

func TestInsertBetween_NarrowGap(t *testing.T) {
	// When gap is exactly 1, result should still be between (integer division)
	got := CalculateInsertBetween(100, 101)
	// (100 + 101) / 2 = 100 — same as above, which means rebalance needed
	if got != 100 {
		t.Errorf("CalculateInsertBetween(100, 101) = %d, want 100", got)
	}
}

func TestNeedsRebalance(t *testing.T) {
	tests := []struct {
		name       string
		sortOrders []int
		want       bool
	}{
		{"well_spaced", []int{0, 100, 200, 300}, false},
		{"narrow_gap", []int{0, 100, 101, 200}, true},
		{"single", []int{100}, false},
		{"empty", []int{}, false},
		{"gap_exactly_10", []int{0, 10, 20}, false},
		{"gap_below_10", []int{0, 9, 20}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NeedsRebalance(tt.sortOrders)
			if got != tt.want {
				t.Errorf("NeedsRebalance(%v) = %v, want %v", tt.sortOrders, got, tt.want)
			}
		})
	}
}

func TestRebalance(t *testing.T) {
	// Input: 5 cards with narrow gaps
	input := []*Instance{
		{ID: "a", KanbanSortOrder: 100},
		{ID: "b", KanbanSortOrder: 101},
		{ID: "c", KanbanSortOrder: 102},
		{ID: "d", KanbanSortOrder: 103},
		{ID: "e", KanbanSortOrder: 104},
	}
	result := Rebalance(input, KanbanBacklog)

	// Should be rebalanced with 100-spacing
	expected := map[string]int{
		"a": 0,
		"b": 100,
		"c": 200,
		"d": 300,
		"e": 400,
	}

	if len(result) != len(expected) {
		t.Fatalf("expected %d results, got %d", len(expected), len(result))
	}

	for id, expectedOrder := range expected {
		if result[id] != expectedOrder {
			t.Errorf("result[%s] = %d, want %d", id, result[id], expectedOrder)
		}
	}

	// Original should be unchanged (immutability)
	if input[0].KanbanSortOrder != 100 {
		t.Errorf("original mutated: input[0].KanbanSortOrder = %d, want 100", input[0].KanbanSortOrder)
	}
}

func TestColumnMoveSortOrder(t *testing.T) {
	// Moving to Design column with 2 existing cards
	existing := []*Instance{
		{KanbanSortOrder: 1000},
		{KanbanSortOrder: 1100},
	}
	got := CalculateMoveSortOrder(KanbanDesign, existing)
	if got != 1200 {
		t.Errorf("expected 1200 (append to design), got %d", got)
	}
}

func TestEmptyColumnSortOrder(t *testing.T) {
	got := CalculateMoveSortOrder(KanbanDesign, nil)
	if got != 1000 {
		t.Errorf("expected 1000 (first in design), got %d", got)
	}
}
