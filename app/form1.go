// ==============================================================================
// Event handling and business logic can be added to this file
// Which is automatically created when it does not exist
// ==============================================================================

package app

import (
	"fmt"
	"time"

	"github.com/energye/lcl/api"
	"github.com/energye/lcl/lcl"
	"github.com/energye/lcl/types"
	"github.com/energye/lcl/types/colors"
	"github.com/energye/lcl/types/keys"
	"github.com/energye/lcl/types/messages"
)

const (
	hotkeyID              = 1
	recordHotkeyID        = 2 // 录制热键，与启停热键分开注册
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

// defaultRecordHotkeyIndex 默认 F10：按一下开始录，再按一下停
const defaultRecordHotkeyIndex = 7

type TForm1 struct {
	TForm1UI

	engine clicker

	hotkeyVK   uint32
	hotkeyMod  uint32
	hotkeyName string
	hotkeySet  bool

	recordHotkeyVK   uint32
	recordHotkeyMod  uint32
	recordHotkeyName string
	recordHotkeySet  bool

	shown bool

	trayIcon        lcl.ITrayIcon
	trayMenu        lcl.IPopupMenu
	trayToggleItem  lcl.IMenuItem
	forceExit       bool // 托盘“退出程序”时绕过关闭隐藏分支
	runtimeReleased bool // 退出清理只执行一次

	suppressHotkeyChange bool
	suppressRecKeyChange bool
	loadingConfig        bool
	cfg                  configStore
	lastButtonText       string
	lastCount            int64

	// 脚本（序列）模式
	RadioScript  lcl.IRadioButton
	PanelSeq     lcl.IPanel
	ListSteps    lcl.IListBox
	EditX        lcl.IEdit
	EditY        lcl.IEdit
	EditX2       lcl.IEdit
	EditY2       lcl.IEdit
	EditParam    lcl.IEdit
	EditGap      lcl.IEdit
	EditRepeat   lcl.IEdit
	ComboAct     lcl.IComboBox
	ComboBtn     lcl.IComboBox
	ComboLoop    lcl.IComboBox
	ComboRecKey  lcl.IComboBox
	LabelRecKey  lcl.ILabel
	LabelRecTip  lcl.ILabel
	EditLoopCnt  lcl.IEdit
	EditMaxDur   lcl.IEdit
	EditMaxClick lcl.IEdit
	EditDelay    lcl.IEdit
	BtnStepAdd   lcl.IButton
	BtnStepApply lcl.IButton
	BtnStepDel   lcl.IButton
	BtnStepUp    lcl.IButton
	BtnStepDown  lcl.IButton
	BtnRecord    lcl.IButton
	BtnPickFrom  lcl.IButton
	BtnPickTo    lcl.IButton

	// 说明与分组标题（随动作显隐/改写）
	LabelSeqHead  lcl.ILabel // 模式说明（脚本模式顶部）
	LabelStepHead lcl.ILabel // ① 步骤
	LabelActHead  lcl.ILabel // ② 动作
	LabelActDesc  lcl.ILabel // 当前动作的一句话说明
	LabelPosStart lcl.ILabel
	LabelPosX1    lcl.ILabel
	LabelPosY1    lcl.ILabel
	LabelPosEnd   lcl.ILabel
	LabelPosX2    lcl.ILabel
	LabelPosY2    lcl.ILabel
	LabelBtn      lcl.ILabel
	LabelParam    lcl.ILabel
	LabelParamTip lcl.ILabel
	LabelGap      lcl.ILabel
	LabelGapMs    lcl.ILabel
	LabelRepeat   lcl.ILabel
	LabelRunHead  lcl.ILabel // ③ 运行
	LabelLoop     lcl.ILabel
	LabelLoopCnt  lcl.ILabel
	LabelMaxDur   lcl.ILabel
	LabelMaxSec   lcl.ILabel
	LabelMaxClick lcl.ILabel
	LabelDelay    lcl.ILabel
	LabelDelayTip lcl.ILabel

	LabelSeqTip lcl.ILabel
	pickTimer   lcl.ITimer
	pickLeft    int
	pickTarget  int
	editingStep int32
	seq         []seqStep
	seqCfg      seqConfig
	run         runner

	// 录制
	rec          recorder
	recTimer     lcl.ITimer
	recLeft      int
	recording    bool
	recCount     int32
	hintHoldTill time.Time
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
	m.initSequenceUI()
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
		if theMessage.Msg == messages.WM_HOTKEY {
			recLog("wm_hotkey id=%d", int32(theMessage.WParam))
			switch int32(theMessage.WParam) {
			case hotkeyID:
				m.onHotkeyPress()
			case recordHotkeyID:
				m.onRecordHotkey()
			}
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
	if !m.applyRecordHotkey(m.recordHotkeyVK, m.recordHotkeyMod, m.recordHotkeyName) {
		m.showHint("录制热键 "+m.recordHotkeyName+" 注册失败，可能被其他程序占用，请在脚本模式里换一个", true)
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

	m.LabelHotkey.SetParentFont(false)
	m.LabelHotkey.Font().SetColor(colors.RGBToColor(70, 74, 80))

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

	m.RadioSingle.SetChecked(cfg.Mode == modeSingle)
	m.RadioDouble.SetChecked(cfg.Mode == modeDouble)
	if m.RadioScript != nil {
		m.RadioScript.SetChecked(cfg.Mode == modeScript)
	}
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

	recIdx := recordHotkeyIndex(cfg.RecordHotkey)
	if m.ComboRecKey != nil {
		m.suppressRecKeyChange = true
		m.ComboRecKey.SetItemIndex(int32(recIdx))
		m.suppressRecKeyChange = false
	}
	recOpt := hotkeyOptions[recIdx]
	m.recordHotkeyVK, m.recordHotkeyMod, m.recordHotkeyName = recOpt.vk, recOpt.mod, recOpt.name

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
	m.applySeqConfig(cfg.Sequence)
	m.applyModeUI()
	m.cfg.set(cfg)
}

func (m *TForm1) isDoubleMode() bool {
	return m.RadioDouble.Checked()
}

func (m *TForm1) isScriptMode() bool {
	return m.RadioScript != nil && m.RadioScript.Checked()
}

// applyModeUI 按当前模式切换配置面板，并调整公共控件的位置和窗口高度
func (m *TForm1) applyModeUI() {
	script := m.isScriptMode()
	double := m.isDoubleMode()
	m.lastCount = -1 // 强制刷新计数文字（脚本模式显示"已执行"）

	// 常规参数控件：脚本模式整体隐藏
	m.LabelKey.SetVisible(!script)
	m.PanelKey.SetVisible(!script)
	m.LabelHotkey.SetVisible(!script)
	m.ComboBoxHotkey.SetVisible(!script)
	m.CheckBoxLock.SetVisible(!script)
	m.LinkGitHub.SetVisible(!script)

	m.LabelSpeed.SetVisible(!script && !double)
	m.TrackBarSpeed.SetVisible(!script && !double)
	m.LabelDoubleGap.SetVisible(!script && double)
	m.TrackBarDoubleGap.SetVisible(!script && double)
	m.LabelDoubleInterval.SetVisible(!script && double)
	m.TrackBarDoubleInterval.SetVisible(!script && double)

	// 模式单选按钮始终可见
	m.LabelMode.SetVisible(true)
	m.RadioSingle.SetVisible(true)
	m.RadioDouble.SetVisible(true)
	if m.RadioScript != nil {
		m.RadioScript.SetVisible(true)
	}
	if m.PanelSeq != nil {
		m.PanelSeq.SetVisible(script)
	}

	if script {
		m.showSeqMode(true)
		m.updateTimingLabels()
		return
	}
	m.showSeqMode(false)

	var delta int32
	if !double {
		delta = -56 // 单击面板更矮，公共控件整体上移
	}
	m.LabelHotkey.SetTop(244 + delta)
	m.ComboBoxHotkey.SetTop(240 + delta)
	m.CheckBoxLock.SetTop(275 + delta)
	m.ButtonToggle.SetTop(315 + delta)
	m.LabelHint.SetTop(371 + delta)
	m.LinkGitHub.SetTop(397 + delta)
	if double {
		m.SetHeight(429)
		m.showHint("双击：两下为一组，组间等待 · "+m.hotkeyName+" 开始 / 再按暂停", false)
	} else {
		m.SetHeight(373)
		m.showHint("连点：在鼠标位置持续点击 · "+m.hotkeyName+" 开始 / 再按暂停", false)
	}
	m.updateTimingLabels()
}

// setMode 切换单击/双击/脚本模式；三种模式的参数各自保留
func (m *TForm1) setMode(mode string) {
	if mode == modeScript {
		m.engine.stop()
		m.run.stop()
		m.run.clearResult()
		m.RadioScript.SetChecked(true)
		m.RadioSingle.SetChecked(false)
		m.RadioDouble.SetChecked(false)
	} else {
		m.run.stop()
		m.run.clearResult()
		if mode == modeDouble {
			m.RadioDouble.SetChecked(true)
			m.RadioSingle.SetChecked(false)
		} else {
			m.RadioSingle.SetChecked(true)
			m.RadioDouble.SetChecked(false)
		}
		if m.RadioScript != nil {
			m.RadioScript.SetChecked(false)
		}
		m.engine.setDouble(mode == modeDouble)
	}
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
	m.LabelDoubleGap.SetCaption(fmt.Sprintf("双击间隔：%d ms（30–300，两次点击之间）", gap))
	m.LabelDoubleInterval.SetCaption(fmt.Sprintf("组间隔：%d ms（20–1000，两组之间）", interval))
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
	if m.isScriptMode() {
		if m.run.state() == stateStopped {
			cfg := m.readSeqConfigFromUI()
			if len(cfg.Steps) == 0 {
				m.showHint("请先添加至少一个步骤", true)
				return
			}
			m.persistSequence()
			if !m.run.start(cfg) {
				m.showHint("启动失败：序列为空", true)
				return
			}
		} else {
			m.run.stop()
		}
		m.refreshUI()
		return
	}
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
	opt := hotkeyOptions[idx]
	oldName, oldVK, oldMod := m.hotkeyName, m.hotkeyVK, m.hotkeyMod

	// 新启停键正好是当前录制键时，先把自己注册的录制键挪开；
	// 否则 RegisterHotKey 会被本进程的录制热键占住，直接失败。
	recMoved := false
	oldRecName, oldRecVK, oldRecMod := m.recordHotkeyName, m.recordHotkeyVK, m.recordHotkeyMod
	var movedRec hotkeyOption
	if opt.name == m.recordHotkeyName {
		movedRec = hotkeyOptions[recordHotkeyIndex(pickRecordHotkey(opt.name))]
		if !m.applyRecordHotkey(movedRec.vk, movedRec.mod, movedRec.name) {
			m.syncHotkeyCombo(oldName)
			m.showHint("换不了录制热键，启停热键保持 "+oldName+" 不变", true)
			return
		}
		recMoved = true
	}

	if m.applyHotkey(opt.vk, opt.mod, opt.name) {
		m.cfg.update(func(c *appConfig) { c.Hotkey = opt.name })
		if recMoved {
			m.syncRecKeyCombo(movedRec.name)
			m.cfg.update(func(c *appConfig) { c.RecordHotkey = movedRec.name })
			m.showHint("录制热键已改为 "+movedRec.name+"，避免和启停热键重复", false)
		}
		return
	}

	// 注册失败：回滚下拉框并恢复原热键；刚才挪过的录制键也恢复
	m.syncHotkeyCombo(oldName)
	m.applyHotkey(oldVK, oldMod, oldName)
	if recMoved {
		if m.applyRecordHotkey(oldRecVK, oldRecMod, oldRecName) {
			m.syncRecKeyCombo(oldRecName)
		} else {
			// 原键已被别的程序抢占，就保留刚才挪到的键，至少录制还能用
			m.applyRecordHotkey(movedRec.vk, movedRec.mod, movedRec.name)
			m.syncRecKeyCombo(movedRec.name)
		}
	}
	m.showHint("热键 "+opt.name+" 注册失败，可能被其他程序占用", true)
}

func (m *TForm1) syncHotkeyCombo(name string) {
	if m.ComboBoxHotkey == nil {
		return
	}
	m.suppressHotkeyChange = true
	for i, item := range hotkeyOptions {
		if item.name == name {
			m.ComboBoxHotkey.SetItemIndex(int32(i))
			break
		}
	}
	m.suppressHotkeyChange = false
}

// onRecordHotkeyChanged 录制热键下拉变化：不能和启停热键重复，注册失败要回滚
func (m *TForm1) onRecordHotkeyChanged() {
	if m.suppressRecKeyChange || m.ComboRecKey == nil {
		return
	}
	idx := int(m.ComboRecKey.ItemIndex())
	if idx < 0 || idx >= len(hotkeyOptions) {
		return
	}
	oldName, oldVK, oldMod := m.recordHotkeyName, m.recordHotkeyVK, m.recordHotkeyMod
	opt := hotkeyOptions[idx]
	if opt.name == m.hotkeyName {
		m.syncRecKeyCombo(oldName)
		m.showHint("录制热键不能和启停热键相同", true)
		return
	}
	if m.applyRecordHotkey(opt.vk, opt.mod, opt.name) {
		m.cfg.update(func(c *appConfig) { c.RecordHotkey = opt.name })
		m.showHint("录制热键已改为 "+opt.name, false)
		return
	}
	m.syncRecKeyCombo(oldName)
	m.applyRecordHotkey(oldVK, oldMod, oldName)
	m.showHint("录制热键 "+opt.name+" 注册失败，可能被其他程序占用", true)
}

func (m *TForm1) syncRecKeyCombo(name string) {
	if m.ComboRecKey == nil {
		return
	}
	m.suppressRecKeyChange = true
	m.ComboRecKey.SetItemIndex(int32(recordHotkeyIndex(name)))
	m.suppressRecKeyChange = false
}

func (m *TForm1) applyRecordHotkey(vk, mod uint32, name string) bool {
	if m.recordHotkeySet {
		unregisterHotKey(uintptr(m.Handle()), recordHotkeyID)
		m.recordHotkeySet = false
	}
	if registerHotKey(uintptr(m.Handle()), recordHotkeyID, vk, mod) {
		m.recordHotkeySet = true
		m.recordHotkeyVK = vk
		m.recordHotkeyMod = mod
		m.recordHotkeyName = name
		recLog("record hotkey ok: %s vk=%d mod=0x%x", name, vk, mod)
		return true
	}
	recLog("record hotkey failed: %s vk=%d mod=0x%x", name, vk, mod)
	return false
}

func (m *TForm1) applyHotkey(vk, mod uint32, name string) bool {
	if m.hotkeySet {
		unregisterHotKey(uintptr(m.Handle()), hotkeyID)
		m.hotkeySet = false
	}
	if registerHotKey(uintptr(m.Handle()), hotkeyID, vk, mod) {
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
	if m.isScriptMode() {
		if m.run.state() == stateStopped {
			m.ButtonToggle.SetCaption("开始脚本 (" + m.hotkeyName + ")")
			m.lastButtonText = m.ButtonToggle.Caption()
		}
		m.showHint("按 "+m.hotkeyName+" 运行 · "+m.recordHotkeyName+" 录制 · Esc 急停", false)
		return
	}
	if m.engine.state() == stateStopped {
		m.ButtonToggle.SetCaption("开始 (" + m.hotkeyName + ")")
		m.lastButtonText = m.ButtonToggle.Caption()
	}
	if m.isDoubleMode() {
		m.showHint("双击：两下为一组，组间等待 · "+m.hotkeyName+" 开始 / 再按暂停", false)
	} else {
		m.showHint("连点：在鼠标位置持续点击 · "+m.hotkeyName+" 开始 / 再按暂停", false)
	}
}

func (m *TForm1) showHint(text string, isError bool) {
	m.LabelHint.SetCaption(text)
	if isError {
		m.LabelHint.Font().SetColor(colors.RGBToColor(200, 60, 60))
	} else {
		m.LabelHint.Font().SetColor(colors.RGBToColor(0, 70, 200))
	}
}

func (m *TForm1) refreshUI() {
	script := m.isScriptMode()
	state := m.engine.state()
	n := m.engine.clicks()
	if script {
		state = m.run.state()
		n = m.run.clicks()
	}
	var text, btnText string
	switch state {
	case stateRunning:
		text = "● 运行中"
		if script {
			btnText = "停止脚本 (" + m.hotkeyName + ")"
		} else {
			btnText = "暂停 (" + m.hotkeyName + ")"
		}
	case stateArming:
		text = "● 即将开始…"
		if script {
			btnText = "取消启动 (" + m.hotkeyName + ")"
		} else {
			btnText = "取消 (" + m.hotkeyName + ")"
		}
	default:
		text = "● 已暂停"
		if script {
			btnText = "开始脚本 (" + m.hotkeyName + ")"
		} else {
			btnText = "开始 (" + m.hotkeyName + ")"
		}
	}
	if m.LabelStatus.Caption() != text {
		m.LabelStatus.SetCaption(text)
		m.LabelStatus.Font().SetColor(statusColor(state))
	}
	if btnText != m.lastButtonText {
		m.ButtonToggle.SetCaption(btnText)
		m.lastButtonText = btnText
	}
	if n != m.lastCount {
		m.lastCount = n
		if script {
			m.LabelCount.SetCaption(fmt.Sprintf("已执行 %d 次", n))
		} else {
			m.LabelCount.SetCaption(fmt.Sprintf("已点击 %d 次", n))
		}
	}
	if script {
		m.refreshSeqHint(state, n)
	}
}

// refreshSeqHint 脚本模式下把进度写到底部提示
func (m *TForm1) refreshSeqHint(state runState, clicked int64) {
	if m.pickLeft > 0 {
		return // 取点倒计时期间不覆盖提示
	}
	if m.recording || m.recTimer.Enabled() {
		return // 录制与倒计时期间保留录制提示
	}
	if !m.hintHoldTill.IsZero() && time.Now().Before(m.hintHoldTill) {
		return // 刚录完的提示多留一会儿
	}
	step, total, loop, reason := m.run.progress()
	switch state {
	case stateRunning:
		m.showHint(fmt.Sprintf("第 %d/%d 步 · 第 %d 轮 · 已执行 %d 次（Esc 急停）", step, total, loop, clicked), false)
	case stateArming:
		m.showHint("延迟启动中，按热键可取消", false)
	default:
		switch reason {
		case limitDuration:
			m.showHint("已到总时长，自动停止", false)
		case limitClicks:
			m.showHint("已达执行次数上限，自动停止", false)
		case limitEscape:
			m.showHint("Esc 急停，已停止", false)
		case limitDone:
			m.showHint("序列已跑完，自动停止", false)
		default:
			if len(m.seq) == 0 {
				m.showHint("点「● 录制」录一段操作，或「新增步骤」手工加一步", false)
			} else {
				m.showHint("按 "+m.hotkeyName+" 运行 · "+m.recordHotkeyName+" 录制 · Esc 急停", false)
			}
		}
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
	return fmt.Sprintf("点击速度：%d 次/秒（间隔 %d ms，范围 1–50）", rate, 1000/rate)
}
