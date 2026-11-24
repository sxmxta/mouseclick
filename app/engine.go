package app

import (
	"sync"
	"sync/atomic"
	"time"
)

// clickButton 连点的鼠标按键
type clickButton int32

const (
	btnLeft clickButton = iota
	btnMiddle
	btnRight
)

// runState 连点器运行状态
type runState int32

const (
	stateStopped runState = iota // 已暂停
	stateArming                  // 预热中（给用户时间把鼠标移回游戏）
	stateRunning                 // 连点中
)

const (
	armDelay = 800 * time.Millisecond // 按下开始后的预热时间
	minHold  = 5 * time.Millisecond   // 按键最短保持
	maxHold  = 25 * time.Millisecond  // 按键最长保持

	minDoubleGap      = 30   // 双击间隔下限，毫秒
	maxDoubleGap      = 300  // 双击间隔上限，毫秒
	minDoubleInterval = 20   // 双击组间隔下限，毫秒
	maxDoubleInterval = 1000 // 双击组间隔上限，毫秒
)

// clicker 连点引擎：配置由 UI 线程写入，点击循环在独立 goroutine 中运行
type clicker struct {
	mu             sync.Mutex
	active         bool
	stopCh         chan struct{}
	doneCh         chan struct{}
	stateVal       atomic.Int32
	clickCount     atomic.Int64
	button         atomic.Int32
	interv         atomic.Int64 // 点击间隔，纳秒
	fixed          atomic.Bool
	fixedX         atomic.Int32
	fixedY         atomic.Int32
	double         atomic.Bool
	doubleGap      atomic.Int64 // 双击两次点击之间的间隔，纳秒
	doubleInterval atomic.Int64 // 双击每组结束后的等待，纳秒
}

func (c *clicker) init(rate int) {
	c.setRate(rate)
	c.button.Store(int32(btnLeft))
	c.stateVal.Store(int32(stateStopped))
	c.double.Store(false)
	c.setDoubleGap(defaultDoubleGap)
	c.setDoubleInterval(defaultDoubleInterval)
}

func (c *clicker) setRate(rate int) {
	if rate < 1 {
		rate = 1
	}
	if rate > 1000 {
		rate = 1000
	}
	iv := time.Second / time.Duration(rate)
	if iv < time.Millisecond {
		iv = time.Millisecond
	}
	c.interv.Store(int64(iv))
}

func (c *clicker) setButton(b clickButton) {
	c.button.Store(int32(b))
}

func (c *clicker) setFixed(on bool, x, y int32) {
	c.fixedX.Store(x)
	c.fixedY.Store(y)
	c.fixed.Store(on)
}

// setDouble 切换单击/双击模式：双击模式每组连续点击两次
func (c *clicker) setDouble(on bool) {
	c.double.Store(on)
}

// setDoubleGap 设置双击两次点击之间的间隔（毫秒）
func (c *clicker) setDoubleGap(ms int) {
	if ms < minDoubleGap {
		ms = minDoubleGap
	}
	if ms > maxDoubleGap {
		ms = maxDoubleGap
	}
	c.doubleGap.Store(int64(ms) * int64(time.Millisecond))
}

// setDoubleInterval 设置双击模式每组结束后的等待（毫秒）
func (c *clicker) setDoubleInterval(ms int) {
	if ms < minDoubleInterval {
		ms = minDoubleInterval
	}
	if ms > maxDoubleInterval {
		ms = maxDoubleInterval
	}
	c.doubleInterval.Store(int64(ms) * int64(time.Millisecond))
}

func (c *clicker) state() runState {
	return runState(c.stateVal.Load())
}

func (c *clicker) clicks() int64 {
	return c.clickCount.Load()
}

// start 开始连点；已在运行时直接返回
func (c *clicker) start() bool {
	c.mu.Lock()
	if c.active {
		c.mu.Unlock()
		return false
	}
	stop := make(chan struct{})
	done := make(chan struct{})
	c.stopCh, c.doneCh = stop, done
	c.active = true
	c.clickCount.Store(0)
	c.stateVal.Store(int32(stateArming))
	c.mu.Unlock()

	enableHighResTimer()
	go c.loop(stop, done)
	return true
}

// stop 停止连点并等待循环退出（保证不会把鼠标按键卡在按下状态）
func (c *clicker) stop() {
	c.mu.Lock()
	if !c.active {
		c.mu.Unlock()
		return
	}
	c.active = false
	stop := c.stopCh
	done := c.doneCh
	c.stopCh, c.doneCh = nil, nil
	close(stop)
	c.mu.Unlock()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}
	disableHighResTimer()
	c.stateVal.Store(int32(stateStopped))
}

func (c *clicker) loop(stop, done chan struct{}) {
	defer close(done)

	arm := time.NewTimer(armDelay)
	select {
	case <-stop:
		arm.Stop()
		c.stateVal.Store(int32(stateStopped))
		return
	case <-arm.C:
	}

	c.stateVal.Store(int32(stateRunning))
	next := time.Now()

	for {
		select {
		case <-stop:
			c.stateVal.Store(int32(stateStopped))
			return
		default:
		}

		if !c.clickOnce(stop) {
			c.stateVal.Store(int32(stateStopped))
			return
		}

		if c.double.Load() {
			// 双击模式：第二下紧跟上，然后等待组间隔
			if !sleepFor(stop, time.Duration(c.doubleGap.Load())) {
				c.stateVal.Store(int32(stateStopped))
				return
			}
			if !c.clickOnce(stop) {
				c.stateVal.Store(int32(stateStopped))
				return
			}
			if !sleepFor(stop, time.Duration(c.doubleInterval.Load())) {
				c.stateVal.Store(int32(stateStopped))
				return
			}
			next = time.Now()
			continue
		}

		iv := time.Duration(c.interv.Load())
		next = next.Add(iv)
		if now := time.Now(); next.Before(now) {
			next = now
		}
		if !sleepUntil(stop, next) {
			c.stateVal.Store(int32(stateStopped))
			return
		}
	}
}

// clickOnce 完成一次按下/保持/抬起；被暂停时先抬键再返回 false
func (c *clicker) clickOnce(stop <-chan struct{}) bool {
	btn := clickButton(c.button.Load())
	if c.fixed.Load() {
		setCursorPos(c.fixedX.Load(), c.fixedY.Load())
	}

	iv := time.Duration(c.interv.Load())
	hold := iv / 3
	if hold > maxHold {
		hold = maxHold
	}
	if hold < minHold {
		hold = minHold
	}

	mouseDown(btn)
	if !sleepFor(stop, hold) {
		mouseUp(btn)
		return false
	}
	mouseUp(btn)
	c.clickCount.Add(1)
	return true
}

func sleepFor(stop <-chan struct{}, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-stop:
		return false
	case <-timer.C:
		return true
	}
}

func sleepUntil(stop <-chan struct{}, t time.Time) bool {
	d := time.Until(t)
	if d <= 0 {
		return true
	}
	return sleepFor(stop, d)
}
