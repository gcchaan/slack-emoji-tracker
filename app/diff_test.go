package main

import (
	"testing"
	"github.com/google/go-cmp/cmp"
)

func TestDiffSortedSlices(t *testing.T) {
	tests := []struct {
		name        string
		oldSlice    []string
		newSlice    []string
		wantAdded   []string
		wantRemoved []string
	}{
		{
			name:        "no changes",
			oldSlice:    []string{"apple", "banana"},
			newSlice:    []string{"apple", "banana"},
			wantAdded:   nil,
			wantRemoved: nil,
		},
		{
			name:        "emojis added",
			oldSlice:    []string{"apple", "banana"},
			newSlice:    []string{"apple", "banana", "cherry"},
			wantAdded:   []string{"cherry"},
			wantRemoved: nil,
		},
		{
			name:        "emojis removed",
			oldSlice:    []string{"apple", "banana", "cherry"},
			newSlice:    []string{"apple", "cherry"},
			wantAdded:   nil,
			wantRemoved: []string{"banana"},
		},
		{
			name:        "both addition and removal",
			oldSlice:    []string{"apple", "banana"},
			newSlice:    []string{"banana", "cherry"},
			wantAdded:   []string{"cherry"},
			wantRemoved: []string{"apple"},
		},
		{
			name:        "initial run with empty cache",
			oldSlice:    []string{},
			newSlice:    []string{"apple", "banana"},
			wantAdded:   []string{"apple", "banana"},
			wantRemoved: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAdded, gotRemoved := diffSortedSlices(tt.oldSlice, tt.newSlice)

			if diff := cmp.Diff(tt.wantAdded, gotAdded); diff != "" {
				t.Errorf("diffSortedSlices() added mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantRemoved, gotRemoved); diff != "" {
				t.Errorf("diffSortedSlices() removed mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
