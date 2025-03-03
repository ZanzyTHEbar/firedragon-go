package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
)

// writePIDFile writes the process ID to a file
func WritePIDFile(pid int, client bool) error {
	// Determine the PID file path based on whether it's a client or server

	serverOrClient := "server"

	if client {
		serverOrClient = "client"
	}

	pidFile := "/tmp/perception-engine-" + serverOrClient + ".pid"
	return os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644)
}

// removePIDFile removes the PID file
func RemovePIDFile(client bool) {
	// Determine the PID file path based on whether it's a client or server
	serverOrClient := "server"
	if client {
		serverOrClient = "client"
	}
	pidFile := "/tmp/perception-engine-" + serverOrClient + ".pid"
	os.Remove(pidFile)
}

// IsServerRunning checks if the server is already running by looking for a PID file
// and verifying if the process with that PID exists.
func IsServerRunning(client bool) bool {
	// Determine the PID file path based on whether it's a client or server
	serverOrClient := "server"
	if client {
		serverOrClient = "client"
	}
	pidFile := "/tmp/perception-engine-" + serverOrClient + ".pid"

	data, err := os.ReadFile(pidFile)
	if err != nil {
		// PID file doesn't exist or can't be read
		fmt.Printf("Debug: Could not read PID file: %v\n", err)
		return false
	}

	// Remove any extra whitespace and convert the content to an integer
	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		fmt.Printf("Debug: Invalid PID format: %v\n", err)
		// The PID file is corrupt
		os.Remove(pidFile)
		return false
	}

	// Check if the process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		fmt.Printf("Debug: Process not found: %v\n", err)
		os.Remove(pidFile)
		return false
	}

	// On Unix-like systems, os.FindProcess always succeeds, so we need to send
	// signal 0 to actually check if the process exists
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		fmt.Printf("Debug: Process not running: %v\n", err)
		os.Remove(pidFile)
		return false
	}

	return true
}
