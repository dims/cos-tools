package utils

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseColumns(t *testing.T) {
	tests := []struct {
		name       string
		rows       []string
		allTitles  []string
		wantTitles []string
		want       map[string][]string
		wantErr    bool
	}{
		{
			name: "vmstat's output with spaced rows",
			rows: []string{
				"r  b   swpd   free   buff  cache   si   so    bi    bo   in   cs us sy id wa st",
				"                                                                                ",
				"5  0      0 14827096      0  25608    0    0     2     5   57    2  1  0 99  0  0",
				"                                                                                ",
				"2  0      0 14827096      0  25608    0    0     0     0 1131 1594  2  1 97  0  1",
				"                                                                                ",
				"2  0      0 14827096      0  25608    0    0     0     0 5283 8037  7  3 90  0  0",
			},
			allTitles: []string{"r", "b", "swpd", "free", "buff", "cache", "si", "so", "bi", "bo",
				"in", "cs", "us", "sy", "id", "wa", "st"},
			wantTitles: []string{"us", "sy", "st"},
			want: map[string][]string{
				"us": {"1", "2", "7"},
				"sy": {"0", "1", "3"},
				"st": {"0", "1", "0"},
			},
		},
		{
			name: "df's output with empty titles slice",
			rows: []string{
				"Filesystem      Size  Used Avail Use% Mounted on",
				"/dev/vdb        7.5G  4.5G  2.4G  66% /",
				"none            492K     0  492K   0% /dev",
				"devtmpfs        7.1G     0  7.1G   0% /dev/tty",
				"/dev/vdb        7.5G  4.5G  2.4G  66% /dev/kvm",
			},
			allTitles: []string{"Filesystem", "Size", "Used", "Avail",
				"Use%", "Mounted on"},
			want: map[string][]string{
				"Filesystem": {"/dev/vdb", "none", "devtmpfs", "/dev/vdb"},
				"Size":       {"7.5G", "492K", "7.1G", "7.5G"},
				"Used":       {"4.5G", "0", "0", "4.5G"},
				"Avail":      {"2.4G", "492K", "7.1G", "2.4G"},
				"Use%":       {"66%", "0%", "0%", "66%"},
				"Mounted on": {"/", "/dev", "/dev/tty", "/dev/kvm"},
			},
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
			allTitles: []string{"r", "b", "swpd", "free", "buff"},
			want: map[string][]string{
				"r":    {"5"},
				"b":    {"0"},
				"swpd": {"0"},
				"free": {"14827096"},
				"buff": {"0"},
			},
		},
		{
			name: "repeated headers",
			rows: []string{
				"Device            r/s     w/s     rkB/s     wkB/s   rrqm/s   wrqm/s  %rrqm  %wrqm r_await w_await aqu-sz rareq-sz wareq-sz  svctm  %util",
				"vdb              0.01    0.58      0.86     20.79     0.00     0.19   0.24  25.02    8.44 1552.07   0.90    96.44    35.68  95.00   5.62",
				"vda              0.00    0.00      0.04      0.00     0.00     0.00   2.73   0.00    3.08    0.00   0.00    62.55     0.00   2.20   0.00",
				"																																		 ",
				"Device            r/s     w/s     rkB/s     wkB/s   rrqm/s   wrqm/s  %rrqm  %wrqm r_await w_await aqu-sz rareq-sz wareq-sz  svctm  %util",
				"																																		 ",
				"Device            r/s     w/s     rkB/s     wkB/s   rrqm/s   wrqm/s  %rrqm  %wrqm r_await w_await aqu-sz rareq-sz wareq-sz  svctm  %util",
				"																																		 ",
				"Device            r/s     w/s     rkB/s     wkB/s   rrqm/s   wrqm/s  %rrqm  %wrqm r_await w_await aqu-sz rareq-sz wareq-sz  svctm  %util",
			},
			allTitles: []string{"Device", "r/s", "w/s", "rkB/s", "wkB/s", "rrqm/s", "wrqm/s", "%rrqm", "%wrqm", "%r_await", "w_await", "aqu-sz", "rareq-sz",
				"wareq-sz", "svctm", "%util"},
			wantTitles: []string{"%util"},
			want: map[string][]string{
				"%util": {"5.62", "0.00"},
			},
		},
		{
			name: "missing titles on some columns",
			rows: []string{
				"              total        used        free      shared  buff/cache   available",
				"Mem:          14520          13       14481           0          25       14506",
				"Swap:             0           0           0",
			},
			allTitles:  []string{"total", "used", "free", "shared", "buff/cache", "available"},
			wantTitles: []string{"total"},
			wantErr:    true,
		},
		{
			name: "unknown titles",
			rows: []string{
				"r  b   swpd    free   buff   cache",
				"5  0      0 14827096     0   25608",
			},
			allTitles:  []string{"r", "b", "swpd", "free", "buff", "cache"},
			wantTitles: []string{"r", "b", "used"},
			wantErr:    true,
		},
		{
			name: "incomplete slice of all titles",
			rows: []string{
				"Filesystem      Size  Used Avail Use% Mounted on",
				"/dev/vdb        7.5G  4.5G  2.4G  67% /",
				"none            492K     0  492K   0% /dev",
				"devtmpfs        7.1G     0  7.1G   0% /dev/tty",
				"/dev/vdb        7.5G  4.5G  2.4G  67% /dev/kvm",
			},
			allTitles: []string{"Filesystem", "Size", "Used", "Avail", "Use%"},
			wantErr:   true,
		},
	}
	for _, test := range tests {
		got, err := ParseColumns(test.rows, test.allTitles, test.wantTitles...)
		if gotErr := err != nil; gotErr != test.wantErr {
			t.Fatalf("ParseColumns(%v, %v, %v) err %v, wantErr: %t", test.rows, test.allTitles, test.wantTitles, err, test.wantErr)
		}
		if diff := cmp.Diff(test.want, got); diff != "" {
			t.Errorf("Ran ParseColumns(%v, %v, %v), but got mismatch between got and want (+got, -want): \n diff %s", test.rows,
				test.allTitles, test.wantTitles, diff)
		}
	}
}
func TestParseRows(t *testing.T) {
	tests := []struct {
		name    string
		rows    []string
		delim   string
		titles  []string
		want    map[string][]string
		wantErr bool
	}{
		{
			name: "lscpu's output with spaced rows",
			rows: []string{
				"Architecture:        x86_64",
				"							 ",
				"CPU op-mode(s):      32-bit, 64-bit",
				"Byte Order:          Little Endian",
				"								 ",
				"Address sizes:       39 bits physical, 48 bits virtual",
				"							 ",
				"CPU(s):              8",
				"On-line CPU(s) list: 0-7",
			},
			delim:  ":",
			titles: []string{"CPU(s)"},
			want: map[string][]string{
				"CPU(s)": {"8"},
			},
		},
		{
			name: "whitespace delimiter",
			rows: []string{
				"vdb               0.63         0.86        22.19     760760   19680952",
				"vda               0.00         0.04         0.00      37845          0",
			},
			delim: " ",
			want: map[string][]string{
				"vdb": {"0.63", "0.86", "22.19", "760760", "19680952"},
				"vda": {"0.00", "0.04", "0.00", "37845", "0"},
			},
		},
		{
			name: "empty slice",
			rows: []string{},
			want: map[string][]string{},
		},
		{
			name: "empty titles",
			rows: []string{
				"processor       : 0",
				"vendor_id       : GenuineIntel",
				"cpu family      : 6",
				"model           : 142",
				"model name      : 06/8e",
			},
			delim:  ":",
			titles: []string{},
			want: map[string][]string{
				"processor":  {"0"},
				"vendor_id":  {"GenuineIntel"},
				"cpu family": {"6"},
				"model":      {"142"},
				"model name": {"06/8e"},
			},
		},
		{
			name: "wrong delimiter",
			rows: []string{
				"vdb               0.63         0.86        22.30     760860   19808528",
				"vda               0.00         0.04         0.00      37845          0",
			},
			delim:   ":",
			wantErr: true,
		},
		{
			name: "unknown titles",
			rows: []string{
				"Device           tps    kB_read/s    kB_wrtn/s    kB_read    kB_wrtn",
				"vdb             0.60         1.54        21.54     836516   11703972",
				"vda             0.00         0.07         0.00      37901          0",
			},
			delim:   " ",
			titles:  []string{"sda"},
			wantErr: true,
		},
	}
	for _, test := range tests {
		got, err := ParseRows(test.rows, test.delim, test.titles...)
		if gotErr := err != nil; gotErr != test.wantErr {
			t.Fatalf("ParseRows(%v, %v) = %v, wantErr: %t", test.rows, test.titles, err, test.wantErr)
		}
		if diff := cmp.Diff(test.want, got); diff != "" {
			t.Errorf("Ran ParseRows(%v, %v), but got mismatch between got and want (+got,-want): \n diff %s", test.rows, test.titles, diff)
		}
	}
}
func TestParseRowsAndColumns(t *testing.T) {
	tests := []struct {
		name    string
		rows    []string
		titles  []string
		want    map[string][]string
		wantErr bool
	}{
		{
			name: "free",
			rows: []string{
				"              total        used        free      shared  buff/cache   available",
				"Mem:          14520          13       14481           0          25       14506",
				"Swap:             0           0           0",
			},
			titles: []string{"Mem:used", "Mem:total", "Swap:used", "Swap:total"},
			want: map[string][]string{
				"Mem:used":   {"13"},
				"Mem:total":  {"14520"},
				"Swap:used":  {"0"},
				"Swap:total": {"0"},
			},
		},
		{
			name: "iostat",
			rows: []string{
				"Device             tps    kB_read/s    kB_wrtn/s    kB_read    kB_wrtn",
				"                                                                      ",
				"vdb               1.27         3.79        41.80     732408    8072028",
				"                                                                       ",
				"vda               0.00         0.20         0.00      37845          0",
			},
			titles:  []string{"vdb:tps", "vda:kB_read", "vdb:kB_wrtn"},
			wantErr: true,
		},
		{
			name: "wrongly formatted titles",
			rows: []string{
				"                  tps    kB_read/s    kB_wrtn/s    kB_read    kB_wrtn",
				"vdb               1.27         3.79        41.80     732408    8072028",
				"vda               0.00         0.20         0.00      37845          0",
			},
			titles:  []string{":tps"},
			wantErr: true,
		},
	}
	for _, test := range tests {
		got, err := ParseRowsAndColumns(test.rows, test.titles...)
		if gotErr := err != nil; gotErr != test.wantErr {
			t.Fatalf("ParseRowsAndColumns(%v, %v) = %v, wantErr: %t", test.rows, test.titles, err, test.wantErr)
		}
		if diff := cmp.Diff(test.want, got); diff != "" {
			t.Errorf("Ran ParseRowsAndColumns(%v, %v), but got mismatch between got and want (+got, -want): \n diff %s", test.rows, test.titles, diff)
		}
	}
}
