//go:build darwin

package inventory

import (
	"bufio"
	"bytes"
	"regexp"
	"strconv"
	"strings"
)

// vmStatPageSizeRe extracts the page size from the vm_stat header line.
var vmStatPageSizeRe = regexp.MustCompile(`page size of (\d+) bytes`)

// parseVmStat parses macOS vm_stat output, returning the page size in bytes
// and a map of statistic name to page count. Statistic names are lower-cased
// and trimmed (e.g. "pages free", "pages active", "pages wired down").
func parseVmStat(data []byte) (pageSize uint64, pages map[string]uint64) {
	pages = make(map[string]uint64)

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()

		// First line: "Mach Virtual Memory Statistics: (page size of 16384 bytes)"
		if m := vmStatPageSizeRe.FindStringSubmatch(line); len(m) == 2 {
			pageSize, _ = strconv.ParseUint(m[1], 10, 64)
			continue
		}

		// Stat lines: 'Pages free:                               12345.'
		// Some lines have quotes: '"Translation faults":   12345.'
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}

		key = strings.Trim(strings.TrimSpace(key), `"`)
		key = strings.ToLower(key)

		val = strings.TrimSpace(val)
		val = strings.TrimSuffix(val, ".")

		n, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			continue
		}

		pages[key] = n
	}

	return pageSize, pages
}
