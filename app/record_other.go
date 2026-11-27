//go:build !windows

package app

// 非 Windows 平台不支持鼠标录制，保留接口便于跨平台编译
func installRecorder(r *recorder) {}

func uninstallRecorder() {}

func startMouseHook() bool { return false }

func stopMouseHook() {}

func windowContains(hwnd uintptr, x, y int32) bool { return false }
