//go:build windows

package network

import (
	"fmt"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

func Traffic(interfaceName string) (uint64, uint64, error) {
	if strings.TrimSpace(interfaceName) == "" {
		return 0, 0, fmt.Errorf("empty network interface")
	}
	var table *windows.MibIfTable2
	if err := windows.GetIfTable2Ex(windows.MibIfTableNormal, &table); err != nil {
		return 0, 0, err
	}
	defer windows.FreeMibTable(unsafe.Pointer(table))
	rows := unsafe.Slice(&table.Table[0], table.NumEntries)
	for _, row := range rows {
		alias := windows.UTF16ToString(row.Alias[:])
		description := windows.UTF16ToString(row.Description[:])
		if strings.EqualFold(alias, interfaceName) || strings.EqualFold(description, interfaceName) {
			return row.InOctets, row.OutOctets, nil
		}
	}
	return 0, 0, fmt.Errorf("network interface not found: %s", interfaceName)
}
