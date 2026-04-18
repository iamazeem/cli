//go:build windows

package telemetry

import (
	"syscall"

	"golang.org/x/sys/windows"
)

// detachAttrs returns SysProcAttr configured to place the child in its own
// process group so that console signals (Ctrl+C) delivered to the parent's
// group are not forwarded to the child.
func detachAttrs() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | windows.DETACHED_PROCESS}
}
