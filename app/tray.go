package app

import (
	"github.com/energye/lcl/lcl"
	"github.com/energye/lcl/types"
)

// initTray 创建托盘图标和右键菜单。托盘在窗口显示/隐藏期间一直存在。
func (m *TForm1) initTray() {
	if m.trayIcon != nil {
		return
	}

	m.trayMenu = lcl.NewPopupMenu(m)
	m.trayToggleItem = lcl.NewMenuItem(m)
	m.trayToggleItem.SetOnClick(func(lcl.IObject) {
		if m.Visible() {
			m.hideMainWindow()
		} else {
			m.showMainWindow()
		}
	})

	exitItem := lcl.NewMenuItem(m)
	exitItem.SetCaption("退出程序")
	exitItem.SetOnClick(func(lcl.IObject) {
		m.exitApplication()
	})

	m.trayMenu.Items().Add(m.trayToggleItem)
	separator := lcl.NewMenuItem(m)
	separator.SetCaption("-")
	m.trayMenu.Items().Add(separator)
	m.trayMenu.Items().Add(exitItem)

	m.trayIcon = lcl.NewTrayIcon(m)
	m.trayIcon.SetHint("鼠标连点器")
	m.trayIcon.SetPopUpMenu(m.trayMenu)
	if icon := lcl.Application.Icon(); icon != nil {
		m.trayIcon.SetIcon(icon)
	} else if icon := m.Icon(); icon != nil {
		m.trayIcon.SetIcon(icon)
	}
	m.trayIcon.SetOnDblClick(func(lcl.IObject) {
		m.showMainWindow()
	})
	m.trayIcon.SetVisible(true)
	m.updateTrayMenu()
}

// updateTrayMenu 根据主窗口当前可见状态切换“显示窗口 / 隐藏窗口”菜单项。
func (m *TForm1) updateTrayMenu() {
	if m.trayToggleItem == nil {
		return
	}
	if m.Visible() {
		m.trayToggleItem.SetCaption("隐藏窗口")
	} else {
		m.trayToggleItem.SetCaption("显示窗口")
	}
}

// showMainWindow 从托盘恢复主窗口并切到前台。
func (m *TForm1) showMainWindow() {
	m.Show()
	m.SetWindowState(types.WsNormal)
	m.ShowOnTop()
	m.updateTrayMenu()
}

// hideMainWindow 隐藏主窗口，但不停止连点引擎；全局热键仍可用。
func (m *TForm1) hideMainWindow() {
	m.Hide()
	m.updateTrayMenu()
}

// releaseRuntime 退出前的统一清理：落盘配置、停止连点、注销热键、移除托盘图标。
// 重复调用（例如托盘退出后 Close 再次触发 OnCloseQuery）不会重复清理。
func (m *TForm1) releaseRuntime() {
	if m.runtimeReleased {
		return
	}
	m.runtimeReleased = true
	m.cfg.flush()
	m.engine.stop()
	m.run.stop()
	if m.recording {
		m.recording = false
		uninstallRecorder()
		stopMouseHook()
		m.recTimer.SetEnabled(false)
	}
	if m.hotkeySet {
		unregisterHotKey(uintptr(m.Handle()), hotkeyID)
		m.hotkeySet = false
	}
	if m.recordHotkeySet {
		unregisterHotKey(uintptr(m.Handle()), recordHotkeyID)
		m.recordHotkeySet = false
	}
	if m.trayIcon != nil {
		m.trayIcon.SetVisible(false)
	}
}

// exitApplication 托盘“退出程序”：绕过关闭隐藏分支，直接结束进程。
func (m *TForm1) exitApplication() {
	m.forceExit = true
	m.releaseRuntime()
	m.Close()
}
