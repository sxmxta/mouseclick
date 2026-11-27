//go:build windows

package app

import (
	"sync"
	"syscall"
	"unsafe"

	"github.com/energye/lcl/pkgs/win"
)

// SendInput 鼠标事件标志（WinUser.h）
const (
	mouseEventLeftDown   = 0x0002
	mouseEventLeftUp     = 0x0004
	mouseEventRightDown  = 0x0008
	mouseEventRightUp    = 0x0010
	mouseEventMiddleDown = 0x0020
	mouseEventMiddleUp   = 0x0040
	mouseEventWheel      = 0x0800
	inputMouse           = 0
	vkEscape             = 0x1B
)

// mouseInput 对应 Win32 MOUSEINPUT，amd64 下大小为 32 字节
type mouseInput struct {
	dx        int32
	dy        int32
	mouseData uint32
	flags     uint32
	time      uint32
	extraInfo uintptr
}

// input 对应 Win32 INPUT。
// 不手工加对齐填充：Go 会按 mouseInput 的 uintptr 对齐自动补齐，
// 这样 386 下是 28 字节、amd64 下是 40 字节，避免 32 位构建结构体偏大。
type input struct {
	typ uint32
	mi  mouseInput
}

var (
	user32DLL     = syscall.NewLazyDLL("user32.dll")
	procSendInput = user32DLL.NewProc("SendInput")
	procKeyState  = user32DLL.NewProc("GetAsyncKeyState")

	winmmDLL            = syscall.NewLazyDLL("winmm.dll")
	procTimeBeginPeriod = winmmDLL.NewProc("timeBeginPeriod")
	procTimeEndPeriod   = winmmDLL.NewProc("timeEndPeriod")

	timerMu     sync.Mutex
	timerRefCnt int
)

func sendMouse(flags uint32) {
	sendMouseData(flags, 0)
}

func sendMouseData(flags uint32, data uint32) {
	var in input
	in.typ = inputMouse
	in.mi.flags = flags
	in.mi.mouseData = data
	// 失败（例如被 UIPI 拦截）时静默忽略：连点器不适合弹窗打断
	_, _, _ = procSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))
}

func mouseDown(b clickButton) {
	switch b {
	case btnMiddle:
		sendMouse(mouseEventMiddleDown)
	case btnRight:
		sendMouse(mouseEventRightDown)
	default:
		sendMouse(mouseEventLeftDown)
	}
}

func mouseUp(b clickButton) {
	switch b {
	case btnMiddle:
		sendMouse(mouseEventMiddleUp)
	case btnRight:
		sendMouse(mouseEventRightUp)
	default:
		sendMouse(mouseEventLeftUp)
	}
}

func setCursorPos(x, y int32) {
	win.SetCursorPos(x, y)
}

// mouseWheel 发送滚轮事件，delta 为 120 的整数倍（正数向上）
func mouseWheel(delta int32) {
	if delta == 0 {
		return
	}
	sendMouseData(mouseEventWheel, uint32(delta))
}

// escapePressed 查询 Esc 是否按下。轮询而非注册热键，不抢占 Esc
func escapePressed() bool {
	r, _, _ := procKeyState.Call(uintptr(vkEscape))
	return uint16(r)&0x8000 != 0
}

// enableHighResTimer 把系统定时器精度提到 1ms，让连点间隔更准
func enableHighResTimer() {
	timerMu.Lock()
	defer timerMu.Unlock()
	if timerRefCnt == 0 {
		if err := procTimeBeginPeriod.Find(); err == nil {
			_, _, _ = procTimeBeginPeriod.Call(1)
		}
	}
	timerRefCnt++
}

func disableHighResTimer() {
	timerMu.Lock()
	defer timerMu.Unlock()
	if timerRefCnt > 0 {
		timerRefCnt--
		if timerRefCnt == 0 {
			if err := procTimeEndPeriod.Find(); err == nil {
				_, _, _ = procTimeEndPeriod.Call(1)
			}
		}
	}
}
