package utils

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSumAtoi(t *testing.T) {
	tests := []struct {
		name    string
		slice   []string
		want    int
		wantErr bool
	}{
		{
			name:  "sum ints",
			slice: []string{"6", "4", "5"},
			want:  15,
		},
		{
			name:    "sum floats",
			slice:   []string{"4.5", "6.4", "2.9"},
			wantErr: true,
		},
		{
			name:  "empty slice",
			slice: []string{},
		},
		{
			name:    "illegal value in slice",
			slice:   []string{"two", "1"},
			wantErr: true,
		},
	}

	for _, test := range tests {
		got, err := SumAtoi(test.slice)
		if gotErr := err != nil; gotErr != test.wantErr {
			t.Fatalf("SumAtoi(%v) = err %v, wantErr %t", test.slice, err, test.wantErr)
		}
		if got != test.want {
			t.Errorf("SumAtoi(%v) = %v, want: %v", test.slice, got, test.want)
		}
	}
}

func TestSumParseFloat(t *testing.T) {
	tests := []struct {
		name    string
		slice   []string
		want    float64
		wantErr bool
	}{
		{
			name:  "sum floats",
			slice: []string{"6.1", "4.46", "5.3"},
			want:  15.86,
		},
		{
			name:  "sum ints",
			slice: []string{"4", "6", "2"},
			want:  12.0,
		},
		{
			name:  "empty slice",
			slice: []string{},
		},
		{
			name:    "illegal value in slice",
			slice:   []string{"two point 5", "1"},
			wantErr: true,
		},
	}

	for _, test := range tests {
		got, err := SumParseFloat(test.slice)
		if gotErr := err != nil; gotErr != test.wantErr {
			t.Fatalf("SumParseFloat(%v) = err %v, wantErr %t", test.slice, err, test.wantErr)
		}
		if got != test.want {
			t.Errorf("SumParseFloat(%v) = %v, want: %v", test.slice, got, test.want)
		}
	}
}

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
