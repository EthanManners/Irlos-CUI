// SPDX-License-Identifier: GPL-3.0-or-later
package poll

import (
	"github.com/NVIDIA/go-nvml/pkg/nvml"
)

// nvmlState tracks whether NVML initialised successfully so we only try once.
var nvmlState struct {
	tried bool
	ok    bool
}

// ReadGPU returns (utilPercent, tempCelsius).
// Returns (-1, -1) if NVML is unavailable (no NVIDIA GPU or driver).
func ReadGPU() (util, temp int) {
	if !nvmlState.tried {
		nvmlState.tried = true
		ret := nvml.Init()
		nvmlState.ok = (ret == nvml.SUCCESS)
		if !nvmlState.ok {
			return -1, -1
		}
	}
	if !nvmlState.ok {
		return -1, -1
	}

	device, ret := nvml.DeviceGetHandleByIndex(0)
	if ret != nvml.SUCCESS {
		return -1, -1
	}

	rates, ret := nvml.DeviceGetUtilizationRates(device)
	if ret != nvml.SUCCESS {
		return -1, -1
	}

	t, ret := nvml.DeviceGetTemperature(device, nvml.TEMPERATURE_GPU)
	if ret != nvml.SUCCESS {
		return int(rates.Gpu), -1
	}

	return int(rates.Gpu), int(t)
}

// ShutdownNVML releases NVML resources. Call on program exit.
func ShutdownNVML() {
	if nvmlState.ok {
		nvml.Shutdown() //nolint:errcheck
	}
}
