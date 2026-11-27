package app

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/energye/lcl/lcl"
	"github.com/energye/lcl/types"
	"github.com/energye/lcl/types/colors"
)

// 序列编辑区的版面（400 宽窗口）。横向固定四列：左标签 16、左控件 92、
// 右标签 232、右控件 292；纵向所有行间距统一，分组标题前后留白一致。
const (
	seqFormWidth          = 400
	seqContentLeft        = 16
	seqContentWidth       = 368
	seqPanelTop           = 112
	seqPanelHeightCompact = 560 // 非拖拽：不显示终点行
	seqPanelHeightFull    = 592 // 拖拽：多一行终点
	seqLabelWidth         = 72
	seqCtrlX              = 92
	seqLabel2X            = 232
	seqLabel2Width        = 52
	seqCtrl2X             = 292
	seqCtrl2Width         = 92
)

// initSequenceUI 创建脚本模式所需的全部控件。这些控件不在 Designer 布局里，
// 只在脚本模式显示，不影响现有的单击/双击界面。
func (m *TForm1) initSequenceUI() {
	m.editingStep = -1
	m.pickTarget = 0

	// 统一窗口宽度：脚本面板与常规模式共用一套左右边距
	m.SetWidth(seqFormWidth)

	// 常规模式的整宽控件跟着窗口延长，避免右侧留出参差的空白
	m.TrackBarSpeed.SetWidth(seqContentWidth)
	m.TrackBarDoubleGap.SetWidth(seqContentWidth)
	m.TrackBarDoubleInterval.SetWidth(seqContentWidth)
	m.LabelSpeed.SetWidth(seqContentWidth)
	m.LabelDoubleGap.SetWidth(seqContentWidth)
	m.LabelDoubleInterval.SetWidth(seqContentWidth)
	m.CheckBoxLock.SetWidth(seqContentWidth)
	m.ButtonToggle.SetWidth(seqContentWidth)
	m.LabelHint.SetWidth(seqContentWidth)
	m.CheckBoxCloseToTray.SetLeft(seqFormWidth - seqContentLeft - 96)
	m.CheckBoxCloseToTray.SetWidth(96)
	m.ComboBoxHotkey.SetWidth(120)
	m.LinkGitHub.SetLeft(seqContentLeft)
	m.LinkGitHub.SetWidth(seqContentWidth)
	m.LinkGitHub.SetAlignment(types.TaRightJustify)

	// 三个模式用两个字的短标签，与上方「鼠标按键」单选行同列对齐
	m.RadioSingle.SetCaption("连点")
	m.LabelMode.SetCaption("模式")
	m.RadioDouble.SetCaption("双击")
	m.RadioSingle.SetLeft(88)
	m.RadioSingle.SetTop(82)
	m.RadioSingle.SetWidth(56)
	m.RadioDouble.SetLeft(150)
	m.RadioDouble.SetTop(82)
	m.RadioDouble.SetWidth(56)

	r := lcl.NewRadioButton(m)
	r.SetCaption("脚本")
	r.SetLeft(215)
	r.SetTop(82)
	r.SetWidth(56)
	r.SetHeight(22)
	r.SetParent(m)
	r.SetOnClick(func(lcl.IObject) { m.setMode(modeScript) })
	m.RadioScript = r

	// 模式说明放在单选行上方（脚本模式下"鼠标按键"整行隐藏，位置空着）
	m.LabelSeqHead = m.seqLabel(m, "脚本模式：把鼠标动作写成步骤，按顺序自动执行", seqContentLeft, 52, seqContentWidth)
	m.LabelSeqHead.Font().SetColor(colors.RGBToColor(110, 110, 110))
	m.LabelSeqHead.SetVisible(false)

	p := lcl.NewPanel(m)
	p.SetBevelOuter(types.BvNone)
	p.SetCaption("")
	p.SetColor(colors.RGBToColor(255, 255, 255))
	p.SetLeft(0)
	p.SetTop(seqPanelTop)
	p.SetWidth(seqFormWidth)
	p.SetHeight(seqPanelHeightCompact)
	p.SetParent(m)
	p.SetVisible(false)
	m.PanelSeq = p

	m.LabelStepHead = m.seqHead(p, "① 步骤列表", 4)
	m.LabelActHead = m.seqHead(p, "② 动作设置", 192)
	m.LabelRunHead = m.seqHead(p, "③ 运行设置", 374)

	lb := lcl.NewListBox(m)
	lb.SetLeft(seqContentLeft)
	lb.SetTop(30)
	lb.SetWidth(seqContentWidth)
	lb.SetHeight(116)
	lb.SetColor(colors.RGBToColor(255, 255, 255))
	lb.SetParent(p)
	lb.SetOnClick(func(lcl.IObject) { m.loadStepFromList() })
	m.ListSteps = lb

	m.BtnStepAdd = m.seqButton(p, "新增步骤", 16, 152, 88, 28, func(lcl.IObject) { m.addStep() })
	m.BtnStepApply = m.seqButton(p, "保存本步", 116, 152, 88, 28, func(lcl.IObject) { m.applyStep() })
	m.BtnStepDel = m.seqButton(p, "删除", 216, 152, 56, 28, func(lcl.IObject) { m.deleteStep() })
	m.BtnStepUp = m.seqButton(p, "↑", 284, 152, 44, 28, func(lcl.IObject) { m.moveStep(-1) })
	m.BtnStepDown = m.seqButton(p, "↓", 340, 152, 44, 28, func(lcl.IObject) { m.moveStep(1) })
	m.BtnRecord = m.seqButton(p, "● 录制", seqCtrl2X, 1, seqCtrl2Width, 22, func(lcl.IObject) { m.toggleRecord() })

	m.seqLabel(p, "动作", seqContentLeft, 226, seqLabelWidth)
	m.ComboAct = m.seqCombo(p, []string{"点击", "长按", "拖拽", "滚轮"}, 0, seqCtrlX, 218, 120, nil)
	m.LabelBtn = m.seqLabel(p, "按键", seqLabel2X, 226, seqLabel2Width)
	m.ComboBtn = m.seqCombo(p, []string{"左键", "中键", "右键"}, 0, seqCtrl2X, 218, seqCtrl2Width, nil)

	m.LabelActDesc = m.seqLabel(p, "", seqContentLeft, 250, seqContentWidth)
	m.LabelActDesc.Font().SetColor(colors.RGBToColor(110, 110, 110))

	m.LabelPosStart = m.seqLabel(p, "位置", seqContentLeft, 282, 60)
	m.LabelPosX1 = m.seqLabel(p, "X", 76, 282, 16)
	m.EditX = m.seqEdit(p, "", 92, 274, 72)
	m.LabelPosY1 = m.seqLabel(p, "Y", 168, 282, 16)
	m.EditY = m.seqEdit(p, "", 184, 274, 72)
	m.BtnPickFrom = m.seqButton(p, "取点", seqCtrl2X, 272, seqCtrl2Width, 28, func(lcl.IObject) { m.pickCursor(0) })

	m.LabelPosEnd = m.seqLabel(p, "终点", seqContentLeft, 314, 60)
	m.LabelPosX2 = m.seqLabel(p, "X", 76, 314, 16)
	m.EditX2 = m.seqEdit(p, "", 92, 306, 72)
	m.LabelPosY2 = m.seqLabel(p, "Y", 168, 314, 16)
	m.EditY2 = m.seqEdit(p, "", 184, 306, 72)
	m.BtnPickTo = m.seqButton(p, "取终点", seqCtrl2X, 304, seqCtrl2Width, 28, func(lcl.IObject) { m.pickCursor(1) })

	m.LabelParam = m.seqLabel(p, "参数", seqContentLeft, 314, seqLabelWidth)
	m.EditParam = m.seqEdit(p, "", seqCtrlX, 306, 60)
	m.LabelParamTip = m.seqLabel(p, "", 158, 314, 226)
	m.LabelParamTip.Font().SetColor(colors.RGBToColor(110, 110, 110))

	m.LabelGap = m.seqLabel(p, "间隔", seqContentLeft, 346, seqLabelWidth)
	m.EditGap = m.seqEdit(p, "50", seqCtrlX, 338, 60)
	m.LabelGapMs = m.seqLabel(p, "ms", 158, 346, 20)
	m.LabelRepeat = m.seqLabel(p, "重复", seqLabel2X, 346, seqLabel2Width)
	m.EditRepeat = m.seqEdit(p, "1", seqCtrl2X, 338, seqCtrl2Width)

	m.LabelLoop = m.seqLabel(p, "循环", seqContentLeft, 410, seqLabelWidth)
	m.ComboLoop = m.seqCombo(p, []string{"跑一遍", "循环 N 次", "一直循环"}, 0, seqCtrlX, 402, 120, nil)
	m.LabelLoopCnt = m.seqLabel(p, "次数", seqLabel2X, 410, seqLabel2Width)
	m.EditLoopCnt = m.seqEdit(p, "1", seqCtrl2X, 402, seqCtrl2Width)

	m.LabelMaxDur = m.seqLabel(p, "总时长", seqContentLeft, 442, seqLabelWidth)
	m.EditMaxDur = m.seqEdit(p, "0", seqCtrlX, 434, 60)
	m.LabelMaxSec = m.seqLabel(p, "秒", 158, 442, 16)
	m.LabelMaxClick = m.seqLabel(p, "最多(次)", seqLabel2X, 442, seqLabel2Width)
	m.EditMaxClick = m.seqEdit(p, "0", seqCtrl2X, 434, seqCtrl2Width)

	m.LabelDelay = m.seqLabel(p, "延迟", seqContentLeft, 474, seqLabelWidth)
	m.EditDelay = m.seqEdit(p, "0", seqCtrlX, 466, 60)
	m.LabelDelayTip = m.seqLabel(p, "秒 · 0 为立即开始", 158, 474, 226)
	m.LabelDelayTip.Font().SetColor(colors.RGBToColor(110, 110, 110))

	m.LabelRecKey = m.seqLabel(p, "录制热键", seqContentLeft, 506, seqLabelWidth)
	recNames := make([]string, 0, len(hotkeyOptions))
	for _, opt := range hotkeyOptions {
		recNames = append(recNames, opt.name)
	}
	m.ComboRecKey = m.seqCombo(p, recNames, defaultRecordHotkeyIndex, seqCtrlX, 498, 120, func(lcl.IObject) { m.onRecordHotkeyChanged() })
	m.LabelRecTip = m.seqLabel(p, "随时开始 / 停止录制", 220, 506, 164)
	m.LabelRecTip.Font().SetColor(colors.RGBToColor(110, 110, 110))

	for _, l := range []lcl.ILabel{m.LabelBtn, m.LabelRepeat, m.LabelLoopCnt, m.LabelMaxClick} {
		l.SetAlignment(types.TaRightJustify)
	}

	m.LabelSeqTip = m.seqLabel(p, "取点会倒计时 3 秒，先把鼠标移到目标位置", seqContentLeft, 534, seqContentWidth)
	m.LabelSeqTip.Font().SetColor(colors.RGBToColor(110, 110, 110))

	m.pickTimer = lcl.NewTimer(m)
	m.pickTimer.SetInterval(1000)
	m.pickTimer.SetEnabled(false)
	m.pickTimer.SetOnTimer(func(lcl.IObject) { m.onPickTick() })

	m.recTimer = lcl.NewTimer(m)
	m.recTimer.SetInterval(1000)
	m.recTimer.SetEnabled(false)
	m.recTimer.SetOnTimer(func(lcl.IObject) { m.onRecTick() })

	m.ComboAct.SetOnChange(func(lcl.IObject) { m.updateActionUI() })
	m.ComboBtn.SetOnChange(func(lcl.IObject) { m.persistSequence() })
	m.ComboLoop.SetOnChange(func(lcl.IObject) { m.updateLoopUI(); m.persistSequence() })
	for _, ed := range []lcl.IEdit{m.EditLoopCnt, m.EditMaxDur, m.EditMaxClick, m.EditDelay} {
		ed.SetOnChange(func(lcl.IObject) { m.persistSequence() })
	}

	m.updateActionUI()
	m.updateLoopUI()
}

func (m *TForm1) seqLabel(parent lcl.IWinControl, text string, left, top, width int32) lcl.ILabel {
	l := lcl.NewLabel(m)
	l.SetAutoSize(false)
	l.SetCaption(text)
	l.SetLeft(left)
	l.SetTop(top)
	l.SetWidth(width)
	l.SetHeight(18)
	l.SetParent(parent)
	return l
}

func (m *TForm1) seqEdit(parent lcl.IWinControl, text string, left, top, width int32) lcl.IEdit {
	e := lcl.NewEdit(m)
	e.SetText(text)
	e.SetLeft(left)
	e.SetTop(top)
	e.SetWidth(width)
	e.SetHeight(24)
	e.SetParent(parent)
	return e
}

func (m *TForm1) seqButton(parent lcl.IWinControl, caption string, left, top, width, height int32, fn lcl.TNotifyEvent) lcl.IButton {
	b := lcl.NewButton(m)
	b.SetCaption(caption)
	b.SetLeft(left)
	b.SetTop(top)
	b.SetWidth(width)
	b.SetHeight(height)
	b.SetParent(parent)
	b.SetOnClick(fn)
	return b
}

func (m *TForm1) seqCombo(parent lcl.IWinControl, items []string, index int32, left, top, width int32, fn lcl.TNotifyEvent) lcl.IComboBox {
	c := lcl.NewComboBox(m)
	c.SetStyle(types.CsDropDownList)
	for _, it := range items {
		c.Items().Add(it)
	}
	c.SetItemIndex(index)
	c.SetLeft(left)
	c.SetTop(top)
	c.SetWidth(width)
	c.SetHeight(24)
	c.SetParent(parent)
	if fn != nil {
		c.SetOnChange(fn)
	}
	return c
}

// ---------- 布局与动作联动 ----------

// seqHead 分组标题：加粗、深色，用来分隔「步骤 / 动作 / 运行」三块
func (m *TForm1) seqHead(parent lcl.IWinControl, text string, top int32) lcl.ILabel {
	l := m.seqLabel(parent, text, seqContentLeft, top, seqContentWidth)
	l.Font().SetStyle(types.NewSet(types.FsBold))
	l.Font().SetColor(colors.RGBToColor(40, 70, 120))
	return l
}

func (m *TForm1) seqWithEnd() bool {
	return m.ComboAct != nil && m.ComboAct.ItemIndex() == 2
}

// setSeqTip ② 下方的说明行：普通提示灰色，校验错误用红色，避免漏看
func (m *TForm1) setSeqTip(text string, isError bool) {
	if m.LabelSeqTip == nil {
		return
	}
	m.LabelSeqTip.SetCaption(text)
	if isError {
		m.LabelSeqTip.Font().SetColor(colors.RGBToColor(200, 60, 60))
	} else {
		m.LabelSeqTip.Font().SetColor(colors.RGBToColor(110, 110, 110))
	}
}

// updateLoopUI 只有「循环 N 次」时「次数」才有意义，其他模式置灰，避免以为填了会生效
func (m *TForm1) updateLoopUI() {
	if m.ComboLoop == nil || m.EditLoopCnt == nil {
		return
	}
	m.EditLoopCnt.SetEnabled(m.ComboLoop.ItemIndex() == 1)
}

// updateActionUI 动作变了就改标签、说明和显隐：滚轮不要按键，非拖拽不要终点
func (m *TForm1) updateActionUI() {
	if m.ComboAct == nil {
		return
	}
	idx := m.ComboAct.ItemIndex()
	var desc, paramCap, paramTip, posCap string
	posCap = "位置"
	switch idx {
	case 1:
		desc = "长按：按住不放，保持指定毫秒后松开；留空 = 当前光标"
		paramCap = "长按"
		paramTip = "毫秒"
		m.BtnPickFrom.SetCaption("取点")
	case 2:
		desc = "拖拽：在起点按下不放，移到终点后再松开"
		paramCap = "步数"
		paramTip = "分几次移动（2–60，越大越慢）"
		posCap = "起点"
		m.BtnPickFrom.SetCaption("取起点")
	case 3:
		desc = "滚轮：在坐标处滚动，不需要选按键；留空 = 当前光标"
		paramCap = "档数"
		paramTip = "正数向上，负数向下"
		m.BtnPickFrom.SetCaption("取点")
	default:
		desc = "点击：在坐标处按一次（默认 30ms）；留空 = 当前光标"
		paramCap = "按住"
		paramTip = "毫秒（默认 30）"
		m.BtnPickFrom.SetCaption("取点")
	}
	m.LabelActDesc.SetCaption(desc)
	m.LabelPosStart.SetCaption(posCap)
	m.LabelParam.SetCaption(paramCap)
	m.LabelParamTip.SetCaption(paramTip)
	showBtn := idx != 3
	m.LabelBtn.SetVisible(showBtn)
	m.ComboBtn.SetVisible(showBtn)

	m.layoutSeqFields()
	m.layoutSeqWindow()
}

// layoutSeqFields 按是否需要终点行，把下面的行整体下移一行（32px）
func (m *TForm1) layoutSeqFields() {
	withEnd := m.seqWithEnd()
	m.LabelPosEnd.SetVisible(withEnd)
	m.LabelPosX2.SetVisible(withEnd)
	m.EditX2.SetVisible(withEnd)
	m.LabelPosY2.SetVisible(withEnd)
	m.EditY2.SetVisible(withEnd)
	m.BtnPickTo.SetVisible(withEnd)

	var shift int32
	if withEnd {
		shift = 32
	}

	m.LabelParam.SetTop(314 + shift)
	m.EditParam.SetTop(306 + shift)
	m.LabelParamTip.SetTop(314 + shift)

	m.LabelGap.SetTop(346 + shift)
	m.EditGap.SetTop(338 + shift)
	m.LabelGapMs.SetTop(346 + shift)
	m.LabelRepeat.SetTop(346 + shift)
	m.EditRepeat.SetTop(338 + shift)

	m.LabelRunHead.SetTop(378 + shift)
	m.LabelLoop.SetTop(410 + shift)
	m.ComboLoop.SetTop(402 + shift)
	m.LabelLoopCnt.SetTop(410 + shift)
	m.EditLoopCnt.SetTop(402 + shift)

	m.LabelMaxDur.SetTop(442 + shift)
	m.EditMaxDur.SetTop(434 + shift)
	m.LabelMaxSec.SetTop(442 + shift)
	m.LabelMaxClick.SetTop(442 + shift)
	m.EditMaxClick.SetTop(434 + shift)

	m.LabelDelay.SetTop(474 + shift)
	m.EditDelay.SetTop(466 + shift)
	m.LabelDelayTip.SetTop(474 + shift)

	m.LabelRecKey.SetTop(506 + shift)
	m.ComboRecKey.SetTop(498 + shift)
	m.LabelRecTip.SetTop(506 + shift)

	m.LabelSeqTip.SetTop(534 + shift)

	if withEnd {
		m.PanelSeq.SetHeight(seqPanelHeightFull)
	} else {
		m.PanelSeq.SetHeight(seqPanelHeightCompact)
	}
}

// layoutSeqWindow 面板高度变化后，下面的按钮、提示和窗口一起贴合
func (m *TForm1) layoutSeqWindow() {
	h := int32(seqPanelHeightCompact)
	if m.seqWithEnd() {
		h = seqPanelHeightFull
	}
	btnTop := int32(seqPanelTop) + h + 12
	m.ButtonToggle.SetTop(btnTop)
	m.LabelHint.SetTop(btnTop + 58)
	m.SetHeight(btnTop + 58 + 64)
}

// showSeqMode 进入/离开脚本模式时统一切换面板与模式说明
func (m *TForm1) showSeqMode(on bool) {
	if m.PanelSeq == nil {
		return
	}
	m.LabelSeqHead.SetVisible(on)
	m.PanelSeq.SetVisible(on)
	if on {
		m.updateActionUI()
		m.reloadStepList()
	}
}

// ---------- 数据与界面同步 ----------

func (m *TForm1) applySeqConfig(cfg seqConfig) {
	cfg = cfg.normalized()
	m.seq = append([]seqStep{}, cfg.Steps...)
	m.seqCfg = cfg

	prev := m.loadingConfig
	m.loadingConfig = true
	switch cfg.LoopMode {
	case loopCount:
		m.ComboLoop.SetItemIndex(1)
	case loopForever:
		m.ComboLoop.SetItemIndex(2)
	default:
		m.ComboLoop.SetItemIndex(0)
	}
	m.EditLoopCnt.SetText(strconv.Itoa(cfg.LoopCount))
	m.EditMaxDur.SetText(strconv.Itoa(cfg.MaxDurationSec))
	m.EditMaxClick.SetText(strconv.Itoa(cfg.MaxClicks))
	m.EditDelay.SetText(strconv.Itoa(cfg.StartDelayMs / 1000))
	m.loadingConfig = prev

	m.updateLoopUI()
	m.reloadStepList()
}

func (m *TForm1) reloadStepList() {
	if m.ListSteps == nil {
		return
	}
	lb := m.ListSteps.Items()
	lb.BeginUpdate()
	lb.Clear()
	for i, st := range m.seq {
		lb.Add(st.listText(i))
	}
	lb.EndUpdate()
	if len(m.seq) == 0 {
		m.editingStep = -1
	} else if m.editingStep < 0 || m.editingStep >= int32(len(m.seq)) {
		m.editingStep = int32(len(m.seq) - 1)
	}
	if m.editingStep >= 0 && m.editingStep < int32(len(m.seq)) {
		m.ListSteps.SetItemIndex(m.editingStep)
		m.writeStepFields(m.seq[m.editingStep])
	}

	if m.LabelStepHead != nil {
		if len(m.seq) == 0 {
			m.LabelStepHead.SetCaption("① 步骤列表（空，点「录制」开始）")
		} else {
			m.LabelStepHead.SetCaption(fmt.Sprintf("① 步骤列表（%d 步）", len(m.seq)))
		}
	}
}

func (m *TForm1) writeStepFields(st seqStep) {
	prev := m.loadingConfig
	m.loadingConfig = true
	switch st.Action {
	case actHold:
		m.ComboAct.SetItemIndex(1)
	case actDrag:
		m.ComboAct.SetItemIndex(2)
	case actWheel:
		m.ComboAct.SetItemIndex(3)
	default:
		m.ComboAct.SetItemIndex(0)
	}
	switch parseButton(st.Button) {
	case btnMiddle:
		m.ComboBtn.SetItemIndex(1)
	case btnRight:
		m.ComboBtn.SetItemIndex(2)
	default:
		m.ComboBtn.SetItemIndex(0)
	}
	m.EditX.SetText(coordText(st.X))
	m.EditY.SetText(coordText(st.Y))
	m.EditX2.SetText(coordText(st.X2))
	m.EditY2.SetText(coordText(st.Y2))
	m.EditParam.SetText(strconv.Itoa(st.Param))
	m.EditGap.SetText(strconv.Itoa(st.DelayMs))
	m.EditRepeat.SetText(strconv.Itoa(st.Repeat))
	m.loadingConfig = prev
	m.updateActionUI()
}

func (m *TForm1) loadStepFromList() {
	idx := m.ListSteps.ItemIndex()
	if idx < 0 || idx >= int32(len(m.seq)) {
		return
	}
	m.editingStep = idx
	m.writeStepFields(m.seq[idx])
}

func (m *TForm1) readStepFields() seqStep {
	st := defaultSeqStep()
	switch m.ComboAct.ItemIndex() {
	case 1:
		st.Action = actHold
	case 2:
		st.Action = actDrag
	case 3:
		st.Action = actWheel
	default:
		st.Action = actClick
	}
	switch m.ComboBtn.ItemIndex() {
	case 1:
		st.Button = "middle"
	case 2:
		st.Button = "right"
	default:
		st.Button = "left"
	}
	st.X = parseEditCoord(m.EditX)
	st.Y = parseEditCoord(m.EditY)
	st.X2 = parseEditCoord(m.EditX2)
	st.Y2 = parseEditCoord(m.EditY2)
	st.Param = parseEditInt(m.EditParam, 0)
	st.DelayMs = parseEditInt(m.EditGap, 50)
	st.Repeat = parseEditInt(m.EditRepeat, 1)
	return st.normalized()
}

func (m *TForm1) addStep() {
	st := m.readStepFields()
	if st.Action == actDrag && (!st.hasPos() || !st.hasEnd()) {
		m.setSeqTip("拖拽步骤需要起点和终点坐标（可点“取起点/取终点”）", true)
		return
	}
	m.seq = append(m.seq, st)
	m.editingStep = int32(len(m.seq) - 1)
	m.reloadStepList()
	m.persistSequence()
	m.setSeqTip("已添加步骤", false)
}

func (m *TForm1) applyStep() {
	st := m.readStepFields()
	if m.editingStep < 0 || m.editingStep >= int32(len(m.seq)) {
		m.setSeqTip("请先在列表里选一行", true)
		return
	}
	if st.Action == actDrag && (!st.hasPos() || !st.hasEnd()) {
		m.setSeqTip("拖拽步骤需要起点和终点坐标", true)
		return
	}
	m.seq[m.editingStep] = st
	m.reloadStepList()
	m.persistSequence()
	m.setSeqTip(fmt.Sprintf("第 %d 步已更新", m.editingStep+1), false)
}

func (m *TForm1) deleteStep() {
	if m.editingStep < 0 || m.editingStep >= int32(len(m.seq)) {
		m.setSeqTip("请先在列表里选一行", true)
		return
	}
	idx := m.editingStep
	m.seq = append(m.seq[:idx], m.seq[idx+1:]...)
	if m.editingStep >= int32(len(m.seq)) {
		m.editingStep = int32(len(m.seq) - 1)
	}
	m.reloadStepList()
	m.persistSequence()
	m.setSeqTip("已删除该步骤", false)
}

func (m *TForm1) moveStep(delta int) {
	idx := m.editingStep
	if idx < 0 || idx >= int32(len(m.seq)) {
			m.setSeqTip("请先在列表里选一行", true)
		return
	}
	to := idx + int32(delta)
	if to < 0 || to >= int32(len(m.seq)) {
		return
	}
	m.seq[idx], m.seq[to] = m.seq[to], m.seq[idx]
	m.editingStep = to
	m.reloadStepList()
	m.persistSequence()
}

// readSeqConfigFromUI 把界面上的运行控制与步骤表合成一份配置
func (m *TForm1) readSeqConfigFromUI() seqConfig {
	cfg := seqConfig{Steps: append([]seqStep{}, m.seq...)}
	switch m.ComboLoop.ItemIndex() {
	case 1:
		cfg.LoopMode = loopCount
	case 2:
		cfg.LoopMode = loopForever
	default:
		cfg.LoopMode = loopOnce
	}
	cfg.LoopCount = parseEditInt(m.EditLoopCnt, 1)
	cfg.MaxDurationSec = parseEditInt(m.EditMaxDur, 0)
	cfg.MaxClicks = parseEditInt(m.EditMaxClick, 0)
	cfg.StartDelayMs = parseEditInt(m.EditDelay, 0) * 1000
	return cfg.normalized()
}

func (m *TForm1) persistSequence() {
	if m.loadingConfig {
		return
	}
	cfg := m.readSeqConfigFromUI()
	m.seqCfg = cfg
	seq := cfg
	m.cfg.update(func(c *appConfig) { c.Sequence = seq })
}

// ---------- 录制 ----------

// onHotkeyPress 启停热键：录制中先停录制，倒计时中则取消，其余交给运行开关
func (m *TForm1) onHotkeyPress() {
	if m.recording {
		m.stopRecord(true)
		return
	}
	if m.recTimer.Enabled() {
		m.cancelRecordStart()
		return
	}
	m.toggle()
}

// onRecordHotkey 录制热键：随时开始 / 停止录制，不在脚本模式时自动切过去
func (m *TForm1) onRecordHotkey() {
	if m.recording {
		m.stopRecord(true)
		return
	}
	if m.recTimer.Enabled() {
		m.cancelRecordStart()
		return
	}
	if !m.isScriptMode() {
		m.setMode(modeScript)
	}
	m.toggleRecord()
}

func (m *TForm1) cancelRecordStart() {
	m.recTimer.SetEnabled(false)
	m.BtnRecord.SetCaption("● 录制")
	m.refreshSeqHint(m.run.state(), m.run.clicks())
}

func (m *TForm1) showNotice(text string) {
	m.hintHoldTill = time.Now().Add(3 * time.Second)
	m.showHint(text, false)
}

// toggleRecord 点「录制」：开始 3 秒倒计时，或再点一次停止
func (m *TForm1) toggleRecord() {
	if m.recording {
		m.stopRecord(true)
		return
	}
	if m.recTimer.Enabled() {
		m.cancelRecordStart()
		return
	}
	m.engine.stop()
	m.run.stop()
	m.run.clearResult()
	m.recLeft = 3
	m.BtnRecord.SetCaption(fmt.Sprintf("%d 秒后录制", m.recLeft))
	m.showHint(fmt.Sprintf("3 秒后开始录制：切到要操作的界面，按 %s 停止", m.recordHotkeyName), false)
	m.recTimer.SetEnabled(true)
}

func (m *TForm1) onRecTick() {
	m.recLeft--
	if m.recLeft > 0 {
		m.BtnRecord.SetCaption(fmt.Sprintf("%d 秒后录制", m.recLeft))
		return
	}
	m.recTimer.SetEnabled(false)
	m.rec.reset()
	installRecorder(&m.rec)
	if !startMouseHook() {
		uninstallRecorder()
		m.BtnRecord.SetCaption("● 录制")
		m.showHint("无法安装鼠标钩子，录制未开始", true)
		recLog("record start failed: hook not installed")
		return
	}
	recLog("record start: countdown done, hook ready")
	m.recording = true
	m.recCount = 0
	m.BtnRecord.SetCaption("■ 停止")
	m.SetCaption("● 录制中 — 鼠标连点器")
	m.SetWindowState(types.WsMinimized)
}

// stopRecord 停止录制并把结果追加到步骤表
func (m *TForm1) stopRecord(byUser bool) {
	if !m.recording {
		return
	}
	m.recording = false
	uninstallRecorder()
	stopMouseHook()
	m.SetWindowState(types.WsNormal)
	m.SetCaption("鼠标连点器")
	m.BtnRecord.SetCaption("● 录制")
	if byUser {
		hwnd := uintptr(m.Handle())
		m.rec.dropLastClickInside(func(x, y int32) bool { return windowContains(hwnd, x, y) })
	}
	steps := m.rec.steps()
	recLog("record stop: byUser=%v steps=%d", byUser, len(steps))
	if len(steps) == 0 {
		m.showHint("没有录到鼠标操作", true)
		return
	}
	room := maxSeqSteps - len(m.seq)
	if room <= 0 {
		m.showHint("步骤已满（最多 300 步），本次录制未插入", true)
		return
	}
	if len(steps) > room {
		steps = steps[:room]
	}
	m.seq = append(m.seq, steps...)
	m.editingStep = int32(len(m.seq) - 1)
	m.reloadStepList()
	m.persistSequence()
	m.showNotice(fmt.Sprintf("已录制 %d 步，已插入步骤表", len(steps)))
}

// ---------- 取点 ----------

func (m *TForm1) pickCursor(target int) {
	m.pickTarget = target
	m.pickLeft = 3
	m.setSeqTip("3 秒后取点，请把鼠标移到目标位置…", false)
	m.pickTimer.SetEnabled(true)
}

func (m *TForm1) onPickTick() {
	if m.pickLeft > 1 {
		m.pickLeft--
		m.setSeqTip(fmt.Sprintf("%d 秒后取点…", m.pickLeft), false)
		return
	}
	m.pickTimer.SetEnabled(false)
	pos := lcl.Mouse.CursorPos()
	if m.pickTarget == 0 {
		m.EditX.SetText(strconv.Itoa(int(pos.X)))
		m.EditY.SetText(strconv.Itoa(int(pos.Y)))
		m.setSeqTip(fmt.Sprintf("起点已取：(%d,%d)", pos.X, pos.Y), false)
		return
	}
	m.EditX2.SetText(strconv.Itoa(int(pos.X)))
	m.EditY2.SetText(strconv.Itoa(int(pos.Y)))
	m.setSeqTip(fmt.Sprintf("终点已取：(%d,%d)", pos.X, pos.Y), false)
}

// ---------- 小工具 ----------

func coordText(v int32) string {
	if v < 0 {
		return ""
	}
	return strconv.Itoa(int(v))
}

func parseEditInt(ed lcl.IEdit, def int) int {
	if ed == nil {
		return def
	}
	s := strings.TrimSpace(ed.Text())
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func parseEditCoord(ed lcl.IEdit) int32 {
	if ed == nil {
		return -1
	}
	s := strings.TrimSpace(ed.Text())
	if s == "" {
		return -1
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return -1
	}
	if n < -1 {
		return -1
	}
	if n > 32767 {
		return 32767
	}
	return int32(n)
}
