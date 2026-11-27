package app

import (
	"sync"
	"sync/atomic"
	"time"
)

const (
	limitNone     = 0
	limitDuration = 1
	limitClicks   = 2
	limitEscape   = 3
	limitDone     = 4
)

// sleepGuard 的返回码
const (
	guardOK     = 0
	guardStop   = 1
	guardEscape = 2
)

// runner 序列执行器：配置由 UI 线程写入，执行在独立 goroutine
type runner struct {
	mu      sync.Mutex
	active  bool
	timerOn bool
	stopCh  chan struct{}
	doneCh  chan struct{}

	stateVal   atomic.Int32
	clickCnt   atomic.Int64
	stepNo     atomic.Int32
	stepAll    atomic.Int32
	loopNo     atomic.Int32
	stopReason atomic.Int32
}

func (r *runner) start(cfg seqConfig) bool {
	cfg = cfg.normalized()
	if len(cfg.Steps) == 0 {
		return false
	}

	r.mu.Lock()
	if r.active {
		r.mu.Unlock()
		return false
	}
	stop := make(chan struct{})
	done := make(chan struct{})
	r.stopCh, r.doneCh = stop, done
	r.active = true
	r.clickCnt.Store(0)
	r.stepNo.Store(0)
	r.stepAll.Store(int32(len(cfg.Steps)))
	r.loopNo.Store(1)
	r.stopReason.Store(limitNone)
	r.stateVal.Store(int32(stateArming))
	r.mu.Unlock()

	enableHighResTimer()
	r.mu.Lock()
	r.timerOn = true
	r.mu.Unlock()

	go r.loop(stop, done, cfg)
	return true
}

// stop 停止序列并等待执行循环退出，保证不会把鼠标键卡在按下状态
func (r *runner) stop() {
	r.mu.Lock()
	if !r.active {
		r.mu.Unlock()
		return
	}
	r.active = false
	stop, done := r.stopCh, r.doneCh
	r.stopCh, r.doneCh = nil, nil
	close(stop)
	r.mu.Unlock()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}
	r.releaseTimer()
	r.stateVal.Store(int32(stateStopped))
}

func (r *runner) releaseTimer() {
	r.mu.Lock()
	on := r.timerOn
	r.timerOn = false
	r.mu.Unlock()
	if on {
		disableHighResTimer()
	}
}

func (r *runner) state() runState {
	return runState(r.stateVal.Load())
}

// clearResult 清掉上一次运行的进度与停止原因，用于切换模式时复位提示
func (r *runner) clearResult() {
	r.clickCnt.Store(0)
	r.stepNo.Store(0)
	r.stopReason.Store(limitNone)
}

func (r *runner) clicks() int64 {
	return r.clickCnt.Load()
}

func (r *runner) progress() (step, total, loop int32, reason int32) {
	return r.stepNo.Load(), r.stepAll.Load(), r.loopNo.Load(), r.stopReason.Load()
}

func (r *runner) setStopReason(code int) {
	if code == guardEscape {
		r.stopReason.Store(limitEscape)
	} else {
		r.stopReason.Store(limitNone)
	}
}

func (r *runner) loop(stop chan struct{}, done chan struct{}, cfg seqConfig) {
	defer func() {
		r.mu.Lock()
		r.active = false
		r.stopCh, r.doneCh = nil, nil
		r.mu.Unlock()
		r.stateVal.Store(int32(stateStopped))
		close(done)
		r.releaseTimer()
	}()

	if code := sleepGuard(stop, time.Duration(cfg.StartDelayMs)*time.Millisecond); code != guardOK {
		r.setStopReason(code)
		return
	}

	startAt := time.Now()
	r.stateVal.Store(int32(stateRunning))
	loops := 0

	for {
		if code := stopCode(stop); code != guardOK {
			r.setStopReason(code)
			return
		}

		for i := range cfg.Steps {
			st := cfg.Steps[i]
			r.stepNo.Store(int32(i + 1))
			for rep := 0; rep < st.Repeat; rep++ {
				if !r.execStep(stop, st) {
					return
				}
				if code := sleepGuard(stop, time.Duration(st.DelayMs)*time.Millisecond); code != guardOK {
					r.setStopReason(code)
					return
				}
				if code := r.limitCode(startAt, cfg); code != limitNone {
					r.stopReason.Store(int32(code))
					return
				}
			}
		}

		loops++
		switch cfg.LoopMode {
		case loopOnce:
			r.stopReason.Store(limitDone)
			return
		case loopCount:
			if loops >= cfg.LoopCount {
				r.stopReason.Store(limitDone)
				return
			}
		}
		r.loopNo.Store(int32(loops + 1))
	}
}

func (r *runner) limitCode(startAt time.Time, cfg seqConfig) int {
	if cfg.MaxDurationSec > 0 && time.Since(startAt) >= time.Duration(cfg.MaxDurationSec)*time.Second {
		return limitDuration
	}
	if cfg.MaxClicks > 0 && int(r.clickCnt.Load()) >= cfg.MaxClicks {
		return limitClicks
	}
	return limitNone
}

// execStep 执行一个动作，返回 false 表示被停止
func (r *runner) execStep(stop <-chan struct{}, st seqStep) bool {
	btn := parseButton(st.Button)

	switch st.Action {
	case actHold:
		if st.hasPos() {
			setCursorPos(st.X, st.Y)
		}
		hold := st.Param
		if hold < 5 {
			hold = 5
		}
		mouseDown(btn)
		if code := sleepGuard(stop, time.Duration(hold)*time.Millisecond); code != 0 {
			mouseUp(btn)
			r.setStopReason(code)
			return false
		}
		mouseUp(btn)
		r.clickCnt.Add(1)

	case actClick:
		if st.hasPos() {
			setCursorPos(st.X, st.Y)
		}
		hold := st.Param
		if hold <= 0 {
			hold = 30
		}
		mouseDown(btn)
		if code := sleepGuard(stop, time.Duration(hold)*time.Millisecond); code != 0 {
			mouseUp(btn)
			r.setStopReason(code)
			return false
		}
		mouseUp(btn)
		r.clickCnt.Add(1)

	case actDrag:
		// 需要起点和终点，缺一个就跳过
		if !st.hasEnd() || !st.hasPos() {
			return true
		}
		steps := st.Param
		if steps < 2 {
			steps = 2
		}
		setCursorPos(st.X, st.Y)
		mouseDown(btn)
		for i := 1; i <= steps; i++ {
			x := st.X + (st.X2-st.X)*int32(i)/int32(steps)
			y := st.Y + (st.Y2-st.Y)*int32(i)/int32(steps)
			setCursorPos(x, y)
			if code := sleepGuard(stop, 25*time.Millisecond); code != 0 {
				mouseUp(btn)
				r.setStopReason(code)
				return false
			}
		}
		mouseUp(btn)
		r.clickCnt.Add(1)

	case actWheel:
		if st.hasPos() {
			setCursorPos(st.X, st.Y)
		}
		mouseWheel(int32(st.Param) * 120)
	}
	return true
}

func stopCode(stop <-chan struct{}) int {
	select {
	case <-stop:
		return guardStop
	default:
	}
	if escapePressed() {
		return guardEscape
	}
	return guardOK
}

// sleepGuard 分片睡眠，期间响应停止信号与 Esc 急停。返回 0 正常，1 停止，2 Esc
func sleepGuard(stop <-chan struct{}, d time.Duration) int {
	if d <= 0 {
		return stopCode(stop)
	}
	deadline := time.Now().Add(d)
	for {
		remain := time.Until(deadline)
		if remain <= 0 {
			return guardOK
		}
		chunk := 20 * time.Millisecond
		if remain < chunk {
			chunk = remain
		}
		timer := time.NewTimer(chunk)
		select {
		case <-stop:
			timer.Stop()
			return guardStop
		case <-timer.C:
		}
		if escapePressed() {
			return guardEscape
		}
	}
}
