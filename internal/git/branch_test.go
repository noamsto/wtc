package git

import "testing"

func TestParseBranchList(t *testing.T) {
	t.Run("mixed prefixes", func(t *testing.T) {
		output := "* main\n  feature-a\n+ feature-b\n  feature-c\n"
		branches := ParseBranchList(output)
		expected := []string{"main", "feature-a", "feature-b", "feature-c"}
		for _, b := range expected {
			if !branches[b] {
				t.Errorf("expected branch %q to be present", b)
			}
		}
		if len(branches) != len(expected) {
			t.Errorf("expected %d branches, got %d", len(expected), len(branches))
		}
	})

	t.Run("empty output", func(t *testing.T) {
		branches := ParseBranchList("")
		if len(branches) != 0 {
			t.Errorf("expected 0 branches, got %d", len(branches))
		}
	})

	t.Run("trailing newline only", func(t *testing.T) {
		branches := ParseBranchList("\n")
		if len(branches) != 0 {
			t.Errorf("expected 0 branches, got %d", len(branches))
		}
	})
}
