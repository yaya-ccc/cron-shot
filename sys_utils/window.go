package sys_utils

import (
	"syscall"
	"unsafe"

	"github.com/lxn/win"
)

// GetProcessWindows 获取指定进程打开的所有可见窗口标题
func GetProcessWindows(processName string) ([]string, error) {
	// 合并逻辑：复用详细枚举结果，仅提取标题
	infos, err := GetProcessWindowsDetailed(processName)
	if err != nil {
		return nil, err
	}
	titles := make([]string, 0, len(infos))
	for _, info := range infos {
		titles = append(titles, info.Title)
	}
	return titles, nil
}

// getWindowTextLength 返回窗口标题长度
func getWindowTextLength(hwnd win.HWND) int32 {
	ret, _, _ := procGetWindowTextLengthW.Call(uintptr(hwnd))
	return int32(ret)
}

// getWindowText 将窗口标题写入缓冲区
func getWindowText(hwnd win.HWND, str *uint16, maxCount int32) int32 {
	ret, _, _ := procGetWindowTextW.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(str)),
		uintptr(maxCount),
	)
	return int32(ret)
}

// enumWindows 枚举所有顶级窗口并调用回调
func enumWindows(lpEnumFunc uintptr, lParam uintptr) bool {
	ret, _, _ := procEnumWindows.Call(
		lpEnumFunc,
		lParam,
	)
	return ret != 0
}

func GetWindowTitleByHWND(hwnd win.HWND) string {
	tl := getWindowTextLength(hwnd)
	if tl <= 0 {
		return ""
	}
	buf := make([]uint16, tl+1)
	getWindowText(hwnd, &buf[0], int32(tl+1))
	return syscall.UTF16ToString(buf)
}
