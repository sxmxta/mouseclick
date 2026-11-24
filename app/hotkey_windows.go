//go:build windows

package app

import "github.com/energye/lcl/pkgs/win"

// registerHotKey 绑定全局热键到窗体句柄
func registerHotKey(hwnd uintptr, vk, mod uint32) bool {
	return win.RegisterHotKey(hwnd, hotkeyID, mod|modNoRepeat, vk)
}

func unregisterHotKey(hwnd uintptr, id int32) {
	win.UnregisterHotKey(hwnd, id)
}
