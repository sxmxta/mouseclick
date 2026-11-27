package app

import (
	"fmt"
	"strings"
)

// 步骤动作类型
const (
	actClick = "click"
	actHold  = "hold"
	actDrag  = "drag"
	actWheel = "wheel"
)

// 序列循环方式
const (
	loopOnce    = "once"
	loopCount   = "count"
	loopForever = "forever"
)

const (
	maxStepRepeat = 1000
	maxStepDelay  = 60000
	maxStepHold   = 10000
	maxDragSteps  = 60
	maxWheelNotch = 30
	maxSeqSteps   = 300
)

// seqStep 序列中的一个动作。坐标使用屏幕坐标，-1 表示"不移动，用当前位置"。
type seqStep struct {
	Action  string `json:"action"` // click / hold / drag / wheel
	Button  string `json:"button"` // left / middle / right
	X       int32  `json:"x"`
	Y       int32  `json:"y"`
	X2      int32  `json:"x2"` // 拖拽终点
	Y2      int32  `json:"y2"`
	Param   int    `json:"param"`   // 长按=毫秒，拖拽=步数，滚轮=档数（正上负下）
	DelayMs int    `json:"delayMs"` // 本步执行完后的等待
	Repeat  int    `json:"repeat"`  // 本步重复次数
}

// seqConfig 序列配置
type seqConfig struct {
	Steps          []seqStep `json:"steps"`
	LoopMode       string    `json:"loopMode"` // once / count / forever
	LoopCount      int       `json:"loopCount"`
	MaxDurationSec int       `json:"maxDurationSec"` // 0 = 不限
	MaxClicks      int       `json:"maxClicks"`      // 0 = 不限
	StartDelayMs   int       `json:"startDelayMs"`
}

func (c seqConfig) normalized() seqConfig {
	if len(c.Steps) > maxSeqSteps {
		c.Steps = c.Steps[:maxSeqSteps]
	}
	for i := range c.Steps {
		c.Steps[i] = c.Steps[i].normalized()
	}
	switch c.LoopMode {
	case loopOnce, loopCount, loopForever:
	default:
		c.LoopMode = loopOnce
	}
	if c.LoopCount < 1 {
		c.LoopCount = 1
	}
	if c.LoopCount > 10000 {
		c.LoopCount = 10000
	}
	if c.MaxDurationSec < 0 {
		c.MaxDurationSec = 0
	}
	if c.MaxDurationSec > 86400 {
		c.MaxDurationSec = 86400
	}
	if c.MaxClicks < 0 {
		c.MaxClicks = 0
	}
	if c.MaxClicks > 100000000 {
		c.MaxClicks = 100000000
	}
	if c.StartDelayMs < 0 {
		c.StartDelayMs = 0
	}
	if c.StartDelayMs > 600000 {
		c.StartDelayMs = 600000
	}
	return c
}

func (s seqStep) normalized() seqStep {
	switch s.Action {
	case actClick, actHold, actDrag, actWheel:
	default:
		s.Action = actClick
	}
	s.Button = buttonName(parseButton(s.Button))
	s.X, s.Y = clampCoord(s.X), clampCoord(s.Y)
	s.X2, s.Y2 = clampCoord(s.X2), clampCoord(s.Y2)
	if s.DelayMs < 0 {
		s.DelayMs = 0
	}
	if s.DelayMs > maxStepDelay {
		s.DelayMs = maxStepDelay
	}
	if s.Repeat < 1 {
		s.Repeat = 1
	}
	if s.Repeat > maxStepRepeat {
		s.Repeat = maxStepRepeat
	}
	switch s.Action {
	case actHold:
		if s.Param < 0 {
			s.Param = 0
		}
		if s.Param > maxStepHold {
			s.Param = maxStepHold
		}
	case actDrag:
		if s.Param < 2 {
			s.Param = 12
		}
		if s.Param > maxDragSteps {
			s.Param = maxDragSteps
		}
	case actWheel:
		if s.Param == 0 {
			s.Param = 3
		}
		if s.Param > maxWheelNotch {
			s.Param = maxWheelNotch
		}
		if s.Param < -maxWheelNotch {
			s.Param = -maxWheelNotch
		}
	default: // click
		if s.Param < 0 {
			s.Param = 0
		}
		if s.Param > maxStepHold {
			s.Param = maxStepHold
		}
	}
	return s
}

func clampCoord(v int32) int32 {
	if v < -1 {
		return -1
	}
	if v > 32767 {
		return 32767
	}
	return v
}

// 位置是否有效（-1 表示沿用当前光标）
func (s seqStep) hasPos() bool { return s.X >= 0 && s.Y >= 0 }

func (s seqStep) hasEnd() bool { return s.X2 >= 0 && s.Y2 >= 0 }

func stepButtonLabel(name string) string {
	switch parseButton(name) {
	case btnMiddle:
		return "中键"
	case btnRight:
		return "右键"
	default:
		return "左键"
	}
}

func stepActionLabel(action string) string {
	switch action {
	case actHold:
		return "长按"
	case actDrag:
		return "拖拽"
	case actWheel:
		return "滚轮"
	default:
		return "点击"
	}
}

func (s seqStep) posText() string {
	if !s.hasPos() {
		return "当前位置"
	}
	return fmt.Sprintf("(%d,%d)", s.X, s.Y)
}

// listText 步骤列表中的一行
func (s seqStep) listText(idx int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%02d ", idx+1)
	switch s.Action {
	case actHold:
		fmt.Fprintf(&b, "长按(%s) %s %dms", stepButtonLabel(s.Button), s.posText(), s.Param)
	case actDrag:
		end := "未设终点"
		if s.hasEnd() {
			end = fmt.Sprintf("(%d,%d)", s.X2, s.Y2)
		}
		fmt.Fprintf(&b, "拖拽(%s) %s→%s %d步", stepButtonLabel(s.Button), s.posText(), end, s.Param)
	case actWheel:
		fmt.Fprintf(&b, "滚轮 %+d档 %s", s.Param, s.posText())
	default:
		fmt.Fprintf(&b, "点击(%s) %s", stepButtonLabel(s.Button), s.posText())
	}
	if s.DelayMs > 0 {
		fmt.Fprintf(&b, " · 间隔%dms", s.DelayMs)
	}
	if s.Repeat > 1 {
		fmt.Fprintf(&b, " · ×%d", s.Repeat)
	}
	return b.String()
}

func defaultSeqStep() seqStep {
	return seqStep{
		Action:  actClick,
		Button:  "left",
		X:       -1,
		Y:       -1,
		X2:      -1,
		Y2:      -1,
		Param:   0,
		DelayMs: 50,
		Repeat:  1,
	}
}

func defaultSeqConfig() seqConfig {
	return seqConfig{
		Steps:     []seqStep{},
		LoopMode:  loopOnce,
		LoopCount: 1,
	}
}
