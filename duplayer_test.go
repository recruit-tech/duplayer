package duplayer

import (
	"testing"
)

func TestIsDuplicated(t *testing.T) {

	upper := files{
		map[string]int64{"a1": 10, "b1/b2": 20},
		map[string]int64{"s1": 1, "t1/t2": 1},
		map[string]int64{"x1": 14, "y1/y2": 25},
		int64(138),
	}

	tests := []struct {
		input    string
		expected bool
	}{
		{
			input:    "a1",
			expected: true,
		}, {
			input:    "a1/a2",
			expected: true,
		}, {
			input:    "b1",
			expected: false,
		}, {
			input:    "b1/b2",
			expected: true,
		}, {
			input:    "s1",
			expected: false,
		}, {
			input:    "s1/s2",
			expected: true,
		}, {
			input:    "t1",
			expected: false,
		}, {
			input:    "t1/t2",
			expected: false,
		}, {
			input:    "x1",
			expected: true,
		}, {
			input:    "x1/x2",
			expected: false,
		}, {
			input:    "y1",
			expected: false,
		}, {
			input:    "y1/y2",
			expected: true,
		},
	}

	for _, test := range tests {
		got := upper.isDuplicate(test.input)
		if got != test.expected {
			t.Errorf("%q: unexpected result. expected: %v, but got: %v", test.input, test.expected, got)
		}
	}
}
