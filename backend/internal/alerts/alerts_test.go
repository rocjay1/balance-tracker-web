package alerts

import (
	"testing"
	"time"
)

func TestIsSameDay(t *testing.T) {
	tests := []struct {
		name     string
		t1       time.Time
		t2       time.Time
		expected bool
	}{
		{
			name:     "Same day",
			t1:       time.Date(2023, 10, 26, 10, 0, 0, 0, time.UTC),
			t2:       time.Date(2023, 10, 26, 14, 30, 0, 0, time.UTC),
			expected: true,
		},
		{
			name:     "Different day",
			t1:       time.Date(2023, 10, 26, 10, 0, 0, 0, time.UTC),
			t2:       time.Date(2023, 10, 27, 10, 0, 0, 0, time.UTC),
			expected: false,
		},
		{
			name:     "Different month",
			t1:       time.Date(2023, 10, 26, 10, 0, 0, 0, time.UTC),
			t2:       time.Date(2023, 11, 26, 10, 0, 0, 0, time.UTC),
			expected: false,
		},
		{
			name:     "Different year",
			t1:       time.Date(2023, 10, 26, 10, 0, 0, 0, time.UTC),
			t2:       time.Date(2024, 10, 26, 10, 0, 0, 0, time.UTC),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSameDay(tt.t1, tt.t2); got != tt.expected {
				t.Errorf("isSameDay() = %v, want %v", got, tt.expected)
			}
		})
	}
}
