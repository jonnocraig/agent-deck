package session

import "sort"

const (
	// columnBase is the multiplier for column index in sort order
	columnBase = 1000
	// positionStep is the gap between consecutive cards
	positionStep = 100
	// minGap is the minimum gap before rebalancing is triggered
	minGap = 10
)

// CalculateInitialSortOrder returns the sort order for a new card appended to a column.
// If existingCards is nil or empty, returns the column base (column_index * 1000).
// Otherwise returns max_existing + 100.
func CalculateInitialSortOrder(column KanbanColumn, existingCards []*Instance) int {
	base := column.Index() * columnBase
	if len(existingCards) == 0 {
		return base
	}
	maxOrder := base
	for _, card := range existingCards {
		if card.KanbanSortOrder > maxOrder {
			maxOrder = card.KanbanSortOrder
		}
	}
	return maxOrder + positionStep
}

// CalculateInsertBetween returns a sort order between two existing values.
// Uses integer midpoint: (above + below) / 2.
func CalculateInsertBetween(above, below int) int {
	return (above + below) / 2
}

// NeedsRebalance returns true if any adjacent sort orders have a gap less than minGap.
func NeedsRebalance(sortOrders []int) bool {
	if len(sortOrders) < 2 {
		return false
	}
	sorted := make([]int, len(sortOrders))
	copy(sorted, sortOrders)
	sort.Ints(sorted)
	for i := 1; i < len(sorted); i++ {
		if sorted[i]-sorted[i-1] < minGap {
			return true
		}
	}
	return false
}

// Rebalance calculates new sort orders for cards in a column.
// Returns a map of instance ID -> new sort order (immutable — does not modify input).
// Sort orders are recalculated as: column_base + position * positionStep.
// Use BatchUpdateSortOrders to persist the changes.
func Rebalance(cards []*Instance, column KanbanColumn) map[string]int {
	base := column.Index() * columnBase

	// Sort by current order first
	sorted := make([]*Instance, len(cards))
	copy(sorted, cards)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].KanbanSortOrder < sorted[j].KanbanSortOrder
	})

	// Calculate new sort orders
	result := make(map[string]int, len(sorted))
	for i, card := range sorted {
		result[card.ID] = base + i*positionStep
	}
	return result
}

// CalculateMoveSortOrder returns the sort order for a card being moved to a target column.
// Appends after the last card in the target column.
func CalculateMoveSortOrder(targetColumn KanbanColumn, existingCards []*Instance) int {
	return CalculateInitialSortOrder(targetColumn, existingCards)
}
