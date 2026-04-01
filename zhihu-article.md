# 我写了一个解决 WSL 环境下无法粘贴 Windows 截图的工具

## 遇到了什么问题

在 Windows 上写代码时，我习惯用 Win+Shift+S 截图，然后用 Ctrl+V 粘贴到各种地方。

但最近我在 WSL（Windows Subsystem for Linux）里使用 Claude Code CLI 时遇到了麻烦：截图后按 Ctrl+V，终端里没有任何反应，什么都没发生。

这导致我没法让 AI 直接分析截图内容，每次都得：
1. 截图
2. 手动另存为到某个文件夹
3. 复制文件路径
4. 再粘贴到终端

非常麻烦。

## 这个工具能做什么

简单说就是：**在 WSL 里按 Ctrl+V 粘贴截图时，直接输出图片的文件路径**，就像在 Windows 原生环境里一样方便。

## 工作原理（简单解释）

程序会：
1. 在后台监控 Windows 的剪贴板
2. 发现有新的截图时，自动保存到电脑的临时目录
3. 把图片的保存路径复制到剪贴板，这样你粘贴时得到的就是文件路径
4. 截图只保存一份，不会重复占用空间
5. 超过 7 天的截图会自动清理掉

整个过程程序在后台运行，不影响你正常使用电脑。

## 安装和使用

**安装**（在 WSL 终端里粘贴这行命令回车）：

```bash
curl -fsSL https://raw.githubusercontent.com/kakaxi3019/wsl-clipboard-screenshot/master/scripts/install.sh | bash
```

**启动**：

```bash
wsl-clipboard-screenshot start --daemon
```

**之后正常使用就行**：
1. Windows 上按 Win+Shift+S 截图
2. 在 WSL 终端按 Ctrl+V
3. 终端里会显示截图的保存路径

截图保存成功后，Windows 右下角会弹个通知，显示截图的保存路径。

## 常见问题

**这个工具安全吗？**
安全的。程序只读取剪贴板里的图片，不会访问你电脑上的其他文件，也不会上传任何数据到网上。

**会影响电脑性能吗？**
基本不会。程序很轻量，只在后台默默运行，占用很少的内存和 CPU。

**需要一直开着吗？**
需要。关闭终端后程序会停止，下次用的时候重新启动就行。

如果每次打开 Claude Code 时想自动启动，可以配置 Claude Code 的 hooks（设置入口）让它自动启动。多个 Claude Code 窗口可以同时运行，它们会共用同一个后台程序，所以关闭其中一个不会影响其他窗口里的截图功能。

## 下载

GitHub 地址：https://github.com/kakaxi3019/wsl-clipboard-screenshot

环境要求：Windows 10/11 系统 + 开启了 WSL
