package profiler

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name    string
		fakeCmd Command
		want    map[string][]string
		wantErr bool
	}{
		{
			name: "vmstat",
			fakeCmd: &vmstat{
				name:   "testdata/vmstat.sh",
				count:  3,
				titles: []string{"us", "st", "sy"},
			},
			want: map[string][]string{
				"us": {"1", "2", "7"},
				"sy": {"0", "1", "3"},
				"st": {"0", "0", "0"},
			},
		},
		{
			name: "lscpu",
			fakeCmd: &lscpu{
				name:   "testdata/lscpu.sh",
				titles: []string{"CPU(s)"},
			},
			want: map[string][]string{
				"CPU(s)": {"8"},
			},
		},
		{
			name: "free",
			fakeCmd: &free{
				name: "testdata/free.sh",
				titles: []string{"Mem:used", "Mem:total",
					"Swap:used", "Swap:total"},
			},
			want: map[string][]string{
				"Mem:used":   {"13"},
				"Mem:total":  {"14520"},
				"Swap:used":  {"0"},
				"Swap:total": {"0"},
			},
		},
		{
			name: "iostat",
			fakeCmd: &iostat{
				name:   "testdata/iostat.sh",
				flags:  "xdz",
				titles: []string{"%util"},
			},
			want: map[string][]string{
				"%util": {"5.59", "0.00"},
			},
		},
		{
			name: "no titles",
			fakeCmd: &vmstat{
				name:  "testdata/vmstat.sh",
				count: 2,
			},
			want: map[string][]string{
				"r":  {"3", "1"},
				"us": {"1", "2"},
				"sy": {"0", "1"},
				"id": {"96", "98"},
				"wa": {"3", "0"},
				"st": {"0", "0"},
			},
		},
		{
			name: "spaced rows",
			fakeCmd: &vmstat{
				name:   "testdata/vmstat.sh",
				count:  4,
				titles: []string{"us", "st", "sy"},
			},
			want: map[string][]string{
				"us": {"7", "3", "1", "1"},
				"sy": {"2", "2", "2", "0"},
				"st": {"0", "0", "0", "0"},
			},
		},
		{
			name: "illegal argument",
			fakeCmd: &vmstat{
				name:  "testdata/vmstat.sh",
				count: -4,
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		got, err := test.fakeCmd.Run()

		if gotErr := err != nil; gotErr != test.wantErr {
			t.Fatalf("Run() err %v, wantErr %t", err, test.wantErr)
		}
		if diff := cmp.Diff(test.want, got); diff != "" {
			t.Errorf("Ran Run(), but got mismatch between got and want (-got, +want): \n diff %s", diff)
		}
	}
}
