# Download / 下载

**WoW Backup Tool (wow-backup-tool-go)** — 一键备份和还原《魔兽世界》插件、配置与字体。
A desktop app that backs up and restores World of Warcraft addons, settings and fonts.

## 最新版本 / Latest release

请从 GitHub Releases 下载 / Download from GitHub Releases:

**https://github.com/panle80/wow-backup-tool-go/releases/latest**

| 平台 / Platform | 文件 / File |
|---|---|
| Windows 10/11 x64 | `WoWBackupTool-v*-Windows-x64.zip` |
| macOS Apple Silicon | `WoWBackupTool-v*-macOS-arm64.zip` |

解压后直接运行 `WoWBackupTool.exe`（Windows）或 `WoWBackupTool.app`（macOS），
无需安装。No installer is required.

## 功能 / What it does

- 自动检测 WoW 安装（注册表、Battle.net 配置、常见路径，及 `config.json` 自定义）
- 支持正式服、经典怀旧服、经典旧世、大灾变/时光服、无限服、PTR/Beta
- 备份 / 还原 `Interface`（插件）、`WTF`（设置）、`Fonts`（字体）
- 标准 ZIP + `manifest.json`；还原前校验备份，避免误删游戏文件
- 还原 `WTF` 时可保留本机显示配置（分辨率、窗口模式等）

详见 [使用说明.txt](使用说明.txt) 与 [README.md](README.md)。

## 系统要求 / Requirements

- Windows 10 21H2+ / Windows 11 x64（WebView2 运行时，系统通常已自带）
- macOS Apple Silicon（M 系列）
- 无需额外运行库，无需联网

## 代码签名 / Code signing

Windows release builds are code-signed.

Free code signing provided by [SignPath.io](https://signpath.io/), certificate by
[SignPath Foundation](https://signpath.org/).

签名政策 / Signing policy: [CODE_SIGNING_POLICY.md](CODE_SIGNING_POLICY.md)

## 隐私 / Privacy

本程序不收集、不上传任何数据，不包含遥测或联网请求。备份与还原完全在本机完成。
This program does not collect or transmit any data.

## 许可证 / License

[MIT](LICENSE) © 2026 潘公子大人 (panle80)
