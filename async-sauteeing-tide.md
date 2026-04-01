# wsl-clipboard-screenshot 实现计划

## Context

在 WSL 环境中使用 Claude Code CLI 时，无法直接将 Windows 剪贴板中的截图粘贴到终端。需要一个守护进程监控 Windows 剪贴板，检测到截图后自动保存到文件，并在终端输出文件路径。

参考实现：`/home/centos/mypro/github/wsl-screenshot-cli`（Go + PowerShell STA 架构）

## 项目信息

- **项目路径**：`/home/centos/mypro/wsl-screenshot-cli`
- **工具名称**：`wsl-clipboard-screenshot`
- **输出目录**：`/tmp/.wsl-clipboard-screenshot/`
- **PID 文件**：`/tmp/.wsl-clipboard-screenshot.pid`
- **日志文件**：`/tmp/.wsl-clipboard-screenshot.log`

---

## 架构设计

```
WSL CLI (Go) <--stdin/stdout pipe--> PowerShell STA <--.NET APIs--> Windows Clipboard
                                           |
                                    3-format enrichment:
                                    CF_UNICODETEXT (WSL path)
                                    CF_BITMAP (image)
                                    CF_HDROP (Windows UNC path)
```

### 核心原理

1. Go 程序通过 pipe 启动 PowerShell STA 线程
2. PowerShell 使用 `System.Windows.Forms.Clipboard` 监控剪贴板
3. 检测到新图片 → Go 计算 SHA256 → 保存为 `<hash>.png`
4. Go 通知 PowerShell 更新剪贴板：同时设置文本路径 + 原图 + 文件拖放列表
5. 用户在 WSL 终端粘贴 → 得到文件路径 `/tmp/.wsl-clipboard-screenshot/<hash>.png`

---

## 目录结构

```
wsl-clipboard-screenshot/
├── main.go                          # Entry point (signal handling)
├── go.mod / go.sum                  # Go module
├── Makefile                         # build/test/release targets
├── cmd/
│   ├── root.go                      # Root cobra command, --version
│   ├── start.go                     # start --daemon/--interval/--output
│   ├── stop.go                      # SIGTERM to PID
│   ├── status.go                    # diagnostics (PID/uptime/CPU%/mem/screenshots)
│   └── update.go                    # self-update via install.sh
├── internal/
│   ├── clipboard/
│   │   ├── clipboard.go              # Go <-> PowerShell pipe client
│   │   ├── clipboard.ps1             # Embedded PowerShell STA script
│   │   └── types.go                 # Command/Response constants
│   ├── daemon/
│   │   ├── daemon.go                # Daemonize(), Run(), Stop()
│   │   ├── pid.go                   # PID file read/write/cleanup
│   │   └── status.go                # /proc parsing for diagnostics
│   ├── poller/
│   │   ├── poller.go                # Poll loop with SHA256 dedup
│   │   └── hash.go                  # SHA256 content-addressable
│   ├── platform/
│   │   └── platform.go              # WSL/Interop environment checks
│   └── version/
│       └── check.go                 # GitHub releases API version check
├── scripts/
│   └── install.sh                   # 一键安装脚本
├── .goreleaser.yml
└── README.md
```

---

## 关键模块设计

### 1. CLI 命令 (`cmd/`)

| 命令 | 职责 | 关键 flags |
|------|------|------------|
| `start` | 启动监控 | `--interval` (默认250ms), `--output`, `--daemon`, `--verbose` |
| `stop` | 停止守护进程 | 无 |
| `status` | 诊断信息 | 无 |
| `update` | 自更新 | 无 |

### 2. PowerShell 通信协议 (`internal/clipboard/`)

**Go → PowerShell**：
```
CHECK\n                           # 检查剪贴板是否有新图片
UPDATE|<wslPath>|<winPath>\n    # 更新剪贴板（设置3种格式）
EXIT\n                             # 关闭 PowerShell 进程
```

**PowerShell → Go**：
```
READY\n                           # 启动完成
NONE\n                            # 无新图片
IMAGE\n<base64>\nEND\n      # 有新图片
OK\n                             # 更新成功
ERR|<message>\n                  # 错误
```

### 3. PowerShell 脚本要点 (`clipboard.ps1`)

- 使用 `Add-Type -AssemblyName System.Windows.Forms`（预编译 API，不触发 EDR）
- STA 线程 + `DoEvents()` 防止其他应用冻结
- 跳过 Excel/Sheets 复制单元格为图片的场景
- 自检测：剪贴板已有我们写入的三种格式时跳过

### 4. 三格式 Clipboard Enrichment

当检测到新截图并保存后，同时设置：

| 格式 | 内容 | 用途 |
|------|------|------|
| `CF_UNICODETEXT` | `/tmp/.wsl-clipboard-screenshot/<hash>.png` | WSL 终端粘贴得到路径 |
| `CF_BITMAP` | 原始图片字节 | Windows 应用粘贴得到图片 |
| `CF_HDROP` | `\\wsl$\Ubuntu\tmp\.wsl-clipboard-screenshot\<hash>.png` | 文件对话框粘贴 |

### 5. Poller 逻辑 (`internal/poller/`)

- `time.Ticker` 按 interval 轮询
- SHA256 内容寻址去重（hash 相同则不重复保存）
- 熔断机制：连续 5 次错误后重启 PowerShell 客户端
- 优雅关闭：context cancel 时清理资源

### 6. Daemon 逻辑 (`internal/daemon/`)

- `Daemonize()`：syscall.Setsid 后台运行
- PID 文件：`/tmp/.wsl-clipboard-screenshot.pid`
- 状态文件：`/tmp/.wsl-clipboard-screenshot.state`
- 清理：陈旧 PID 文件（WSL 重启后）自动删除

---

## 实现步骤

### Phase 1: 项目初始化
1. 创建目录结构
2. 初始化 Go module
3. 添加依赖：`spf13/cobra`
4. 实现 `main.go` + `cmd/root.go`

### Phase 2: 核心剪贴板通信
5. 实现 `internal/clipboard/types.go`（协议常量）
6. 实现 `internal/clipboard/clipboard.ps1`
7. 实现 `internal/clipboard/clipboard.go`（Go 端 pipe 客户端）
8. 测试 PowerShell 进程通信

### Phase 3: 轮询逻辑
9. 实现 `internal/poller/hash.go`（SHA256）
10. 实现 `internal/poller/poller.go`（轮询 + 去重 + 熔断）

### Phase 4: 守护进程
11. 实现 `internal/daemon/daemon.go`
12. 实现 `internal/daemon/pid.go`
13. 实现 `internal/daemon/status.go`
14. 添加 `cmd/start.go --daemon` 支持

### Phase 5: CLI 命令
15. 实现 `cmd/stop.go`
16. 实现 `cmd/status.go`
17. 实现 `cmd/update.go`（自更新）

### Phase 6: 平台检查
18. 实现 `internal/platform/platform.go`

### Phase 7: 安装与发布
19. 创建 `Makefile`
20. 创建 `.goreleaser.yml`
21. 创建 `scripts/install.sh`（一键安装 + Claude Code hooks）
22. 编写 `README.md`

---

## 可复用代码（从参考实现迁移）

| 参考文件 | 复用要点 |
|----------|----------|
| `internal/clipboard/clipboard.go` | Pipe I/O 模式、协议处理、超时控制 |
| `internal/clipboard/clipboard.ps1` | STA 线程、DoEvents、自检测逻辑 |
| `internal/poller/poller.go` | 轮询循环、SHA256 去重、熔断机制 |
| `internal/daemon/daemon.go` | Setsid 守护化、PID/State 文件 |
| `internal/daemon/status.go` | /proc 解析（uptime/CPU/mem） |
| `scripts/install.sh` | 平台检测、GitHub API、SHA256、Claude hooks |

---

## 验证计划

1. **编译验证**：`go build -o wsl-clipboard-screenshot .`
2. **单元测试**：`CGO_ENABLED=1 go test -race -count=1 -v ./...`
3. **手动测试**（Windows + WSL 环境下）：
   - 在 Windows 上截图（Win+Shift+S）
   - 运行 `wsl-clipboard-screenshot start --verbose`
   - 在 WSL 终端粘贴，验证输出文件路径
   - 检查 Windows 侧粘贴图片仍正常工作
4. **Claude Code 集成测试**：
   - 配置 hooks 到 `~/.claude/settings.json`
   - 启动 Claude Code，验证自动启动
   - 关闭 Claude Code，验证自动停止

---

## 使用说明

### 日常启动

```bash
# 启动守护进程（后台运行）
./wsl-clipboard-screenshot start --daemon

# 或前台运行（方便调试）
./wsl-clipboard-screenshot start --verbose
```

### 常用命令

```bash
./wsl-clipboard-screenshot status   # 查看守护进程状态
./wsl-clipboard-screenshot stop     # 停止守护进程
./wsl-clipboard-screenshot start --daemon  # 重新启动
./wsl-clipboard-screenshot update   # 检查并更新到最新版本
```

### 工作流程

1. 启动守护进程：`./wsl-clipboard-screenshot start --daemon`
2. Windows 上按 `Win+Shift+S` 截图
3. 在 WSL 终端按 `Ctrl+V` 粘贴
4. 输出文件路径：`/tmp/.wsl-clipboard-screenshot/<hash>.png`

### 自动清理

启动时会自动清理 **7 天前** 的截图文件，避免临时目录堆积。

### Windows 通知

截图保存成功后，会发送 Windows 通知显示文件名。通知实现优先级：
1. BurntToast PowerShell 模块
2. `notify` 命令
3. Windows.UI.Notifications API（静默失败，不影响主流程）

### 安装到系统

```bash
# 一键安装（零配置，安装后即可使用）
curl -fsSL https://raw.githubusercontent.com/kakaxi3019/wsl-clipboard-screenshot/master/scripts/install.sh | bash
```

**零配置特性**：
- 自动检测并添加到 PATH（优先使用 `/etc/profile.d/`，无需 sudo）
- 为当前会话自动 export PATH
- 自动检测已安装版本

### Claude Code Hooks 配置

在 Claude Code 启动时自动启动守护进程，关闭时自动停止。

编辑 `~/.claude/settings.json`，添加以下配置：

```json
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "wsl-clipboard-screenshot start --daemon 2>/dev/null || true"
          }
        ]
      }
    ]
  }
}
```

这样每次启动 Claude Code 时会自动启动截图监控。**注意**：不设置 SessionEnd hook，因为多个 Claude Code 共用一个守护进程，只有一个退出时不应该关闭守护进程。

### 停止守护进程

```bash
# 方式1：使用程序自带命令
./wsl-clipboard-screenshot stop

# 方式2：使用一键停止脚本
./scripts/stop.sh
```

### 构建

```bash
make build    # 编译
make test     # 运行测试
make snapshot # 发布快照
```

---

## CLAUDE.md 集成

此工具为独立 CLI，不依赖项目其他代码。相关命令：

```bash
go build -o wsl-clipboard-screenshot .  # 编译
./wsl-clipboard-screenshot start --daemon # 启动守护进程
./wsl-clipboard-screenshot status         # 查看状态
./wsl-clipboard-screenshot stop           # 停止
```

---

## 已实现功能

| 功能 | 状态 | 说明 |
|------|------|------|
| CLI 命令 (start/stop/status/update) | ✓ | 完整的 cobra 命令行界面 |
| PowerShell STA 剪贴板监控 | ✓ | 使用预编译 .NET API，兼容 EDR |
| 三格式 Clipboard Enrichment | ✓ | CF_UNICODETEXT + CF_BITMAP + CF_HDROP |
| SHA256 内容寻址去重 | ✓ | 相同内容不重复保存 |
| 熔断机制 | ✓ | 连续 5 次错误后重启 PowerShell |
| 守护进程化 | ✓ | syscalls.Setsid 后台运行 |
| PID/State 文件管理 | ✓ | 自动清理陈旧 PID 文件 |
| 自动清理 7 天前截图 | ✓ | 启动时清理 |
| Windows 通知 | ✓ | 截图保存后弹窗通知 |
| 零配置安装 | ✓ | 自动 PATH 配置，无需手动设置 |
| Claude Code Hooks 集成 | ✓ | SessionStart 自动启动 |

---

## 待实现功能

| 功能 | 优先级 | 说明 |
|------|--------|------|
| 系统托盘菜单 | 中 | 右键菜单（停止/打开目录/设置） |
| GUI 配置工具 | 低 | 交互式配置界面 |
| 配置文件支持 | 低 | `~/.config/wsl-clipboard-screenshot.toml` |
