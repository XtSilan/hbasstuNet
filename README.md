# 理工校园网登录器

**hbasstuNet** 是面向湖北文理学院理工学院校园网的 Windows 桌面认证客户端。它将 Portal 认证、网络接口识别和连接状态集中到一个轻量的 Wails 应用中。

![Windows](https://img.shields.io/badge/Windows-10%20%7C%2011-0078D4?style=flat-square&logo=windows) ![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go) ![Vue](https://img.shields.io/badge/Vue-3-42B883?style=flat-square&logo=vuedotjs) ![License](https://img.shields.io/badge/License-MIT-27C79A?style=flat-square)

> [!WARNING]
> 本项目仅用于学习、研究和个人合法网络接入。请遵守学校网络管理规定，不得用于绕过认证、共享账号、干扰网络或其他未授权行为。使用前请阅读[免责声明](./DISCLAIMER.md)。

## 🚀 项目简介

hbasstuNet 自动识别校园无线网络和本机网络接口，使用当前 IPv4 与 MAC 地址完成账号检查、登录、状态展示和登出，减少每次打开浏览器认证的重复操作。

桌面界面由 Wails 与 Vue 3 构建，网络识别、凭据管理和 Portal 请求由 Go 负责。当前版本专注 Windows 10/11，macOS 与 Linux 尚未适配。

## ✨ 当前能力

- 识别 `Student-XYW`、`Teacher-XYW` 网络SSID
- 支持学生与教师账号模式切换
- 支持联通、移动、电信运营商选项
- 完成 CSRF Token、Cookie、账号检查、登录和登出流程
- 将 Portal 请求绑定到校园网接口的 IPv4 地址
- 展示 SSID、IPv4、MAC、接口与 Wi-Fi 信号强度
- 可在设置中启用自动登录并加入当前 Windows 用户的开机启动项
- 使用 Windows DPAPI 加密本地保存的密码
- 非校园网络下不发起 Portal 认证请求

> [!NOTE]
> 关闭窗口可选择最小化到系统托盘，后台会继续检查认证状态并在需要时重新登录。

## 🧱 技术栈与架构

| 层级 | 技术 | 用途 |
| --- | --- | --- |
| 桌面容器 | Wails v2 | Windows 原生窗口与 Go/前端绑定 |
| 前端 | Vue 3 + TypeScript + Vite | 登录与网络状态界面 |
| 图标 | Lucide Vue Next | 一致的线性界面图标 |
| 核心 | Go | Portal 认证、网络检测、状态管理 |
| Windows 集成 | Registry + DPAPI | 开机启动与凭据保护 |

```text
hbasstuNet.exe
├── Wails UI
│   └── Vue 3 + TypeScript
└── Go Core
    ├── Windows Wi-Fi 检测
    ├── Portal HTTP 客户端
    ├── 设置与 DPAPI 凭据保护
    └── 开机启动管理
```

## 📦 环境要求

- Windows 10 或 Windows 11（64 位）
- [Go 1.25 或更高版本](https://go.dev/dl/)
- [Node.js 20 或更高版本](https://nodejs.org/)
- [Wails v2 CLI](https://wails.io/docs/gettingstarted/installation)
- Microsoft Edge WebView2 Runtime
- 构建安装包时需要 [NSIS](https://nsis.sourceforge.io/Download)

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@latest
wails doctor
```

如果 PowerShell 找不到 `wails`，请把 `%USERPROFILE%\go\bin` 加入 `PATH`，或使用：

```powershell
& "$(go env GOPATH)\bin\wails.exe" doctor
```

## 🛠️ 本地开发

```powershell
git clone <仓库地址>
cd hbasstuNet
npm ci --prefix frontend
wails dev
```

## 🔨 构建

构建 Windows 可执行文件：

```powershell
npm ci --prefix frontend
wails build -clean
```

结果位于 `build/bin/hbasstuNet.exe`。Wails 配置已包含 Windows 打包所需的清单和 NSIS 模板。

生成 Windows NSIS 安装包时，请先安装 NSIS 并确保 `makensis` 可从 `PATH` 访问，然后执行：

```powershell
wails build -clean -nsis
```

安装包同样输出到 `build/bin/`。如果出现 `makensis not found`，说明可执行文件已构建，但安装包未生成。

只验证代码时：

```powershell
go test ./...
go vet ./...
npm run build --prefix frontend
```

> [!TIP]
> `go test` 和前端构建可以在不连接校园网的环境中运行；实际 Portal 登录只能在有权限的校园网络中验证。

## 🚀 发布与自动构建

普通提交和 Pull Request 只运行 Go 测试、静态检查与前端构建，不会上传 Actions artifact。推送以 `v` 开头的语义化版本标签时，GitHub Actions 会在 Windows runner 上完成检查、Wails 编译、SHA-256 校验，并将 exe 与校验文件直接发布到 GitHub Release：

```powershell
git tag v1.0.0
git push origin v1.0.0
```

标签会注入程序版本信息，软件“关于”页面会显示对应版本；“检查更新”会读取仓库最新 Release 的标签和发布说明。Release 正文取自该 tag 指向提交的提交说明，不会自动追加 GitHub 的 Full Changelog。Release 工作流不使用 `actions/upload-artifact`，构建目录只存在于临时 runner，发布完成后还会显式清理 `build/bin` 和 `release`。GitHub 托管 runner 在任务结束后也会销毁，因此不需要手动删除 runner 文件；需要注意的是，GitHub Release 资产会长期保留并计入仓库的 Release 存储。

程序为单文件便携版。若移动了 exe，先从新位置手动启动一次；程序会检测并同步 Windows 启动项到当前路径，之后开机启动将继续有效。

## 🖥️ 使用方式

1. 启动 hbasstuNet，应用会扫描附近的 `Student-XYW` 或 `Tercher-XYW` 网络。
2. 选择已发现的学生或教师网络。
3. 输入校园网账号、密码与运营商。
4. 根据需要启用保存密码；如需开机后自动认证，请在设置中启用“自动登录”。
5. 点击“连接校园网”，认证成功后查看当前网络信息。
6. 需要注销 Portal 会话时点击“断开连接”。

设置文件保存在：

```text
%AppData%\hbasstuNet\settings.json
```

启用“保存密码”后，文件中只保存由 Windows DPAPI 加密的密文。

> [!CAUTION]
> 校园 Portal 当前使用内网 HTTP 服务，认证请求不具备 HTTPS 端到端加密保护。请仅在可信校园网络环境使用，不要提交 `settings.json`、账号、密码、Cookie 或 Token。

## 📁 目录结构

```text
.
├── app.go                    # Wails 应用服务与状态管理
├── main.go                   # 桌面程序入口
├── internal/config/          # 设置与 DPAPI 凭据保护
├── internal/network/         # Windows Wi-Fi 和接口识别
├── internal/portal/          # Portal API 客户端
├── internal/startup/         # Windows 开机启动项
├── frontend/src/             # Vue 3 界面
├── build/windows/            # 图标、清单和 NSIS 配置
└── .github/workflows/        # 自动测试与 Windows 构建
```

## 🔐 安全说明

- 不会把账号密码编译进程序。
- 不会在日志中输出密码、CSRF Token 或 Cookie。
- 保存密码使用 Windows 当前用户范围的 DPAPI。
- Portal 请求只在识别到指定校园 SSID 后发起。
- 开机启动项写入 `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`

发现安全问题时，请不要公开真实账号、会话标识或校园网内部数据。

## 🗺️ 路线图

- [x] 系统托盘与窗口隐藏行为
- [x] Portal 状态检查与自动重连
- [ ] Windows 安装程序中的开机启动选项
- [ ] 多账号管理
- [ ] 多接入链路负载均衡
- [ ] CLI 版本
- [ ] macOS 与 Linux 网络适配

## 🤝 参与开发

提交改动前请运行：

```powershell
go test ./...
go vet ./...
npm ci --prefix frontend
npm run build --prefix frontend
```

请勿在 Issue、提交记录、测试夹具或截图中包含真实账号、密码、Cookie、Token、完整会话标识或其他个人信息。

## 📄 许可证

源代码依据 [MIT License](./LICENSE) 开放。项目名称、校园网络名称及相关标识不代表学校官方背书；使用本软件还须遵守[免责声明](./DISCLAIMER.md)和所在网络的管理规定。
