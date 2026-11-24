//go:build !windows

package app

// 非 Windows 平台不支持全局热键，保留接口便于跨平台编译
func registerHotKey(hwnd uintptr, vk, mod uint32) bool {
	return false
}

func unregisterHotKey(hwnd uintptr, id int32) {
}
