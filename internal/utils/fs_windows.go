//go:build windows

package utils

// IsSocketWritable always returns true when os is windows.
func IsSocketWritable(socketPath string) bool {
	return true
}
