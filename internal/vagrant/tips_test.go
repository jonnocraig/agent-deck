package vagrant

import (
	"testing"
)

func TestTipCount(t *testing.T) {
	if len(tips) != 100 {
		t.Errorf("Expected 100 tips, got %d", len(tips))
	}
}

func TestTipCategories(t *testing.T) {
	vagrantCount := 0
	worldCount := 0

	for _, tip := range tips {
		if tip.Category == "vagrant" {
			vagrantCount++
		} else if tip.Category == "world" {
			worldCount++
		} else {
			t.Errorf("Invalid category: %s", tip.Category)
		}
	}

	if vagrantCount != 50 {
		t.Errorf("Expected 50 vagrant tips, got %d", vagrantCount)
	}

	if worldCount != 50 {
		t.Errorf("Expected 50 world tips, got %d", worldCount)
	}
}

func TestGetRandomTip(t *testing.T) {
	tip := GetRandomTip()

	if tip.Text == "" {
		t.Error("Expected non-empty tip text")
	}

	if tip.Source == "" {
		t.Error("Expected non-empty tip source")
	}

	if tip.Category != "vagrant" && tip.Category != "world" {
		t.Errorf("Expected category 'vagrant' or 'world', got %s", tip.Category)
	}

	// Test multiple calls to ensure randomness is working
	tips := make(map[string]bool)
	for i := 0; i < 20; i++ {
		tip := GetRandomTip()
		tips[tip.Text] = true
	}

	// With 100 tips and 20 calls, we should get at least 2 different tips
	if len(tips) < 2 {
		t.Error("GetRandomTip appears to not be random")
	}
}

func TestGetNextTipRotation(t *testing.T) {
	// Test sequential access
	tip0 := GetNextTip(0)
	tip1 := GetNextTip(1)
	tip2 := GetNextTip(2)

	if tip0.Text == "" || tip1.Text == "" || tip2.Text == "" {
		t.Error("Expected non-empty tip text")
	}

	// Test that different indices give different tips
	if tip0.Text == tip1.Text {
		t.Error("Expected different tips for different indices")
	}

	// Test wrap-around
	tip100 := GetNextTip(100)
	if tip100.Text != tip0.Text {
		t.Error("Expected index 100 to wrap to index 0")
	}

	tip101 := GetNextTip(101)
	if tip101.Text != tip1.Text {
		t.Error("Expected index 101 to wrap to index 1")
	}

	// Test negative indices wrap correctly
	tipNeg1 := GetNextTip(-1)
	if tipNeg1.Text == "" {
		t.Error("Expected non-empty tip text for negative index")
	}
}

func TestTipStructure(t *testing.T) {
	for i, tip := range tips {
		if tip.Text == "" {
			t.Errorf("Tip %d has empty Text field", i)
		}

		if tip.Source == "" {
			t.Errorf("Tip %d has empty Source field", i)
		}

		if tip.Category != "vagrant" && tip.Category != "world" {
			t.Errorf("Tip %d has invalid category: %s", i, tip.Category)
		}
	}
}
