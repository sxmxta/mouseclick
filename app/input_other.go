//go:build !windows

package app

// 非 Windows 平台不支持鼠标注入，保留接口便于跨平台编译
func mouseDown(b clickButton) {}

func mouseUp(b clickButton) {}

func setCursorPos(x, y int32) {}

func mouseWheel(delta int32) {}

func escapePressed() bool { return false }

func enableHighResTimer() {}

func disableHighResTimer() {}
