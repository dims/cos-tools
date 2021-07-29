package profiler

import (
	"testing"

	"cos.googlesource.com/cos/tools.git/src/pkg/nodeprofiler/utils"
)

func TestCollectUtilization(t *testing.T) {
	tests := []struct {
		name      string
		component Component
		outputs   map[string]utils.ParsedOutput
		want      float64
		wantErr   bool
	}{
		{
			name:      "cpu",
			component: &CPU{"fake", &USEMetrics{}},
			outputs: map[string]utils.ParsedOutput{
				"vmstat": {
					"us": {"1", "2", "7"},
					"sy": {"0", "1", "3"},
					"st": {"0", "0", "0"},
				},
			},
			want: 6.5,
		},
		{
			name:      "vmstat slices of length 1",
			component: &CPU{"fake", &USEMetrics{}},
			outputs: map[string]utils.ParsedOutput{
				"vmstat": {
					"us": {"1"},
					"sy": {"0"},
					"st": {"0"},
				},
			},
			wantErr: true,
		},
		{
			name:      "empty vmstat slices",
			component: &CPU{"fake", &USEMetrics{}},
			outputs: map[string]utils.ParsedOutput{
				"vmstat": {
					"us": {},
					"sy": {},
					"st": {},
				},
			},
			wantErr: true,
		},
		{
			name:      "missing titles",
			component: &CPU{"fake", &USEMetrics{}},
			outputs: map[string]utils.ParsedOutput{
				"vmstat": {
					"us": {"1", "2", "7"},
					"sy": {"0", "1", "3"},
				},
			},
			wantErr: true,
		},
		{
			name:      "missing commands output",
			component: &CPU{"fake", &USEMetrics{}},
			outputs:   map[string]utils.ParsedOutput{},
			wantErr:   true,
		},
		{
			name:      "memory capacity",
			component: &MemCap{"fake", &USEMetrics{}},
			outputs: map[string]utils.ParsedOutput{
				"free": {
					"Mem:used":   {"13"},
					"Swap:used":  {"0"},
					"Mem:total":  {"14520"},
					"Swap:total": {"0"},
				},
			},
			want: 0.001,
		},
		{
			name:      "missing titles",
			component: &MemCap{"fake", &USEMetrics{}},
			outputs: map[string]utils.ParsedOutput{
				"free": {},
			},
			wantErr: true,
		},
		{
			name:      "missing commands output",
			component: &MemCap{"fake", &USEMetrics{}},
			outputs: map[string]utils.ParsedOutput{
				"vmstat": {
					"si": {"0", "0", "3"},
					"so": {"0", "1", "5"},
				},
			},
			wantErr: true,
		},
		{
			name:      "storage device I/O",
			component: &StorageDevIO{"fake", &USEMetrics{}},
			outputs: map[string]utils.ParsedOutput{
				"iostat": {
					"%util": {"4.76", "0.09"},
				},
			},
			want: 2.425,
		},
		{
			name:      "missing titles",
			component: &StorageDevIO{"fake", &USEMetrics{}},
			outputs: map[string]utils.ParsedOutput{
				"iostat": {
					"%idle": {"94.27"},
				},
			},
			wantErr: true,
		},
		{
			name:      "missing commands output",
			component: &StorageDevIO{"fake", &USEMetrics{}},
			outputs:   map[string]utils.ParsedOutput{},
			wantErr:   true,
		},
		{
			name:      "storage capacity",
			component: &StorageCap{"fake", &USEMetrics{}, []string{"/dev/vda"}},
			outputs: map[string]utils.ParsedOutput{
				"df": {
					"Filesystem": {"/dev/vdb", "/dev/vda"},
					"Used":       {"4764604", "50448"},
					"1K-blocks":  {"7864320", "50540"},
				},
			},
			want: 99.82,
		},
		{
			name:      "devices not set",
			component: &StorageCap{"fake", &USEMetrics{}, []string{}},
			outputs: map[string]utils.ParsedOutput{
				"df": {
					"Filesystem": {"/dev/sda", "/dev/vda"},
					"Used":       {"4764604", "50448"},
					"1K-blocks":  {"7864320", "50540"},
				},
			},
			want: 60.59,
		},
		{
			name:      "device (sda) with different partitions",
			component: &StorageCap{"fake", &USEMetrics{}, []string{}},
			outputs: map[string]utils.ParsedOutput{
				"df": {
					"Filesystem": {"/dev/sda1", "/dev/sda8"},
					"Used":       {"95384", "24"},
					"1K-blocks":  {"5971884", "11756"},
				},
			},
			want: 1.59,
		},
		{
			name:      "several occurrences of same device",
			component: &StorageCap{"fake", &USEMetrics{}, []string{"tmpfs"}},
			outputs: map[string]utils.ParsedOutput{
				"df": {
					"Filesystem": {"tmpfs", "tmpfs", "/dev/root", "tmpfs"},
					"Used":       {"0", "468", "1051636", "168"},
					"1K-blocks":  {"1884128", "1884128", "2003760", "1024"},
				},
			},
			want: 0.02,
		},
		{
			name:      "missing [default] device",
			component: &StorageCap{"fake", &USEMetrics{}, []string{}},
			outputs: map[string]utils.ParsedOutput{
				"df": {
					"Filesystem": {"/dev/vdb", "/dev/vda"},
					"Used":       {"4764604", "50448"},
					"1K-blocks":  {"7864320", "50540"},
				},
			},
			wantErr: true,
		},
		{
			name:      "missing titles",
			component: &StorageCap{"fake", &USEMetrics{}, []string{"/dev/vda"}},
			outputs: map[string]utils.ParsedOutput{
				"df": {
					"Use%": {"67%", "0%", "100%", "2%"},
				},
			},
			wantErr: true,
		},
		{
			name:      "missing commands output",
			component: &StorageCap{"fake", &USEMetrics{}, []string{"/dev/vda"}},
			outputs:   map[string]utils.ParsedOutput{},
			wantErr:   true,
		},
	}
	for _, test := range tests {
		err := test.component.CollectUtilization(test.outputs)
		got := test.component.USEMetrics().Utilization
		if gotErr := err != nil; gotErr != test.wantErr {
			t.Fatalf("CollectUtilization(%v) err %v, wantErr %t", test.outputs, err, test.wantErr)
		}
		if got != test.want {
			t.Errorf("CollectUtilization(%v) = %v, want: %v", test.outputs, test.want, got)
		}
	}
}
func TestCollectSaturation(t *testing.T) {
	tests := []struct {
		name      string
		component Component
		outputs   map[string]utils.ParsedOutput
		want      bool
		wantErr   bool
	}{
		{
			name:      "CPU",
			component: &CPU{"fake", &USEMetrics{}},
			outputs: map[string]utils.ParsedOutput{
				"vmstat": {
					"r": {"15", "12", "12"},
				},
				"lscpu": {
					"CPU(s)": {"8"},
				},
			},
			want: true,
		},
		{
			name:      "vmstat slices of length 1",
			component: &CPU{"fake", &USEMetrics{}},
			outputs: map[string]utils.ParsedOutput{
				"vmstat": {
					"r": {"15"},
				},
				"lscpu": {
					"CPU(s)": {"8"},
				},
			},
			wantErr: true,
		},
		{
			name:      "empty vmstat slices",
			component: &CPU{"fake", &USEMetrics{}},
			outputs: map[string]utils.ParsedOutput{
				"vmstat": {
					"r": {},
				},
				"lscpu": {
					"CPU(s)": {"8"},
				},
			},
			wantErr: true,
		},
		{
			name:      "missing titles",
			component: &CPU{"fake", &USEMetrics{}},
			outputs: map[string]utils.ParsedOutput{
				"vmstat": {},
				"lscpu": {
					"CPU(s):": {"8"},
				},
			},
			wantErr: true,
		},
		{
			name:      "missing commands output",
			component: &CPU{"fake", &USEMetrics{}},
			outputs: map[string]utils.ParsedOutput{
				"vmstat": {
					"r": {"10", "10", "10"},
				},
			},
			wantErr: true,
		},
		{
			name:      "Memory capacity",
			component: &MemCap{"fake", &USEMetrics{}},
			outputs: map[string]utils.ParsedOutput{
				"vmstat": {
					"si": {"0", "0", "3"},
					"so": {"0", "1", "5"},
				},
				"free": {
					"Mem:total":  {"14520"},
					"Swap:total": {"0"},
				},
			},
		},
		{
			name:      "missing commands output",
			component: &MemCap{"fake", &USEMetrics{}},
			outputs: map[string]utils.ParsedOutput{
				"vmstat": {
					"si": {"0", "0", "3"},
					"so": {"0", "1", "5"},
				},
			},
			wantErr: true,
		},
		{
			name:      "missing titles",
			component: &MemCap{"fake", &USEMetrics{}},
			outputs: map[string]utils.ParsedOutput{
				"vmstat": {
					"si": {"0", "0", "3"},
					"so": {"0", "1", "5"},
				},
				"free": {},
			},
			wantErr: true,
		},
		{
			name:      "Storage Device",
			component: &StorageDevIO{"fake", &USEMetrics{}},
			outputs: map[string]utils.ParsedOutput{
				"iostat": {
					"aqu-sz": {"0.04", "0.00"},
				},
			},
		},
		{
			name:      "missing commands output",
			component: &StorageDevIO{"fake", &USEMetrics{}},
			outputs:   map[string]utils.ParsedOutput{},
			wantErr:   true,
		},
		{
			name:      "missing titles",
			component: &StorageDevIO{"fake", &USEMetrics{}},
			outputs: map[string]utils.ParsedOutput{
				"iostat": {},
			},
			wantErr: true,
		},
	}
	for _, test := range tests {
		err := test.component.CollectSaturation(test.outputs)
		got := test.component.USEMetrics().Saturation
		if gotErr := err != nil; gotErr != test.wantErr {
			t.Fatalf("CollectSaturation(%v) err %v, wantErr %t", test.outputs, err, test.wantErr)
		}
		if got != test.want {
			t.Errorf("CollectSaturation(%v) = %v, want: %v", test.outputs, test.want, got)
		}
	}
}
