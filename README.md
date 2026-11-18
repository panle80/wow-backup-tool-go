# WoW Backup Tool / WoW 插件备份恢复工具

一键备份和还原《魔兽世界》插件、配置与字体的桌面工具。基于 Go + Wails v2，
单文件、约 11 MB、启动迅速。支持 Windows x64 与 macOS (Apple Silicon)。

**下载：[GitHub Releases](https://github.com/panle80/wow-backup-tool-go/releases)**

## 功能

- 自动检测 WoW 安装（注册表、Battle.net 配置、常见路径，并支持 `config.json` 自定义）
- 支持正式服、经典怀旧服、经典旧世、大灾变/时光服、无限服（`_forever_`）、PTR/Beta
- 备份 / 还原 `Interface`、`WTF`、`Fonts`
- 标准 ZIP 格式并内嵌 `manifest.json`；还原前先校验备份，避免误删
- 还原 `WTF` 时可保留本机显示配置（分辨率、窗口模式等）
- 全局游戏选择器 + 双 Tab 深色界面，进度实时反馈

## 使用说明

详见 [使用说明.txt](使用说明.txt)。

快速开始：下载 exe → 运行 → 在顶部选择游戏版本 → 勾选要备份的目录 →
选择保存位置 → 创建备份。默认保存到 `Documents\WoWBackups`。

## 系统要求

- Windows 10 21H2+ / Windows 11（需要 WebView2 运行时，系统通常已自带），64 位
- macOS Apple Silicon（M 系列）

## 从源码构建

前置要求：Go 1.21+、Node.js 18+、[Wails CLI](https://wails.io/)

```bash
# 安装 Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 开发模式（热更新）
wails dev

# Windows 构建 → build/bin/WoWBackupTool.exe
wails build -platform windows/amd64 -clean

# macOS 构建（Apple Silicon）→ build/bin/WoWBackupTool.app
wails build -platform darwin/arm64 -clean
```

## 代码签名

Windows Release 构建由 [SignPath.io](https://signpath.io/) 提供的免费开源
代码签名服务签名，证书由 [SignPath Foundation](https://signpath.org/) 签发。

Free code signing provided by [SignPath.io](https://signpath.io/), certificate
by [SignPath Foundation](https://signpath.org/). See
[Code Signing Policy](CODE_SIGNING_POLICY.md).

## 隐私

本程序不收集、不上传任何数据，不包含遥测或分析，也没有任何联网请求；
备份与还原完全在本机完成。

为记住上次保存位置，Windows 下会在注册表
`HKCU\SOFTWARE\pan.wowbackup` 写入 `LastOutputDir`，macOS 下使用
`UserDefaults`。程序为用户自行创建的 `config.json` 仅用于自定义 WoW 路径。

卸载：删除程序文件即可；如要清除保存目录记录，删除上述注册表项。

## License

[MIT](LICENSE) © 2026 潘公子大人 (panle80)

---

## English summary

**WoW Backup Tool** is a desktop app for backing up and restoring World of
Warcraft addons (`Interface`), game settings (`WTF`) and fonts. Built with
Go + Wails v2 and shipped as a single executable for Windows x64 and
macOS arm64.

- Download: [GitHub Releases](https://github.com/panle80/wow-backup-tool-go/releases)
- Free code signing provided by [SignPath.io](https://signpath.io/),
  certificate by [SignPath Foundation](https://signpath.org/); see
  [Code Signing Policy](CODE_SIGNING_POLICY.md).
- Privacy: no telemetry and no network transfers. The last output directory is
  stored locally (Windows registry / macOS UserDefaults).
- License: [MIT](LICENSE).
