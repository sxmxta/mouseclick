// ==============================================================================
// Event handling and business logic can be added to this file
// Which is automatically created when it does not exist
// ==============================================================================

package app

import (
	"fmt"

	"github.com/energye/lcl/api"
	"github.com/energye/lcl/lcl"
	"github.com/energye/lcl/types"
	"github.com/energye/lcl/types/colors"
	"github.com/energye/lcl/types/keys"
	"github.com/energye/lcl/types/messages"
)

const (
	hotkeyID              = 1
	modNoRepeat           = 0x4000 // MOD_NOREPEAT，按住热键时不重复触发
	defaultRate           = 20     // 默认 20 次/秒
	minRate               = 1
	maxRate               = 50
	defaultDoubleGap      = 100 // 默认双击两次点击间隔 100 ms
	defaultDoubleInterval = 300 // 默认每组两次点击后等待 300 ms
	lockCaptionOn         = "锁定点击位置：已锁定 (%d, %d)"
	lockCaptionOff        = "锁定点击位置（勾选时固定为当前鼠标位置）"
	githubURL             = "https://github.com/sxmxta/mouseclick"
)

type hotkeyOption struct {
	name string
	mod  uint32
	vk   uint32
}

var hotkeyOptions = []hotkeyOption{
	{"Ctrl+F8", modControl, uint32(keys.VkF8)},
	{"Ctrl+F9", modControl, uint32(keys.VkF9)},
	{"Ctrl+F10", modControl, uint32(keys.VkF10)},
	{"Alt+F8", modAlt, uint32(keys.VkF8)},
	{"Alt+F9", modAlt, uint32(keys.VkF9)},
	{"F8", 0, uint32(keys.VkF8)},
	{"F9", 0, uint32(keys.VkF9)},
	{"F10", 0, uint32(keys.VkF10)},
	{"F11", 0, uint32(keys.VkF11)},
	{"F12", 0, uint32(keys.VkF12)},
}

// defaultHotkeyIndex 默认 F8；游戏技能栏占用 F8 时可在下拉里切换 Ctrl+F8 / Alt+F8 等
const defaultHotkeyIndex = 5

type TForm1 struct {
	TForm1UI

	engine clicker

	hotkeyVK   uint32
	hotkeyMod  uint32
	hotkeyName string
	hotkeySet  bool
	shown      bool

	trayIcon        lcl.ITrayIcon
	trayMenu        lcl.IPopupMenu
	trayToggleItem  lcl.IMenuItem
	forceExit       bool // 托盘“退出程序”时绕过关闭隐藏分支
	runtimeReleased bool // 退出清理只执行一次

	suppressHotkeyChange bool
	loadingConfig        bool
	cfg                  configStore
	lastButtonText       string
	lastCount            int64
}

// OnFormCreate 窗体初始化事件
func (m *TForm1) OnFormCreate(sender lcl.IObject) {
	m.SetBorderStyleToFormBorderStyle(types.BsSingle)
	m.SetPosition(types.PoScreenCenter)
	m.EnabledMaximize(false)
	m.SetColor(colors.RGBToColor(246, 247, 250))
	m.Font().SetName("微软雅黑")
	m.Font().SetSize(9)

	m.applyStyle()
	m.initHotkey()
	m.engine.init(defaultRate)
	m.engine.setButton(btnLeft)
	m.applyConfig(loadAppConfig())

	m.RadioLeft.SetOnClick(func(lcl.IObject) { m.syncButton(); m.rememberButton() })
	m.RadioMiddle.SetOnClick(func(lcl.IObject) { m.syncButton(); m.rememberButton() })
	m.RadioRight.SetOnClick(func(lcl.IObject) { m.syncButton(); m.rememberButton() })

	m.RadioSingle.SetOnClick(func(lcl.IObject) { m.setMode(modeSingle) })
	m.RadioDouble.SetOnClick(func(lcl.IObject) { m.setMode(modeDouble) })

	m.TrackBarSpeed.SetOnChange(func(lcl.IObject) {
		rate := int(m.TrackBarSpeed.Position())
		if rate < minRate {
			rate = minRate
		}
		if rate > maxRate {
			rate = maxRate
		}
		m.engine.setRate(rate)
		m.updateTimingLabels()
		m.rememberRate(rate)
	})

	m.TrackBarDoubleGap.SetOnChange(func(lcl.IObject) {
		gap := int(m.TrackBarDoubleGap.Position())
		if gap < minDoubleGap {
			gap = minDoubleGap
		}
		if gap > maxDoubleGap {
			gap = maxDoubleGap
		}
		m.engine.setDoubleGap(gap)
		m.updateTimingLabels()
		if !m.loadingConfig {
			m.cfg.update(func(c *appConfig) { c.DoubleGap = gap })
		}
	})

	m.TrackBarDoubleInterval.SetOnChange(func(lcl.IObject) {
		interval := int(m.TrackBarDoubleInterval.Position())
		if interval < minDoubleInterval {
			interval = minDoubleInterval
		}
		if interval > maxDoubleInterval {
			interval = maxDoubleInterval
		}
		m.engine.setDoubleInterval(interval)
		m.updateTimingLabels()
		if !m.loadingConfig {
			m.cfg.update(func(c *appConfig) { c.DoubleInterval = interval })
		}
	})

	m.CheckBoxLock.SetOnClick(func(lcl.IObject) {
		if m.CheckBoxLock.Checked() {
			pos := lcl.Mouse.CursorPos()
			m.engine.setFixed(true, pos.X, pos.Y)
			m.CheckBoxLock.SetCaption(fmt.Sprintf(lockCaptionOn, pos.X, pos.Y))
			m.cfg.update(func(c *appConfig) {
				c.Lock = true
				c.LockX, c.LockY = pos.X, pos.Y
			})
		} else {
			m.engine.setFixed(false, 0, 0)
			m.CheckBoxLock.SetCaption(lockCaptionOff)
			m.cfg.update(func(c *appConfig) {
				c.Lock = false
				c.LockX, c.LockY = 0, 0
			})
		}
	})

	m.CheckBoxCloseToTray.SetOnClick(func(lcl.IObject) {
		m.cfg.update(func(c *appConfig) {
			c.CloseToTray = m.CheckBoxCloseToTray.Checked()
		})
	})

	m.LinkGitHub.SetOnLinkClick(func(sender lcl.IObject, link string, linkType types.TSysLinkType) {
		if linkType == types.SltURL {
			api.SysOpen(link)
		}
	})
	m.LinkGitHub.SetCaption("<a href=\"" + githubURL + "\">Github</a>")
	m.LinkGitHub.SetHint(githubURL)
	m.LinkGitHub.SetShowHint(true)

	m.ButtonToggle.SetOnClick(func(lcl.IObject) { m.toggle() })

	m.TimerUI.SetOnTimer(func(lcl.IObject) { m.refreshUI() })

	m.SetOnWndProc(func(theMessage *types.TLMessage) {
		if theMessage.Msg == messages.WM_HOTKEY && int32(theMessage.WParam) == hotkeyID {
			m.toggle()
		}
		m.InheritedWndProc(theMessage)
	})

	m.lastButtonText = m.ButtonToggle.Caption()
	m.refreshUI()
}

// OnShow 窗体显示事件
func (m *TForm1) OnShow(sender lcl.IObject) {
	if m.shown {
		return
	}
	m.shown = true
	m.initTray()
	if !m.applyHotkey(m.hotkeyVK, m.hotkeyMod, m.hotkeyName) {
		m.showHint("热键 "+m.hotkeyName+" 注册失败，可能被其他程序占用，请换一个", true)
	}
}

// OnCloseQuery 窗口关闭事件：勾选“关闭隐藏窗口”时只隐藏，否则真正退出。
// 托盘菜单的退出会先设置 forceExit，因此不会被这里的隐藏逻辑拦住。
func (m *TForm1) OnCloseQuery(sender lcl.IObject, canClose *bool) {
	if !m.forceExit && m.CheckBoxCloseToTray.Checked() {
		*canClose = false
		m.cfg.flush()
		m.hideMainWindow()
		return
	}
	m.releaseRuntime()
	*canClose = true
}

func (m *TForm1) applyStyle() {
	m.LabelStatus.SetParentFont(false)
	m.LabelStatus.Font().SetSize(13)
	m.LabelStatus.Font().SetBold(true)
	m.LabelStatus.Font().SetColor(statusColor(stateStopped))

	m.LabelCount.SetParentFont(false)

	m.LabelKey.SetParentFont(false)
	m.LabelKey.Font().SetColor(colors.RGBToColor(70, 74, 80))

	m.LabelMode.SetParentFont(false)
	m.LabelMode.Font().SetColor(colors.RGBToColor(70, 74, 80))

	m.LabelSpeed.SetParentFont(false)
	m.LabelSpeed.Font().SetColor(colors.RGBToColor(70, 74, 80))

	m.ButtonToggle.SetParentFont(false)
	m.ButtonToggle.Font().SetSize(12)
	m.ButtonToggle.Font().SetBold(true)

	m.LabelDoubleGap.SetParentFont(false)
	m.LabelDoubleGap.Font().SetColor(colors.RGBToColor(70, 74, 80))

	m.LabelDoubleInterval.SetParentFont(false)
	m.LabelDoubleInterval.Font().SetColor(colors.RGBToColor(70, 74, 80))

	m.LabelHint.SetParentFont(false)
	m.LinkGitHub.SetParentFont(false)
	m.LinkGitHub.Font().SetColor(colors.RGBToColor(0, 102, 204))
}

func (m *TForm1) initHotkey() {
	opt := hotkeyOptions[defaultHotkeyIndex]
	m.hotkeyName = opt.name
	m.hotkeyVK = opt.vk
	m.hotkeyMod = opt.mod

	for _, item := range hotkeyOptions {
		m.ComboBoxHotkey.Items().Add(item.name)
	}
	m.suppressHotkeyChange = true
	m.ComboBoxHotkey.SetItemIndex(defaultHotkeyIndex)
	m.suppressHotkeyChange = false
	m.ComboBoxHotkey.SetOnChange(func(lcl.IObject) { m.onHotkeyChanged() })
}

// applyConfig 把持久化配置应用到界面和引擎；调用时事件处理器尚未绑定
func (m *TForm1) applyConfig(cfg appConfig) {
	m.loadingConfig = true
	defer func() { m.loadingConfig = false }()

	cfg = cfg.normalized()
	m.TrackBarSpeed.SetPosition(int32(cfg.Rate))
	m.engine.setRate(cfg.Rate)

	m.RadioSingle.SetChecked(cfg.Mode != modeDouble)
	m.RadioDouble.SetChecked(cfg.Mode == modeDouble)
	m.TrackBarDoubleGap.SetPosition(int32(cfg.DoubleGap))
	m.TrackBarDoubleInterval.SetPosition(int32(cfg.DoubleInterval))
	m.engine.setDouble(cfg.Mode == modeDouble)
	m.engine.setDoubleGap(cfg.DoubleGap)
	m.engine.setDoubleInterval(cfg.DoubleInterval)

	btn := parseButton(cfg.Button)
	m.engine.setButton(btn)
	m.RadioLeft.SetChecked(btn == btnLeft)
	m.RadioMiddle.SetChecked(btn == btnMiddle)
	m.RadioRight.SetChecked(btn == btnRight)

	idx := hotkeyIndex(cfg.Hotkey)
	m.suppressHotkeyChange = true
	m.ComboBoxHotkey.SetItemIndex(int32(idx))
	m.suppressHotkeyChange = false
	opt := hotkeyOptions[idx]
	m.hotkeyVK, m.hotkeyMod, m.hotkeyName = opt.vk, opt.mod, opt.name

	if cfg.Lock {
		m.CheckBoxLock.SetChecked(true)
		m.engine.setFixed(true, cfg.LockX, cfg.LockY)
		m.CheckBoxLock.SetCaption(fmt.Sprintf(lockCaptionOn, cfg.LockX, cfg.LockY))
	} else {
		m.CheckBoxLock.SetChecked(false)
		m.engine.setFixed(false, 0, 0)
		m.CheckBoxLock.SetCaption(lockCaptionOff)
	}
	m.CheckBoxCloseToTray.SetChecked(cfg.CloseToTray)
	m.applyModeUI()
	m.cfg.set(cfg)
}

func (m *TForm1) isDoubleMode() bool {
	return m.RadioDouble.Checked()
}

// applyModeUI 按当前模式切换配置面板，并调整公共控件的位置和窗口高度
func (m *TForm1) applyModeUI() {
	double := m.isDoubleMode()
	m.LabelSpeed.SetVisible(!double)
	m.TrackBarSpeed.SetVisible(!double)
	m.LabelDoubleGap.SetVisible(double)
	m.TrackBarDoubleGap.SetVisible(double)
	m.LabelDoubleInterval.SetVisible(double)
	m.TrackBarDoubleInterval.SetVisible(double)

	var delta int32
	if !double {
		delta = -56 // 单击面板更矮，公共控件整体上移
	}
	m.LabelHotkey.SetTop(239 + delta)
	m.ComboBoxHotkey.SetTop(235 + delta)
	m.CheckBoxLock.SetTop(270 + delta)
	m.ButtonToggle.SetTop(310 + delta)
	m.LabelHint.SetTop(366 + delta)
	m.LinkGitHub.SetTop(392 + delta)
	if double {
		m.SetHeight(424)
	} else {
		m.SetHeight(368)
	}
	m.updateTimingLabels()
}

// setMode 切换单击/双击模式；两种模式的参数各自保留
func (m *TForm1) setMode(mode string) {
	if mode == modeDouble {
		m.RadioDouble.SetChecked(true)
	} else {
		m.RadioSingle.SetChecked(true)
	}
	m.engine.setDouble(mode == modeDouble)
	m.applyModeUI()
	if !m.loadingConfig {
		m.cfg.update(func(c *appConfig) { c.Mode = mode })
	}
}

func (m *TForm1) updateTimingLabels() {
	rate := int(m.TrackBarSpeed.Position())
	gap := int(m.TrackBarDoubleGap.Position())
	interval := int(m.TrackBarDoubleInterval.Position())
	m.LabelSpeed.SetCaption(rateText(rate))
	m.LabelDoubleGap.SetCaption(fmt.Sprintf("双击间隔：%d ms", gap))
	m.LabelDoubleInterval.SetCaption(fmt.Sprintf("组间隔：%d ms（每组两次后等待）", interval))
}

func (m *TForm1) selectedButton() clickButton {
	switch {
	case m.RadioMiddle.Checked():
		return btnMiddle
	case m.RadioRight.Checked():
		return btnRight
	default:
		return btnLeft
	}
}

func (m *TForm1) rememberButton() {
	if m.loadingConfig {
		return
	}
	b := m.selectedButton()
	m.cfg.update(func(c *appConfig) { c.Button = buttonName(b) })
}

func (m *TForm1) rememberRate(rate int) {
	if m.loadingConfig {
		return
	}
	m.cfg.update(func(c *appConfig) { c.Rate = rate })
}

func (m *TForm1) syncButton() {
	m.engine.setButton(m.selectedButton())
}

func (m *TForm1) toggle() {
	if m.engine.state() == stateStopped {
		m.engine.start()
	} else {
		m.engine.stop()
	}
	m.refreshUI()
}

func (m *TForm1) onHotkeyChanged() {
	if m.suppressHotkeyChange {
		return
	}
	idx := int(m.ComboBoxHotkey.ItemIndex())
	if idx < 0 || idx >= len(hotkeyOptions) {
		return
	}
	oldName, oldVK, oldMod := m.hotkeyName, m.hotkeyVK, m.hotkeyMod
	opt := hotkeyOptions[idx]
	if m.applyHotkey(opt.vk, opt.mod, opt.name) {
		m.cfg.update(func(c *appConfig) { c.Hotkey = opt.name })
		return
	}
	// 注册失败：回滚下拉框并恢复原热键
	m.suppressHotkeyChange = true
	for i, item := range hotkeyOptions {
		if item.name == oldName {
			m.ComboBoxHotkey.SetItemIndex(int32(i))
			break
		}
	}
	m.suppressHotkeyChange = false
	m.applyHotkey(oldVK, oldMod, oldName)
	m.showHint("热键 "+opt.name+" 注册失败，可能被其他程序占用", true)
}

func (m *TForm1) applyHotkey(vk, mod uint32, name string) bool {
	if m.hotkeySet {
		unregisterHotKey(uintptr(m.Handle()), hotkeyID)
		m.hotkeySet = false
	}
	if registerHotKey(uintptr(m.Handle()), vk, mod) {
		m.hotkeySet = true
		m.hotkeyVK = vk
		m.hotkeyMod = mod
		m.hotkeyName = name
		m.updateHotkeyTexts()
		return true
	}
	return false
}

func (m *TForm1) updateHotkeyTexts() {
	if m.engine.state() == stateStopped {
		m.ButtonToggle.SetCaption("开始 (" + m.hotkeyName + ")")
		m.lastButtonText = m.ButtonToggle.Caption()
	}
	m.showHint("把鼠标放到指定位置上，按 "+m.hotkeyName+" 开始 / 再按一次暂停", false)
}

func (m *TForm1) showHint(text string, isError bool) {
	m.LabelHint.SetCaption(text)
	if isError {
		m.LabelHint.Font().SetColor(colors.RGBToColor(200, 60, 60))
	}
}

func (m *TForm1) refreshUI() {
	state := m.engine.state()
	var text, btnText string
	switch state {
	case stateRunning:
		text = "● 运行中"
		btnText = "暂停 (" + m.hotkeyName + ")"
	case stateArming:
		text = "● 即将开始…"
		btnText = "取消 (" + m.hotkeyName + ")"
	default:
		text = "● 已暂停"
		btnText = "开始 (" + m.hotkeyName + ")"
	}
	if m.LabelStatus.Caption() != text {
		m.LabelStatus.SetCaption(text)
		m.LabelStatus.Font().SetColor(statusColor(state))
	}
	if btnText != m.lastButtonText {
		m.ButtonToggle.SetCaption(btnText)
		m.lastButtonText = btnText
	}
	if n := m.engine.clicks(); n != m.lastCount {
		m.lastCount = n
		m.LabelCount.SetCaption(fmt.Sprintf("已点击 %d 次", n))
	}
}

func statusColor(state runState) types.TColor {
	switch state {
	case stateRunning:
		return colors.RGBToColor(34, 160, 82)
	case stateArming:
		return colors.RGBToColor(230, 140, 30)
	default:
		return colors.RGBToColor(120, 124, 130)
	}
}

func rateText(rate int) string {
	return fmt.Sprintf("点击速度：%d 次/秒（间隔 %d ms）", rate, 1000/rate)
}
