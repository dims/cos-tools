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
			want: 4.67,
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
			name:      "missing commands",
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
			name:      "missing commands",
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
			name:      "storage device",
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
			name:      "missing commands",
			component: &StorageDevIO{"fake", &USEMetrics{}},
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
					"r": {"15", "12", "2"},
				},
				"lscpu": {
					"CPU(s)": {"8"},
				},
			},
			want: true,
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
			name:      "missing command",
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
			name:      "missing command",
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
			name:      "missing commands",
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
