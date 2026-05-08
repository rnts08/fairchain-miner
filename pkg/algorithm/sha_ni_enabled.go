//go:build amd64 && sha_ni

package algorithm

// hasSHANI is true when the sha_ni build tag is provided.
// This bypasses issues with missing fields in the cpu package.
const hasSHANI = true