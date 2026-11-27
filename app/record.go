package app

import (
	"sync"
)

// 录制参数：与手工编辑的步骤表共用同一套数据模型
const (
	recMaxDelayMs   = 60000 // 单步最长间隔
	recClickMergeMs = 350   // 同点连击合并成重复的次数
	recWheelMergeMs = 400   // 同向滚轮合并档数
	recDragStepMs   = 25    // 与执行器的每步耗时一致，用来把拖拽时长换算成步数
	recDragMinPx    = 5     // 位移小于该值算点击
	recMinHoldMs    = 10
)

// recItem 录制得到的一步，结束时再转换成 seqStep
type recItem struct {
	button  clickButton
	action  string
	x, y    int32
	x2, y2  int32
	param   int
	delayMs int
	repeat  int
	at      uint32 // 该动作结束的时刻（钩子时间戳，毫秒）
}

// recorder 录制状态机：只认鼠标按下 / 抬起 / 滚轮，移动过程不记
type recorder struct {
	mu      sync.Mutex
	items   []recItem
	lastEnd uint32
	down    bool
	downBtn clickButton
	downX   int32
	downY   int32
	downAt  uint32
	wheelAt uint32
}

func (r *recorder) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = nil
	r.lastEnd = 0
	r.down = false
	r.wheelAt = 0
}

func (r *recorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.items)
}

// steps 把录制结果转成步骤表
func (r *recorder) steps() []seqStep {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]seqStep, 0, len(r.items))
	for _, it := range r.items {
		st := seqStep{
			Action:  it.action,
			Button:  string(it.button),
			X:       it.x,
			Y:       it.y,
			X2:      it.x2,
			Y2:      it.y2,
			Param:   it.param,
			DelayMs: it.delayMs,
			Repeat:  it.repeat,
		}
		out = append(out, st)
	}
	return out
}

// 录制事件类型，由平台层换算成具体消息
const (
	recEvDown = iota
	recEvUp
	recEvWheel
)

func tickGap(from, to uint32) int32 {
	d := to - from // 无符号相减，回绕时仍然正确
	if d > 0x7fffffff {
		return 0
	}
	return int32(d)
}

func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// onEvent 钩子回调里调用，只做记录，不碰界面
func (r *recorder) onEvent(kind int, btn clickButton, x, y int32, notches int, at uint32) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	switch kind {
	case recEvDown:
		if r.down { // 丢了抬起事件时，按上一次按下收尾
			return
		}
		r.down = true
		r.downBtn = btn
		r.downX, r.downY, r.downAt = x, y, at
		r.wheelAt = 0

	case recEvUp:
		if !r.down || btn != r.downBtn {
			return
		}
		r.down = false
		dur := tickGap(r.downAt, at)
		delay := r.delayAt(r.downAt)
		dx, dy := x-r.downX, y-r.downY
		if abs32(dx) <= recDragMinPx && abs32(dy) <= recDragMinPx {
			// 原地按下抬起：点击，按下时长取实际值
			item := recItem{
				button:  r.downBtn,
				action:  actClick,
				x:       r.downX,
				y:       r.downY,
				param:   clampInt(int(dur), recMinHoldMs, maxStepHold),
				delayMs: int(delay),
				repeat:  1,
				at:      at,
			}
			if len(r.items) > 0 {
				last := &r.items[len(r.items)-1]
				if last.action == actClick && last.button == item.button &&
					abs32(last.x-item.x) <= 2 && abs32(last.y-item.y) <= 2 &&
					tickGap(last.at, r.downAt) <= recClickMergeMs {
					last.repeat = clampInt(last.repeat+1, 1, maxStepRepeat)
					last.at = at
					r.lastEnd = at
					return
				}
			}
			r.items = append(r.items, item)
			r.lastEnd = at
			return
		}
		// 带位移：拖拽，按实际耗时换算步数
		item := recItem{
			button:  r.downBtn,
			action:  actDrag,
			x:       r.downX,
			y:       r.downY,
			x2:      x,
			y2:      y,
			param:   clampInt(int(dur)/recDragStepMs, 2, maxDragSteps),
			delayMs: int(delay),
			repeat:  1,
			at:      at,
		}
		r.items = append(r.items, item)
		r.lastEnd = at

	case recEvWheel:
		if notches == 0 {
			return
		}
		if len(r.items) > 0 {
			last := &r.items[len(r.items)-1]
			sameDir := (last.param > 0) == (notches > 0)
			if last.action == actWheel && sameDir && tickGap(r.wheelAt, at) <= recWheelMergeMs {
				last.param = clampInt(last.param+notches, -maxWheelNotch, maxWheelNotch)
				last.at = at
				r.lastEnd = at
				r.wheelAt = at
				return
			}
		}
		item := recItem{
			button:  btnLeft,
			action:  actWheel,
			x:       x,
			y:       y,
			param:   clampInt(notches, -maxWheelNotch, maxWheelNotch),
			delayMs: int(r.delayAt(at)),
			repeat:  1,
			at:      at,
		}
		r.items = append(r.items, item)
		r.lastEnd = at
		r.wheelAt = at
	}
}

// delayAt 计算与上一步结束之间的间隔
func (r *recorder) delayAt(now uint32) int {
	if len(r.items) == 0 {
		return 0
	}
	return clampInt(int(tickGap(r.lastEnd, now)), 0, recMaxDelayMs)
}

// dropLastClickInside 停止录制时，去掉落在本窗口内的结尾点击（多半是点了「停止」）
func (r *recorder) dropLastClickInside(inside func(x, y int32) bool) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.items) == 0 || inside == nil {
		return false
	}
	last := r.items[len(r.items)-1]
	if last.action != actClick || !inside(last.x, last.y) {
		return false
	}
	r.items = r.items[:len(r.items)-1]
	r.lastEnd = 0
	if len(r.items) > 0 {
		r.lastEnd = r.items[len(r.items)-1].at
	}
	return true
}
