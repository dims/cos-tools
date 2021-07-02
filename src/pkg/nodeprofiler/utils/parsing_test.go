package utils

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseColumns(t *testing.T) {
	tests := []struct {
		name    string
		rows    []string
		titles  []string
		want    map[string][]string
		wantErr bool
	}{

		{
			name: "basic",
			rows: []string{
				"r  b   swpd   free   buff  cache   si   so    bi    bo   in   cs us sy id wa st",
				"5  0      0 14827096      0  25608    0    0     2     5   57    2  1  0 99  0  0",
				"2  0      0 14827096      0  25608    0    0     0     0 1131 1594  2  1 97  0  1",
				"2  0      0 14827096      0  25608    0    0     0     0 5283 8037  7  3 90  0  0",
			},
			titles: []string{"us", "sy", "st"},
			want: map[string][]string{
				"us": {"1", "2", "7"},
				"sy": {"0", "1", "3"},
				"st": {"0", "1", "0"},
			},
		},
		{
			name: "spaced rows",
			rows: []string{
				"r  b   swpd   free   buff  cache   si   so    bi    bo   in   cs us sy id wa st",
				"                                                                                ",
				"2  0      0 14827096      0  25608    0    0     0     0 1131 1594  2  1 97  0  0",
				"                                                                                ",
				"5  0      0 14827780      0  25740    0    0     0     5    3    7  1  0 96  3  0",
				"                                                                                ",
				"1  0      0 14827724      0  25608    0    0     1     6   10   37  1  0 96  2  0",
			},
			titles: []string{"r"},
			want: map[string][]string{
				"r": {"2", "5", "1"},
			},
		},
		{
			name: "unknown titles",
			rows: []string{
				"r  b   swpd   free   buff  cache   si   so    bi    bo   in   cs us sy id wa st",
				"5  0      0 14827096      0  25608    0    0     2     5   57    2  1  0 99  0  0",
			},
			titles:  []string{"us", "sy", "steal"},
			wantErr: true,
		},
		{
			name: "empty slice",
			rows: []string{},
			want: map[string][]string{},
		},
		{
			name: "empty titles",
			rows: []string{
				"r  b   swpd   free   buff",
				"5  0      0 14827096      0",
			},
			want: map[string][]string{
				"r":    {"5"},
				"b":    {"0"},
				"swpd": {"0"},
				"free": {"14827096"},
				"buff": {"0"},
			},
		},
	}

	for _, test := range tests {
		got, err := ParseColumns(test.rows, test.titles...)
		if gotErr := err != nil; gotErr != test.wantErr {
			t.Fatalf("ParseColumns(%v, %v) err %q, wantErr: %v", test.rows, test.titles, err, test.wantErr)
		}
		if diff := cmp.Diff(test.want, got); diff != "" {
			t.Errorf("Ran ParseColumns(%v, %v), but got mismatch between got and want (-got, +want): \n diff %s", test.rows, test.titles, diff)
		}
	}
}

func TestParseRows(t *testing.T) {
	tests := []struct {
		name    string
		rows    []string
		titles  []string
		want    map[string][]string
		wantErr bool
	}{
		{
			name: "lscpu's output",
			rows: []string{
				"Architecture:        x86_64",
				"CPU op-mode(s):      32-bit, 64-bit",
				"Byte Order:          Little Endian",
				"Address sizes:       39 bits physical, 48 bits virtual",
				"CPU(s):              8",
				"On-line CPU(s) list: 0-7",
			},

			titles: []string{"CPU(s)"},
			want: map[string][]string{
				"CPU(s)": {"8"},
			},
		},
		{
			name: "spaced rows",
			rows: []string{
				"Architecture:        x86_64",
				"							 ",
				"CPU op-mode(s):      32-bit, 64-bit",
				"							 ",
				"CPU(s):              8",
			},
			titles: []string{"CPU(s)"},
			want: map[string][]string{
				"CPU(s)": {"8"},
			},
		},
		{
			name: "free's output",
			rows: []string{
				"Mem:          14518          13       14480           0          25       14505",
				"Swap:             0           0           0									",
			},
			titles: []string{"Mem", "Swap"},
			want: map[string][]string{
				"Mem":  {"14518", "13", "14480", "0", "25", "14505"},
				"Swap": {"0", "0", "0"},
			},
		},
		{
			name: "unknown titles",
			rows: []string{
				"Device:             tps    kB_read/s    kB_wrtn/s    kB_read    kB_wrtn",
				"vdb:               0.60         1.54        21.54     836516   11703972",
				"vda:               0.00         0.07         0.00      37901          0",
			},
			titles:  []string{"sda"},
			wantErr: true,
		},
		{
			name: "empty slice",
			rows: []string{},
			want: map[string][]string{},
		},
		{
			name: "empty titles",
			rows: []string{
				"processor: 0",
				"vendor_id: GenuineIntel",
				"cpu family: 6",
				"model: 142",
				"model name: 06/8e",
			},
			titles: []string{},
			want: map[string][]string{
				"processor":  {"0"},
				"vendor_id":  {"GenuineIntel"},
				"cpu family": {"6"},
				"model":      {"142"},
				"model name": {"06/8e"},
			},
		},
	}

	for _, test := range tests {
		got, err := ParseRows(test.rows, test.titles...)
		if gotErr := err != nil; gotErr != test.wantErr {
			t.Fatalf("ParseRows(%v, %v) = %q, wantErr: %v", test.rows, test.titles, err, test.wantErr)
		}
		if diff := cmp.Diff(test.want, got); diff != "" {
			t.Errorf("Ran ParseRows(%v, %v), but got mismatch between got and want (-got, +want): \n diff %s", test.rows, test.titles, diff)
		}
	}
}
