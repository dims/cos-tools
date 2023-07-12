package modules

import (
	"testing"
)

func TestSetModuleParameters(t *testing.T) {
	for _, tc := range []struct {
		testName         string
		value            string
		module           string
		moduleParameters string
		expectError      bool
	}{
		{"param", "nvidia.NVreg_EnableGpuFirmware=0", "nvidia", "NVreg_EnableGpuFirmware=0", false},
		{"param incorrect module", "nvidia,NVreg_EnableGpuFirmware=0", "", "", true},
		{"param incorrect key", "nvidia.NVreg_EnableGpuFirmware", "", "", true},
		{"param incorrect value", "nvidia.NVreg_EnableGpuFirmware=", "", "", true},
	} {
		t.Run(tc.testName, func(t *testing.T) {
			m := NewModuleParameters()
			err := m.Set(tc.value)
			if (err == nil) == tc.expectError {
				t.Errorf("Unexpected error: %v", err)
			}
			if tc.module != "" {
				if len(m[tc.module]) == 0 {
					t.Errorf("module %v not found", tc.module)
				}
				if m[tc.module][0] != tc.moduleParameters {
					t.Errorf("Unexpected parameters want %v, got %v", tc.moduleParameters, m[tc.module][0])
				}
			}
		})
	}
}
