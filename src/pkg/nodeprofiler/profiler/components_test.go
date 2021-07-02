package profiler

import (
	"testing"

	"cos.googlesource.com/cos/tools.git/src/pkg/nodeprofiler/utils"
)

func TestCollectUtilization(t *testing.T) {
	tests := []struct {
		name    string
		outputs map[string]utils.ParsedOutput
		want    float64
		wantErr bool
	}{
		{
			name: "simple",
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
			name: "missing columns",
			outputs: map[string]utils.ParsedOutput{
				"vmstat": {
					"us": {"1", "2", "7"},
					"sy": {"0", "1", "3"},
				},
			},
			wantErr: true,
		},
		{
			name:    "empty outputs",
			outputs: map[string]utils.ParsedOutput{},
			wantErr: true,
		},
	}

	for _, test := range tests {
		fakeCPU := &CPU{name: "fakeCPU", metrics: &USEMetrics{}}
		err := fakeCPU.CollectUtilization(test.outputs)
		got := fakeCPU.metrics.Utilization

		if gotErr := err != nil; gotErr != test.wantErr {
			t.Fatalf("CollectUtilization(%v) err %q, wantErr %v", test.outputs, err, test.wantErr)
		}
		if got != test.want {
			t.Errorf("CollectUtilization(%v) = %v, want: %v", test.outputs, test.want, got)
		}
	}
}

func TestCollectSaturation(t *testing.T) {
	tests := []struct {
		name    string
		outputs map[string]utils.ParsedOutput
		want    bool
		wantErr bool
	}{
		{
			name: "non-saturated",
			outputs: map[string]utils.ParsedOutput{
				"vmstat": {
					"r": {"5", "2", "2"},
				},
				"lscpu": {
					"CPU(s):": {"8"},
				},
			},
		},
		{
			name: "saturated",
			outputs: map[string]utils.ParsedOutput{
				"vmstat": {
					"r": {"10", "10", "10"},
				},
				"lscpu": {
					"CPU(s):": {"8"},
				},
			},
			want: true,
		},
		{
			name: "wrong columns",
			outputs: map[string]utils.ParsedOutput{
				"vmstat": {
					"us": {"1", "2", "7"},
					"sy": {"0", "1", "3"},
					"st": {"0", "0", "0"},
				},
				"lscpu": {
					"CPU(s):": {"8"},
				},
			},
			wantErr: true,
		},
		{
			name: "missing lscpu",
			outputs: map[string]utils.ParsedOutput{
				"vmstat": {
					"r": {"10", "10", "10"},
				},
			},
			wantErr: true,
		},
		{
			name:    "empty outputs",
			outputs: map[string]utils.ParsedOutput{},
			wantErr: true,
		},
	}

	for _, test := range tests {
		fakeCPU := &CPU{name: "fakeCPU", metrics: &USEMetrics{}}
		err := fakeCPU.CollectSaturation(test.outputs)
		got := fakeCPU.metrics.Saturation

		if gotErr := err != nil; gotErr != test.wantErr {
			t.Fatalf("CollectSaturation(%v) err %q, wantErr %v", test.outputs, err, test.wantErr)
		}
		if got != test.want {
			t.Errorf("CollectSaturation(%v) = %v, want: %v", test.outputs, test.want, got)
		}
	}
}
