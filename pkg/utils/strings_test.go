package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		sep      string
		expected []string
	}{
		{
			name:     "com espaços e vírgulas extras",
			input:    "a, b ,c,, ",
			sep:      ",",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "string vazia",
			input:    "",
			sep:      ",",
			expected: []string{},
		},
		{
			name:     "sem separador",
			input:    "abc",
			sep:      ",",
			expected: []string{"abc"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SplitAndTrim(tt.input, tt.sep)
			assert.Equal(t, tt.expected, result)
		})
	}
}
