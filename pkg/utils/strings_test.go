package utils

import "testing"

func TestSplitAndTrim(t *testing.T) {
	input := "a, b ,c,, "
	expected := []string{"a", "b", "c"}
	result := SplitAndTrim(input, ",")
	if len(result) != len(expected) {
		t.Fatalf("esperava %d elementos, obteve %d", len(expected), len(result))
	}
	for i, v := range expected {
		if result[i] != v {
			t.Errorf("esperava %q, obteve %q", v, result[i])
		}
	}
}
