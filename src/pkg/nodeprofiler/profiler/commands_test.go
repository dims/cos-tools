package profiler

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name    string
		fakeCmd Command
		opts    Options
		want    map[string][]string
		wantErr bool
	}{
		{
			name:    "vmstat",
			fakeCmd: &vmstat{"testdata/vmstat.sh"},
			opts: Options{
				Delay:  1,
				Count:  3,
				Titles: []string{"us", "st", "sy"},
			},
			want: map[string][]string{
				"us": {"1", "2", "7"},
				"sy": {"0", "1", "3"},
				"st": {"0", "0", "0"},
			},
		},
		{
			name:    "lscpu",
			fakeCmd: &lscpu{"testdata/lscpu.sh"},
			opts: Options{
				Titles: []string{"CPU(s)"},
			},
			want: map[string][]string{
				"CPU(s)": {"8"},
			},
		},
		{
			name:    "no titles",
			fakeCmd: &vmstat{"testdata/vmstat.sh"},
			opts: Options{
				Delay: 1,
				Count: 2,
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
			name:    "spaced rows",
			fakeCmd: &vmstat{"testdata/vmstat.sh"},
			opts: Options{
				Delay:  1,
				Count:  4,
				Titles: []string{"us", "st", "sy"},
			},
			want: map[string][]string{
				"us": {"7", "3", "1", "1"},
				"sy": {"2", "2", "2", "0"},
				"st": {"0", "0", "0", "0"},
			},
		},
	}

	for _, test := range tests {
		got, err := test.fakeCmd.Run(test.opts)

		if gotErr := err != nil; gotErr != test.wantErr {
			t.Fatalf("Run(%v) err %q, wantErr %v", test.opts, err, test.wantErr)
		}
		if diff := cmp.Diff(test.want, got); diff != "" {
			t.Errorf("Ran Run(%v), but got mismatch between got and want (-got, +want): \n diff %s", test.opts, diff)
		}
	}
}
