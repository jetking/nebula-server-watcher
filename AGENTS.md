# Server Watcher - AI Agent 指南

这份文档旨在为协助开发和维护本项目（Server Watcher）的 AI Agent 提供核心上下文、架构说明及开发规范。

## 1. 项目概述

**Server Watcher** 是一个使用 Golang 编写的 VPS 延迟监控工具。它具备以下主要功能：
- **核心功能**: 通过持续并发的 PING 操作监控多个 VPS 节点的网络延迟。
- **数据持久化**: 使用 SQLite 存储每分钟的统计数据（最小值、最大值、平均值、中位数）。
- **Web 仪表盘**: 提供一个带有过滤和自动刷新功能的现代 Web 界面来展示监控数据。
- **告警通知**: 当节点连续 3 分钟的延迟中位数超过配置阈值时，向指定的 Telegram 频道或用户发送告警（每个节点有 30 分钟的冷却时间）。
- **安全防护**: Web 界面支持通过 `.env` 文件配置的密码保护。

## 2. 架构与技术栈

- **后端语言**: Go (Golang)
- **数据库**: SQLite (通过 `gorm.io/gorm` 和纯 Go 驱动 `github.com/glebarez/sqlite` 操作)
- **Web 框架**: Gin (`github.com/gin-gonic/gin`)
- **PING 库**: Pro-bing (`github.com/prometheus-community/pro-bing`)
- **配置管理**: TOML (`github.com/pelletier/go-toml/v2`) 和 godotenv (`github.com/joho/godotenv`)
- **前端**: 原生 HTML/JS/CSS (嵌入在 `web/` 目录中)

## 3. 核心文件结构

- `main.go`: 应用程序入口，负责初始化配置、数据库、监控器和 Web 服务器。
- `config.go`: 定义 `Config` 结构体，处理 `config.toml` 解析和 `.env` 环境变量加载。
- `monitor.go`: 核心监控逻辑，包含 PING 探测、数据计算、SQLite 数据保存以及 Telegram 告警触发逻辑。
- `db.go`: 数据库初始化和数据模型 (`LatencyRecord`) 定义。
- `web.go`: Web 服务器实现，包含静态文件服务、API 路由（带有鉴权中间件）和页面渲染。
- `web/`: 前端静态资源目录（`index.html`, `vps.html` 等）。

## 4. 开发规范与注意事项

- **并发与安全**: `monitor.go` 中的状态追踪（如告警状态记录）使用了 `sync.Mutex` 保护，在修改相关逻辑时必须确保线程安全。
- **错误处理**: 不要忽略错误，使用 `log.Printf` 记录后台运行中的非致命错误，Web API 需返回适当的 HTTP 状态码及 JSON 错误信息。
- **敏感信息分离**: **永远不要**将密码、Telegram Token (`TG_BOT_TOKEN`) 或 Chat ID (`TG_CHAT_ID`) 硬编码或存放在代码/配置文件模板中。这些必须通过 `.env` 或系统环境变量传入。
- **依赖管理**: 引入新库前需确保其轻量且与当前架构契合，新增依赖后需运行 `go mod tidy`。
- **PING 权限**: 在 Linux 系统下，可能需要通过 `setcap` 为编译后的二进制文件赋予发送 ICMP 包的权限，或使用非特权模式 (`pinger.SetPrivileged(false)`)。

## 5. 环境变量说明

开发或部署时，需在项目根目录创建 `.env` 文件或直接设置系统环境变量：

```env
WATCHER_PASSWORD=your_dashboard_password
TG_BOT_TOKEN=your_telegram_bot_token
TG_CHAT_ID=your_telegram_chat_id
```

## 6. 构建与运行

推荐使用 `build.sh` 脚本进行编译构建，构建产物将放置在 `_bin/` 目录下。

```bash
# 安装依赖
go mod tidy
# 执行构建
./build.sh
# 运行
./_bin/nebula-server-watcher-your-os-arch
```
