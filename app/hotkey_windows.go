//go:build windows

package app

import "github.com/energye/lcl/pkgs/win"

// registerHotKey 绑定全局热键到窗体句柄；id 用来区分「启停热键」和「录制热键」
func registerHotKey(hwnd uintptr, id int32, vk, mod uint32) bool {
	return win.RegisterHotKey(hwnd, id, mod|modNoRepeat, vk)
}

func unregisterHotKey(hwnd uintptr, id int32) {
	win.UnregisterHotKey(hwnd, id)
}
