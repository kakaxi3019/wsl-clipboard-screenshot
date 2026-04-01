# wsl-clipboard-screenshot

在 WSL 中使用 Claude Code 等工具时，无法直接将 Windows 截图粘贴到终端。本工具监控 Windows 剪贴板，检测到截图后自动保存，粘贴时输出文件路径。

## 特性

- **零配置安装** - 一条命令即可安装
- **Windows 通知** - 截图保存成功后弹窗通知
- **剪贴板互通** - Windows 和 WSL 粘贴都能用
- **内容去重** - 相同图片不重复保存
- **自动清理** - 启动时自动清理 7 天前的截图

## 需求

- WSL (Windows Subsystem for Linux)
- Windows 10/11
- PowerShell

## 安全说明

- **仅本地操作** - 所有数据保存在本地，不上传任何内容
- **仅读提取贴板** - 只读取剪贴板中的图片，不访问其他数据
- **无系统侵入** - 不修改系统文件，不安装驱动
- **最小权限** - 仅使用必要的 Windows API

## 安装

在 **WSL 终端** 中执行：

```bash
curl -fsSL https://raw.githubusercontent.com/kakaxi3019/wsl-clipboard-screenshot/main/scripts/install.sh | bash
```

## 快速开始

在 **WSL 终端** 中执行：

```bash
# 启动守护进程
wsl-clipboard-screenshot start --daemon

# Windows 上按 Win+Shift+S 截图
# 在 WSL 终端按 Ctrl+V 粘贴
# 输出文件路径: /tmp/.wsl-clipboard-screenshot/<hash>.png
```

## 命令

```bash
wsl-clipboard-screenshot start --daemon              # 启动守护进程（后台运行，默认开启通知）
wsl-clipboard-screenshot start --verbose              # 前台运行（调试用）
wsl-clipboard-screenshot start --notify=false         # 关闭 Windows 通知
wsl-clipboard-screenshot status                       # 查看状态
wsl-clipboard-screenshot stop                        # 停止守护进程
wsl-clipboard-screenshot update                       # 检查更新
```

**通知说明**：截图保存成功后，Windows 会弹窗显示文件路径。

## Claude Code 集成

编辑 `~/.claude/settings.json`：

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

## 参考

本项目参考了 [wsl-screenshot-cli](https://github.com/Nailuu/wsl-screenshot-cli) 的实现。

## License

MIT
