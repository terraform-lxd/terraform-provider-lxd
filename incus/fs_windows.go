//go:build windows

package incus

// IsSocketWritable always returns true when os is windows.
func IsSocketWritable(socketPath string) bool {
	return true
}
