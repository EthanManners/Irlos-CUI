// SPDX-License-Identifier: GPL-3.0-or-later
package poll

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// cpuPrev stores the previous /proc/stat sample for delta calculation.
// Access is single-threaded from the poll goroutine.
var cpuPrev struct {
	idle  uint64
	total uint64
	valid bool
}

// ReadCPU returns the CPU utilisation percentage (0–100) by diffing
// two consecutive /proc/stat samples.
func ReadCPU() (float64, error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return 0, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	if !sc.Scan() {
		return 0, fmt.Errorf("empty /proc/stat")
	}
	line := sc.Text()

	// Format: "cpu  user nice system idle iowait irq softirq steal guest guest_nice"
	fields := strings.Fields(line)
	if len(fields) < 5 || fields[0] != "cpu" {
		return 0, fmt.Errorf("unexpected /proc/stat format")
	}

	var vals [10]uint64
	count := 0
	for i := 1; i < len(fields) && count < 10; i++ {
		var v uint64
		fmt.Sscanf(fields[i], "%d", &v)
		vals[count] = v
		count++
	}

	idle := vals[3] // idle + iowait
	if count > 4 {
		idle += vals[4]
	}
	var total uint64
	for i := 0; i < count; i++ {
		total += vals[i]
	}

	if !cpuPrev.valid {
		cpuPrev.idle = idle
		cpuPrev.total = total
		cpuPrev.valid = true
		return 0, nil
	}

	dIdle := idle - cpuPrev.idle
	dTotal := total - cpuPrev.total
	cpuPrev.idle = idle
	cpuPrev.total = total

	if dTotal == 0 {
		return 0, nil
	}
	return 100.0 * float64(dTotal-dIdle) / float64(dTotal), nil
}
