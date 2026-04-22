// SPDX-License-Identifier: GPL-3.0-or-later
package poll

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ReadMem returns (usedMiB, totalMiB) from /proc/meminfo.
func ReadMem() (used, total uint64, err error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 1, err
	}
	defer f.Close()

	info := make(map[string]uint64)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		colon := strings.IndexByte(line, ':')
		if colon < 0 {
			continue
		}
		key := strings.TrimSpace(line[:colon])
		var v uint64
		fmt.Sscanf(strings.TrimSpace(line[colon+1:]), "%d", &v)
		info[key] = v
	}

	totalKB := info["MemTotal"]
	availKB, ok := info["MemAvailable"]
	if !ok {
		availKB = info["MemFree"]
	}
	usedKB := totalKB - availKB

	return usedKB / 1024, totalKB / 1024, sc.Err()
}
