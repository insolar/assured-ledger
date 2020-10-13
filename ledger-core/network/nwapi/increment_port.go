package nwapi

import (
	"strconv"
	"strings"
)

// IncrementPort increments port number if it not equals 0
func IncrementPort(address string) string {
	parts := strings.Split(address, ":")
	if len(parts) < 2 {
		panic("failed to get port from address")
	}
	port, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return address
	}

	if port != 0 {
		port++
	}

	parts = append(parts[:len(parts)-1], strconv.Itoa(port))
	return strings.Join(parts, ":")
}
