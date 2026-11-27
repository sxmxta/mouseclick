//go:build windows

package app

import (
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

// 录制调试日志：只有设置了 MOUSECLICK_REC_DEBUG 环境变量才写文件
var recDebugOn = os.Getenv("MOUSECLICK_REC_DEBUG") != ""

func recLog(format string, args ...any) {
	if !recDebugOn {
		return
	}
	f, err := os.OpenFile(os.Getenv("MOUSECLICK_REC_DEBUG"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s %s\n", time.Now().Format("15:04:05.000"), fmt.Sprintf(format, args...))
}

// 低级鼠标钩子：录制时抓点击 / 拖拽 / 滚轮，不注入任何输入
const (
	whMouseLL    = 14
	wmLButtonDn  = 0x0201
	wmLButtonUp  = 0x0202
	wmRButtonDn  = 0x0204
	wmRButtonUp  = 0x0205
	wmMButtonDn  = 0x0207
	wmMButtonUp  = 0x0208
	wmMouseWheel = 0x020A
)

type hookPoint struct {
	x, y int32
}

// msllHook 对应 Win32 MSLLHOOKSTRUCT（分两次定义避免 32/64 位对齐问题）
type msllHook struct {
	pt        hookPoint
	mouseData uint32
	flags     uint32
	time      uint32
	extraInfo uintptr
}

var (
	procSetWindowsHookExW   = user32DLL.NewProc("SetWindowsHookExW")
	procUnhookWindowsHookEx = user32DLL.NewProc("UnhookWindowsHookEx")
	procCallNextHookEx      = user32DLL.NewProc("CallNextHookEx")
	procGetModuleHandleW    = syscall.NewLazyDLL("kernel32.dll").NewProc("GetModuleHandleW")
	procGetCurrentThreadId  = syscall.NewLazyDLL("kernel32.dll").NewProc("GetCurrentThreadId")
	procRtlMoveMemory       = syscall.NewLazyDLL("kernel32.dll").NewProc("RtlMoveMemory")
	procGetWindowRect       = user32DLL.NewProc("GetWindowRect")

	recActiveMu sync.Mutex
	recActive   *recorder

	hookMu     sync.Mutex
	hookHandle uintptr

	mouseHookCallback = syscall.NewCallback(func(nCode, wParam, lParam uintptr) uintptr {
		if int32(nCode) >= 0 {
			if r := currentRecorder(); r != nil && lParam != 0 {
				// 用 RtlMoveMemory 把系统结构体拷进 Go 变量，避免 uintptr 直接转指针
				var ev msllHook
				procRtlMoveMemory.Call(uintptr(unsafe.Pointer(&ev)), lParam, unsafe.Sizeof(ev))
				r.handleHookMessage(uint32(wParam), &ev)
			}
		}
		res, _, _ := procCallNextHookEx.Call(0, nCode, wParam, lParam)
		return res
	})
)

func currentRecorder() *recorder {
	recActiveMu.Lock()
	defer recActiveMu.Unlock()
	return recActive
}

func installRecorder(r *recorder) {
	recActiveMu.Lock()
	recActive = r
	recActiveMu.Unlock()
}

func uninstallRecorder() {
	recActiveMu.Lock()
	recActive = nil
	recActiveMu.Unlock()
}

// windowContains 判断屏幕坐标是否落在窗口矩形内（用于滤掉点「停止」那一下）
func windowContains(hwnd uintptr, x, y int32) bool {
	if hwnd == 0 {
		return false
	}
	var r struct {
		left, top, right, bottom int32
	}
	procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	return x >= r.left && x <= r.right && y >= r.top && y <= r.bottom
}

// handleHookMessage 把 Windows 消息翻译成录制事件
func (r *recorder) handleHookMessage(msg uint32, ev *msllHook) {
	if ev == nil {
		return
	}
	recLog("hook msg=0x%04x pt=(%d,%d)", msg, ev.pt.x, ev.pt.y)
	switch msg {
	case wmLButtonDn:
		r.onEvent(recEvDown, btnLeft, ev.pt.x, ev.pt.y, 0, ev.time)
	case wmLButtonUp:
		r.onEvent(recEvUp, btnLeft, ev.pt.x, ev.pt.y, 0, ev.time)
	case wmRButtonDn:
		r.onEvent(recEvDown, btnRight, ev.pt.x, ev.pt.y, 0, ev.time)
	case wmRButtonUp:
		r.onEvent(recEvUp, btnRight, ev.pt.x, ev.pt.y, 0, ev.time)
	case wmMButtonDn:
		r.onEvent(recEvDown, btnMiddle, ev.pt.x, ev.pt.y, 0, ev.time)
	case wmMButtonUp:
		r.onEvent(recEvUp, btnMiddle, ev.pt.x, ev.pt.y, 0, ev.time)
	case wmMouseWheel:
		delta := int(int16(uint16(ev.mouseData >> 16)))
		notches := delta / 120
		if notches == 0 {
			if delta > 0 {
				notches = 1
			} else {
				notches = -1
			}
		}
		r.onEvent(recEvWheel, btnLeft, ev.pt.x, ev.pt.y, notches, ev.time)
	}
}

// startMouseHook 在调用线程（界面线程）上装钩子：LCL 的主消息循环负责把
// 系统发来的钩子通知取出来，低级鼠标钩子就是这样触发的，不需要自己泵消息。
func startMouseHook() bool {
	hookMu.Lock()
	defer hookMu.Unlock()
	if hookHandle != 0 {
		return true
	}
	tid, _, _ := procGetCurrentThreadId.Call()
	mod, _, _ := procGetModuleHandleW.Call(0)
	h, _, callErr := procSetWindowsHookExW.Call(uintptr(whMouseLL), mouseHookCallback, mod, 0)
	recLog("hook install tid=%d mod=%d h=%d err=%v", tid, mod, h, callErr)
	if h == 0 {
		return false
	}
	hookHandle = h
	return true
}

func stopMouseHook() {
	hookMu.Lock()
	h := hookHandle
	hookHandle = 0
	hookMu.Unlock()
	if h == 0 {
		return
	}
	procUnhookWindowsHookEx.Call(h)
	recLog("hook removed h=%d", h)
}
