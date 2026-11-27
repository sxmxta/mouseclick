package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const configFileName = "config.json"

const (
	modeSingle = "single" // 单击模式：按固定间隔连续点击
	modeDouble = "double" // 双击模式：每组连续点击两次，再等待组间隔
	modeScript = "script" // 脚本模式：按动作序列执行
)

// appConfig 小工具的可持久化配置
type appConfig struct {
	Rate           int    `json:"rate"`             // 单击模式点击速度，次/秒
	Button         string `json:"button"`           // left / middle / right
	Hotkey         string `json:"hotkey"`           // 热键名称，如 F8、Ctrl+F8
	RecordHotkey   string `json:"recordHotkey"`     // 录制热键，默认 F10
	Mode           string `json:"mode"`             // single / double
	Double         bool   `json:"double,omitempty"` // 旧版双击开关，仅用于迁移
	DoubleGap      int    `json:"doubleGap"`        // 双击两次点击之间的间隔，毫秒
	DoubleInterval int    `json:"doubleInterval"`   // 双击每组结束后的等待，毫秒
	Lock           bool   `json:"lock"`             // 是否锁定点击位置
	LockX          int32  `json:"lockX"`            // 锁定的屏幕坐标
	LockY          int32  `json:"lockY"`
	CloseToTray    bool   `json:"closeToTray"` // 点击窗口 X 时隐藏到托盘；否则退出

	Sequence seqConfig `json:"sequence"` // 脚本（序列）模式
}

func defaultConfig() appConfig {
	return appConfig{
		Rate:           defaultRate,
		Button:         buttonName(btnLeft),
		Hotkey:         hotkeyOptions[defaultHotkeyIndex].name,
		RecordHotkey:   hotkeyOptions[defaultRecordHotkeyIndex].name,
		Mode:           modeSingle,
		DoubleGap:      defaultDoubleGap,
		DoubleInterval: defaultDoubleInterval,
		Sequence:       defaultSeqConfig(),
	}
}

func buttonName(b clickButton) string {
	switch b {
	case btnMiddle:
		return "middle"
	case btnRight:
		return "right"
	default:
		return "left"
	}
}

func parseButton(name string) clickButton {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "middle", "中键", "中":
		return btnMiddle
	case "right", "右键", "右":
		return btnRight
	default:
		return btnLeft
	}
}

func hotkeyIndex(name string) int {
	for i, opt := range hotkeyOptions {
		if strings.EqualFold(strings.TrimSpace(name), opt.name) {
			return i
		}
	}
	return defaultHotkeyIndex
}

// recordHotkeyIndex 录制热键的下标，空值或缺省时回退到默认 F10
func recordHotkeyIndex(name string) int {
	name = strings.TrimSpace(name)
	if name == "" {
		return defaultRecordHotkeyIndex
	}
	for i, opt := range hotkeyOptions {
		if strings.EqualFold(name, opt.name) {
			return i
		}
	}
	return defaultRecordHotkeyIndex
}

// pickRecordHotkey 从默认项开始找第一个不与启停热键重名的选项
func pickRecordHotkey(exclude string) string {
	for i := 0; i < len(hotkeyOptions); i++ {
		idx := (defaultRecordHotkeyIndex + i) % len(hotkeyOptions)
		if hotkeyOptions[idx].name != exclude {
			return hotkeyOptions[idx].name
		}
	}
	return hotkeyOptions[defaultRecordHotkeyIndex].name
}

// normalized 校验并回退非法值，避免手改配置后程序异常
func (c appConfig) normalized() appConfig {
	if c.Rate < minRate || c.Rate > maxRate {
		c.Rate = defaultRate
	}
	c.Button = buttonName(parseButton(c.Button))
	idx := hotkeyIndex(c.Hotkey)
	c.Hotkey = hotkeyOptions[idx].name
	recIdx := recordHotkeyIndex(c.RecordHotkey)
	c.RecordHotkey = hotkeyOptions[recIdx].name
	if c.RecordHotkey == c.Hotkey {
		c.RecordHotkey = pickRecordHotkey(c.Hotkey)
	}
	if c.Mode != modeSingle && c.Mode != modeDouble && c.Mode != modeScript {
		if c.Double {
			c.Mode = modeDouble
		} else {
			c.Mode = modeSingle
		}
	}
	c.Double = false
	if c.DoubleGap < minDoubleGap {
		c.DoubleGap = defaultDoubleGap
	}
	if c.DoubleGap > maxDoubleGap {
		c.DoubleGap = maxDoubleGap
	}
	if c.DoubleInterval < minDoubleInterval {
		c.DoubleInterval = defaultDoubleInterval
	}
	if c.DoubleInterval > maxDoubleInterval {
		c.DoubleInterval = maxDoubleInterval
	}
	if !c.Lock {
		c.LockX, c.LockY = 0, 0
	}
	c.Sequence = c.Sequence.normalized()
	return c
}

// appDataDir 当前应用的数据目录。
// Windows 使用 %LOCALAPPDATA%\<应用名>，与框架 wv 的应用数据目录规则一致；
// macOS/Linux 使用系统配置目录。应用名取设计器注入的 app.Info.Name，手动构建时回退到 exe 名。
func appDataDir() string {
	var base string
	if runtime.GOOS == "windows" {
		base = os.Getenv("LOCALAPPDATA")
		if base == "" {
			if home, err := os.UserHomeDir(); err == nil {
				base = filepath.Join(home, "AppData", "Local")
			}
		}
	} else {
		base, _ = os.UserConfigDir()
	}
	if base == "" {
		base = "."
	}

	name := strings.TrimSpace(Info.Name)
	if name == "" {
		if exe, err := os.Executable(); err == nil {
			name = strings.TrimSuffix(filepath.Base(exe), filepath.Ext(exe))
		}
	}
	if name == "" {
		name = "mouseclick"
	}
	// 避免应用名里的路径分隔符把配置写到目录外
	name = strings.NewReplacer("\\", "-", "/", "-", ":", "-").Replace(name)
	return filepath.Join(base, name)
}

func configPath() string {
	return filepath.Join(appDataDir(), configFileName)
}

func loadAppConfig() appConfig {
	cfg := defaultConfig()
	data, err := os.ReadFile(configPath())
	if err != nil {
		return cfg
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return defaultConfig()
	}
	return cfg.normalized()
}

func saveAppConfig(cfg appConfig) error {
	if err := os.MkdirAll(appDataDir(), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	path := configPath()
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// configStore 缓存一份 UI 线程的配置，防抖写盘；关闭窗口时强制 flush
type configStore struct {
	mu      sync.Mutex
	writeMu sync.Mutex
	data    appConfig
	timer   *time.Timer
}

func (s *configStore) set(cfg appConfig) {
	s.mu.Lock()
	s.data = cfg
	s.mu.Unlock()
}

func (s *configStore) update(fn func(*appConfig)) {
	s.mu.Lock()
	fn(&s.data)
	s.mu.Unlock()
	s.schedule()
}

func (s *configStore) schedule() {
	s.mu.Lock()
	if s.timer != nil {
		s.timer.Stop()
	}
	s.timer = time.AfterFunc(300*time.Millisecond, s.flush)
	s.mu.Unlock()
}

func (s *configStore) flush() {
	s.mu.Lock()
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
	cfg := s.data
	s.mu.Unlock()

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_ = saveAppConfig(cfg)
}
