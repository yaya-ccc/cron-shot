package sys_utils

import "syscall"

var (
	procPrintWindow          = user32.NewProc("PrintWindow")
	procGetWindowDC          = user32.NewProc("GetWindowDC")
	user32                   = syscall.NewLazyDLL("user32.dll")
	procGetWindowTextLengthW = user32.NewProc("GetWindowTextLengthW")
	procGetWindowTextW       = user32.NewProc("GetWindowTextW")
	procEnumWindows          = user32.NewProc("EnumWindows")
)
