package utils

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestTrimCharacter(t *testing.T) {
	tests := []struct {
		name      string
		slice     []string
		character string
		want      []string
	}{
		{
			name:      "percent character",
			slice:     []string{"Use%", "Util%", "Capacity%"},
			character: "%",
			want:      []string{"Use", "Util", "Capacity"},
		},
		{
			name:      "whitespace character",
			slice:     []string{"File   ", "Size    ", "Avail  ", "Used "},
			character: " ",
			want:      []string{"File", "Size", "Avail", "Used"},
		},
		{
			name:  "empty slice",
			slice: []string{},
		},
		{
			name:      "unknown character",
			slice:     []string{"iVar", "iClass", "iPlay"},
			character: "%",
			want:      []string{"iVar", "iClass", "iPlay"},
		},
	}

	for _, test := range tests {
		got := TrimCharacter(test.slice, test.character)
		if diff := cmp.Diff(test.want, got); diff != "" {
			t.Errorf("Ran TrimCharacter(%v, %v) but got mismatch between got and want(+got, -want): \n diff %s", test.slice, test.character, diff)
		}
	}
}
