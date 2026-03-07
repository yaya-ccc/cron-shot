package sys_utils

import (
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"cron-shot/logging"

	"github.com/lxn/win"
)

type WindowInfo struct {
	Title string
	HWND  win.HWND
}

// 说明：使用单例 EnumWindows 回调并通过包级上下文传参，避免频繁 syscall.NewCallback 导致崩溃
var (
	psapi                  = syscall.NewLazyDLL("psapi.dll")
	procGetModuleBaseNameW = psapi.NewProc("GetModuleBaseNameW")
	enumOnceDetailed       sync.Once
	enumCBDetailed         uintptr
	enumMuDetailed         sync.Mutex
	enumTargetDetailed     string
	enumOutDetailed        *[]WindowInfo
)

// GetProcessWindowsDetailed 返回指定进程的可见窗口详细信息（标题与句柄）
// 先过滤不可见/无标题窗口，再解析进程名匹配，降低系统调用成本
func GetProcessWindowsDetailed(processName string) ([]WindowInfo, error) {
	out := make([]WindowInfo, 0)
	target := processName
	if !strings.HasSuffix(strings.ToLower(target), ".exe") {
		target += ".exe"
	}

	enumOnceDetailed.Do(func() {
		enumCBDetailed = syscall.NewCallback(enumCallbackDetailed)
	})

	enumMuDetailed.Lock()
	defer enumMuDetailed.Unlock()
	enumTargetDetailed = target
	enumOutDetailed = &out
	enumWindows(enumCBDetailed, 0)
	enumOutDetailed = nil
	enumTargetDetailed = ""
	return out, nil
}

// 枚举回调：过滤窗口可见性与标题后，匹配进程名并收集结果
func enumCallbackDetailed(hwnd win.HWND, lParam uintptr) uintptr {
	defer logging.RecoverPanic("enumCallbackDetailed")
	tl := getWindowTextLength(hwnd)
	if tl <= 0 {
		return 1
	}
	buf := make([]uint16, tl+1)
	getWindowText(hwnd, &buf[0], int32(tl+1))
	title := syscall.UTF16ToString(buf)
	if title == "" {
		return 1
	}
	var pid uint32
	win.GetWindowThreadProcessId(hwnd, &pid)
	const PROCESS_QUERY_INFORMATION = 0x0400
	const PROCESS_VM_READ = 0x0010
	hProcess, err := syscall.OpenProcess(PROCESS_QUERY_INFORMATION|PROCESS_VM_READ, false, pid)
	if err == nil && hProcess != 0 {
		defer syscall.CloseHandle(hProcess)
		var nameBuf [win.MAX_PATH]uint16
		n := uint32(len(nameBuf))
		ret, _, _ := procGetModuleBaseNameW.Call(
			uintptr(hProcess),
			0,
			uintptr(unsafe.Pointer(&nameBuf[0])),
			uintptr(n),
		)
		if ret != 0 {
			pName := syscall.UTF16ToString(nameBuf[:])
			if strings.EqualFold(pName, enumTargetDetailed) {
				if enumOutDetailed != nil {
					*enumOutDetailed = append(*enumOutDetailed, WindowInfo{Title: title, HWND: hwnd})
				}
			}
		}
	}
	return 1
}
