<p align="center">
  <a href="resources/logo/mouseclick-neon.svg">
    <img src="resources/logo/mouseclick-neon-preview.png" width="320" alt="鼠标连点器霓虹 Logo">
  </a>
</p>

# 鼠标连点器（mouseclick）

一个运行在 Windows 桌面的鼠标连点工具：支持左键 / 中键 / 右键，支持单击与双击两种节奏，支持全局热键启停、锁定点击位置和配置记忆。界面由 **ENERGY GUI Designer + LCL** 构建，业务逻辑使用 Go。

> 适合重复点击、批量确认、界面测试等需要固定节奏点击的场景。请只在你有权操作的窗口和程序中使用。

## 界面预览

| 单击模式（默认 20 次/秒） | 双击模式（双击间隔 100 ms / 组间隔 300 ms） |
|:---:|:---:|
| <img src="docs/images/single-mode.png" width="340" alt="单击模式界面"> | <img src="docs/images/double-mode.png" width="340" alt="双击模式界面"> |

截图在运行中抓取，使用全新配置、未锁定点击位置；右上角是实际累计点击次数（单图 34 次，双图 8 次）。开始后会有 0.8 秒预热，状态从「即将开始」变为「运行中」。

## 功能

- **鼠标按键**：左键、中键、右键三选一。
- **单击模式**：按固定间隔连续点击，速度范围 1–50 次/秒，默认 20 次/秒（间隔 50 ms）。
- **双击模式**：每组连续点击两次后等待，再进入下一组。
  - 双击间隔：两次点击之间，范围 30–300 ms，默认 100 ms。
  - 组间隔：每组两次点击后等待，范围 20–1000 ms，默认 300 ms。
- **模式独立参数**：单击速度与双击间隔 / 组间隔分别保存，切换模式不会覆盖另一组参数。
- **全局热键启停**：默认 `F8`，可在下拉框选择 `Ctrl+F8`、`Ctrl+F9`、`Ctrl+F10`、`Alt+F8`、`Alt+F9`、`F8`–`F12`。
- **锁定点击位置**：勾选时把当前鼠标位置固定为点击点；运行中光标会被固定在该位置，适合点固定按钮。
- **配置记忆**：按键、模式、速度、双击参数、热键、锁定状态、坐标和「关闭隐藏窗口」状态都会写入本地 JSON，下次启动自动恢复。
- **关闭到托盘**：勾选窗口右上角的「关闭隐藏窗口」后，点击窗口的 × 只隐藏窗口，点击引擎继续运行；未勾选时 × 直接退出。托盘菜单中的「退出程序」不受该复选框影响，始终直接退出。
- **状态反馈**：显示「已暂停 / 即将开始 / 运行中」、累计点击次数和热键注册失败提示。
- **安全退出**：真正退出前先释放鼠标按键，再结束点击循环，避免鼠标键卡住。

## 快速开始（通过 ENERGY GUI Designer 运行）

> 程序依赖 ENERGY GUI Designer 维护的 LCL / energy 运行时，请按下面的方式从 Designer 启动。

一、获取 [ENERGY GUI Designer](https://github.com/energye/designer)

1. 启动 **ENERGY GUI Designer**。
2. 打开项目文件 `mouseclick.egp`。
3. 点击工具栏上绿色三角形的 **「运行预览(F9)」**。
4. Designer 会加载 `github.com/energye/lcl`、`github.com/energye/energy/v3` 等依赖，并启动「鼠标连点器」窗口。
5. 把鼠标移到要点击的位置；选择鼠标按键和单击 / 双击模式，按需要调整速度或双击参数。
6. 按 `F8`（或下拉框中选择的其它热键）开始；再按一次暂停。

二、使用预构建的安装程序

说明：

- `F9` 是 Designer 的「运行预览」快捷键；本工具自己的启停热键默认是 `F8`，两者不冲突。
- Designer 运行预览时会自动编译并直接启动窗口，不需要额外执行构建命令，也不要把 `build/` 下的产物当作独立运行程序。
- 安装包和发布流程由你自己的 Designer 发布方式处理，本文不涉及打包步骤。

### 使用「锁定点击位置」

1. 先把鼠标移到目标位置。
2. 勾选「锁定点击位置」，程序会记录当前坐标。
3. 按热键开始后，光标会被固定在目标点。
4. 此时「开始 / 暂停」按钮可能被连点光标抢占，建议始终用全局热键暂停。

### 热键被占用

全局热键是系统独占的。若 `F8` 已被其他程序占用，窗口底部会显示红色提示；换一个热键即可。同时启动多个实例时，后启动的实例也会出现这个提示，这是正常现象。

## 依赖关系

程序入口和运行链路如下：

```text
main.go
  └─ lcl.Init() / lcl.Run(app.Forms...)
       └─ app.Form1（LCL 窗口和控件）
            ├─ app/form1.go       业务逻辑、事件、样式
            ├─ app/engine.go      连点引擎和状态机
            ├─ app/config.go      本地 JSON 配置
            └─ app/input_windows.go / hotkey_windows.go
                 └─ SendInput / RegisterHotKey（Windows 原生 API）
```

主要依赖：

| 依赖 | 版本 | 作用 |
|---|---|---|
| Go | 1.20+ | 编译和运行 |
| `github.com/energye/lcl` | `v1.0.9` | LCL 界面框架、窗口和原生控件 |
| `github.com/energye/energy/v3` | `v3.0.16` | `application/pack` 应用元信息、运行时集成 |
| `github.com/ebitengine/purego` | `v0.10.0`（间接） | LCL 依赖的动态库调用 |
| `libenergy-amd64.dll` | 由 ENERGY Designer 维护 | LCL 的 Windows 原生运行时 |

说明：

- 本机开发和运行由 ENERGY GUI Designer 维护 `%USERPROFILE%\.energy\runtime` 下的 `libenergy-amd64.dll`；本地 `build` 目录不需要手动复制 DLL。
- 如果要把程序分发到其他机器，请按你自己的 Designer 发布流程处理，确保运行时库与程序一起分发，不要只拷贝单个文件。
- macOS / Linux 目前只保留界面和配置逻辑的编译入口；鼠标注入和全局热键只在 Windows 下实现，不能实际点击。

## 在 Designer 中修改与运行

1. 用 ENERGY GUI Designer 打开 `mouseclick.egp`。
2. 界面布局在 `layouts/form1.ui` 中维护；打开 Designer 后可直接在画布上调整控件。
3. 修改后先保存，再点击 **「运行预览(F9)」** 编译并启动，验证界面和交互。
4. 业务逻辑在 `app/form1.go`，Designer 不会覆盖这部分；`app/form1.ui.go` 是生成代码，不要手改。
5. Designer 的运行输出目录是 `build/`，本仓库不提交这些构建产物。

如果只想做静态检查，可在项目根目录执行：

```bat
go vet ./...
```

## 配置记忆

配置文件路径：

```text
%LOCALAPPDATA%\mouseclick\config.json
```

目录名优先取 `app.Info.Name`；若 `Info.Name` 为空，回退为可执行文件名。macOS / Linux 使用 `os.UserConfigDir()` 的系统配置目录。

默认配置：

```json
{
  "rate": 20,
  "button": "left",
  "hotkey": "F8",
  "mode": "single",
  "doubleGap": 100,
  "doubleInterval": 300,
  "lock": false,
  "lockX": 0,
  "lockY": 0,
  "closeToTray": false
}
```

| 字段 | 说明 |
|---|---|
| `rate` | 单击模式速度，1–50 次/秒 |
| `button` | `left` / `middle` / `right` |
| `hotkey` | 热键名称，如 `F8`、`Ctrl+F8` |
| `mode` | `single`（单击）/ `double`（双击） |
| `doubleGap` | 双击两次点击之间的间隔，30–300 ms |
| `doubleInterval` | 双击每组结束后的组间隔，20–1000 ms |
| `lock` / `lockX` / `lockY` | 是否锁定点击位置及锁定的屏幕坐标 |
| `closeToTray` | 点击窗口 × 时是否只隐藏到托盘；`false` 为直接退出 |

速度滑块使用 300 ms 防抖写盘；按键、模式、热键、锁定项即时排队写入；关闭窗口时强制落盘。配置损坏或数值越界会回退到默认值，不影响启动。删除配置文件即可恢复全部默认值。
