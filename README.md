<div align="center">

# 🎭 Coding Plan Mask

**Unlock Your Coding Plan API Key for Any AI Coding Tool**

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-0.4.0-green.svg)](https://github.com/systemime/coding-plan-mask)

*Use your Coding Plan subscription with ANY OpenAI-compatible coding tool*

[English](#-english-documentation) | [中文文档](#-中文文档)

</div>

---

## 📖 English Documentation

### 😤 The Problem: Coding Plan Restrictions

Major AI providers (Zhipu GLM, Alibaba Cloud, MiniMax, DeepSeek, Moonshot, etc.) offer **Coding Plan** subscriptions at attractive prices, but with **severe usage restrictions**:

| What You Pay For | What You Actually Get |
|------------------|----------------------|
| ✅ Fixed monthly fee, unlimited coding | ❌ **Only works with specific IDE tools** |
| ✅ Access to powerful models | ❌ **Cannot use in your favorite tools** |
| ✅ Official API Key provided | ❌ **Cannot use for automation/backend** |

#### 🔒 Official Restrictions

| Allowed | Forbidden |
|---------|-----------|
| ✅ Claude Code, Cursor, Cline | ❌ Your own AI tools |
| ✅ VS Code extensions | ❌ Custom scripts |
| ✅ Interactive coding | ❌ Automated workflows |
| | ❌ Backend integration |
| | ❌ Dify, FastGPT platforms |

**Violation Consequence**: Subscription suspension or API Key ban

#### 📊 Provider Comparison

| Provider | Monthly Fee | Models | Can Use in Custom Tools? |
|----------|-------------|--------|-------------------------|
| Zhipu GLM | $3-15+ | GLM-4.7, GLM-5 | ❌ No |
| Alibaba Cloud | $5.80-29 | Qwen, GLM, MiniMax, Kimi | ❌ No |
| MiniMax | Subscription | M2.1 (not M2.5!) | ❌ No |
| DeepSeek | Subscription | DeepSeek V3 | ❌ No |
| Moonshot | Subscription | Kimi | ❌ No |

### 💡 The Solution: Coding Plan Mask

**Coding Plan Mask** acts as a bridge between your Coding Plan API and any OpenAI-compatible tool. It **masks** your requests to appear as if they come from officially supported IDE tools.

```
┌────────────────────┐     ┌──────────────────────┐     ┌─────────────────────┐
│  Your Favorite AI  │────▶│   Coding Plan Mask   │────▶│   LLM Provider      │
│  Tool (Any!)       │◀────│   (Tool Masking)     │◀────│   (Thinks it's OK)  │
└────────────────────┘     └──────────────────────┘     └─────────────────────┘
```

### ✨ Key Features

| Feature | Description |
|---------|-------------|
| 🎭 **Tool Masking** | Mask as OpenCode, OpenClaw, or custom tool |
| 🔀 **Request Relay** | Relay requests to Coding Plan API endpoints |
| 🔌 **Universal Compatibility** | Works with ANY OpenAI-compatible client |
| 🌐 **Multi-Provider** | Support for 6+ major LLM providers |
| 📊 **Usage Analytics** | Track token consumption in real-time with SQLite storage |
| 🔒 **Local Auth** | Protect your proxy with custom API key |
| ⚡ **High Performance** | Built in Go for maximum efficiency |
| 🔧 **Flexible Configuration** | Support TOML config file, environment variables, and custom API URLs |
| 📈 **Rate Limiting** | Built-in rate limiting to prevent abuse |

### 🚀 Quick Start

#### 1. Install

```bash
# Download from releases or build from source
git clone https://github.com/systemime/coding-plan-mask.git
cd coding-plan-mask
make build
sudo make install
```

#### 2. Configure

Edit `/opt/project/coding-plan-mask/config/config.toml`:

```toml
[server]
listen_host = "127.0.0.1"
listen_port = 8787
timeout = 120                       # Request timeout (seconds)
rate_limit_requests = 100           # Rate limit per 5 minutes

[auth]
provider = "zhipu"                  # Your Coding Plan provider
api_key = "your-coding-plan-api-key"  # Your Coding Plan API Key
local_api_key = "sk-local-secret"   # Key for your tools to use

[endpoint]
use_coding_endpoint = true
disguise_tool = "claudecode"        # Mask as Claude Code (recommended)
# custom_user_agent = ""            # Custom User-Agent (when disguise_tool = "custom")

[api]
# Optional: Custom API URLs
base_url = ""                       # Custom base URL
coding_url = ""                     # Custom coding endpoint URL
```

#### 3. Start

```bash
mask-ctl start
```

#### 4. Use with Any Tool

Configure your AI coding tool to use:

```json
{
    "base_url": "http://127.0.0.1:8787/v1",
    "api_key": "sk-local-secret",
    "model": "glm-4-flash"
}
```

### 🤖 Supported Providers

| Provider | Identifier | Models |
|----------|------------|--------|
| **Zhipu GLM** | `zhipu` | glm-4-flash, glm-4-plus, glm-4-air, glm-4-long |
| **Zhipu GLM v2** | `zhipu_v2` | glm-4-flash, glm-4-plus, glm-4-air, glm-4-long, glm-4.7, glm-5 |
| **Alibaba Cloud** | `aliyun` | qwen-turbo, qwen-plus, qwen-max, qwen2.5-coder-32b-instruct |
| **MiniMax** | `minimax` | abab6.5s-chat, abab6.5g-chat, abab6.5-chat |
| **DeepSeek** | `deepseek` | deepseek-chat, deepseek-coder |
| **Moonshot** | `moonshot` | moonshot-v1-8k, moonshot-v1-32k, moonshot-v1-128k |
| **Custom** | `custom` | Use `[api]` section to configure custom URLs |

### 🎭 Tool Masking Options

```toml
[endpoint]
# Mask as officially supported tools
disguise_tool = "claudecode"  # Claude Code (recommended)
# disguise_tool = "cursor"     # Cursor IDE
# disguise_tool = "cline"      # Cline (VS Code extension)
# disguise_tool = "opencode"   # OpenCode (archived)
# disguise_tool = "openclaw"   # OpenClaw
# disguise_tool = "copilot"    # GitHub Copilot
# disguise_tool = "custom"     # Use custom User-Agent
# custom_user_agent = "YourCustomTool/1.0"
```

| Tool | Identifier | User-Agent | Description |
|------|------------|------------|-------------|
| **Claude Code** | `claudecode` | `claude-code/2.0.64` | Anthropic 官方终端编程助手 (推荐) |
| **Cursor** | `cursor` | `cursor/0.45.0` | AI 代码编辑器 |
| **Cline** | `cline` | `cline/3.0.0` | VS Code AI 编程助手 |
| **OpenCode** | `opencode` | `opencode/0.3.0 (linux)` | 开源编程助手 (已归档) |
| **OpenClaw** | `openclaw` | `OpenClaw-Gateway/1.0` | AI 编程工具 |
| **GitHub Copilot** | `copilot` | `GithubCopilot/1.0` | GitHub AI 编程助手 |
| **Custom** | `custom` | (自定义) | 使用 `custom_user_agent` 配置 |

### 📡 API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/` | GET | Service information |
| `/v1/chat/completions` | POST | Chat completions (streaming supported) |
| `/v1/embeddings` | POST | Text embeddings |
| `/v1/models` | GET | List available models |
| `/health` | GET | Health check |
| `/ready` | GET | Readiness check |
| `/stats` | GET | Usage statistics (JSON) |

### 📊 Statistics & Management

```bash
# View service info and connection
mask-ctl info

# View service status
mask-ctl status

# View real-time logs
mask-ctl logs

# Enable/disable auto-start
mask-ctl enable
mask-ctl disable

# View usage statistics via API
curl http://127.0.0.1:8787/stats
```

### 🔧 Environment Variables

You can also configure via environment variables:

| Variable | Description |
|----------|-------------|
| `PROVIDER` | Provider identifier |
| `API_KEY` | Coding Plan API Key |
| `LOCAL_API_KEY` | Local API Key for authentication |
| `HOST` | Listen host |
| `PORT` | Listen port |
| `DEBUG` | Enable debug mode (true/false) |
| `API_BASE_URL` | Custom API base URL |
| `API_CODING_URL` | Custom coding endpoint URL |

### ⚠️ Risk Warning

> **IMPORTANT: Please read carefully before using this project**

This project is provided for **educational and research purposes only**.

| Risk | Description |
|------|-------------|
| 🔴 **Terms of Service** | May violate your provider's Terms of Service |
| 🔴 **Account Risk** | Improper use may result in API key revocation or account suspension |
| 🟡 **No Warranty** | Software provided "as-is" without any warranty |
| 🟡 **Security** | Exposing proxy to public networks may lead to unauthorized access |
| 🟢 **Self-Responsibility** | Users assume full responsibility for compliance |

**By using this software, you agree to:**
- Use it at your own risk
- Comply with all applicable laws and provider terms
- Accept full responsibility for any consequences

**Recommendations:**
1. Only use with APIs you have legitimate access to
2. Always set `local_api_key` to prevent unauthorized access
3. Bind to `127.0.0.1` unless you fully understand security implications
4. Review your provider's Terms of Service

---

## 📖 中文文档

### 😤 问题背景：Coding Plan 的使用限制

各大 AI 服务商（智谱 GLM、阿里云百炼、MiniMax、DeepSeek、Moonshot 等）推出的 **Coding Plan（编码套餐）** 虽然价格诱人，但有**严格的使用限制**：

| 你以为买到的 | 实际上只能 |
|-------------|-----------|
| ✅ 固定月费，无限编码 | ❌ **只能在指定的 IDE 工具中使用** |
| ✅ 访问强大的模型 | ❌ **不能在你喜欢的工具里用** |
| ✅ 获得官方 API Key | ❌ **不能用于自动化/后端** |

#### 🔒 官方限制条款

以阿里云百炼为例，Coding Plan 明确规定：

| 允许的使用方式 | 禁止的使用方式 |
|---------------|---------------|
| ✅ Claude Code、Cursor、Cline | ❌ 你自己的 AI 工具 |
| ✅ VS Code 插件 | ❌ 自定义脚本 |
| ✅ 人工交互编码 | ❌ 自动化工作流 |
| | ❌ 后端服务调用 |
| | ❌ Dify、FastGPT 等平台 |

**违规后果**：订阅暂停或 API Key 封禁

#### 📊 各厂商限制对比

| 服务商 | 月费 | 模型 | 可用于自定义工具？ |
|--------|------|------|-------------------|
| 智谱 GLM | ¥20-100+ | GLM-4.7, GLM-5 | ❌ 不可以 |
| 阿里云百炼 | ¥40-200 | 通义、GLM、MiniMax、Kimi | ❌ 不可以 |
| MiniMax | 订阅制 | M2.1（非 M2.5！） | ❌ 不可以 |
| DeepSeek | 订阅制 | DeepSeek V3 | ❌ 不可以 |
| Moonshot | 订阅制 | Kimi | ❌ 不可以 |

### 💡 解决方案：Coding Plan Mask

**Coding Plan Mask** 作为你的 Coding Plan API 和任意 OpenAI 兼容工具之间的桥梁。它将你的请求**伪装**成来自官方支持的 IDE 工具。

```
┌────────────────────┐     ┌──────────────────────┐     ┌─────────────────────┐
│   你喜欢的 AI 工具   │────▶│   Coding Plan Mask   │────▶│     LLM 供应商      │
│   （任意！）         │◀────│   （工具伪装）         │◀────│   （以为没问题）     │
└────────────────────┘     └──────────────────────┘     └─────────────────────┘
```

### ✨ 核心功能

| 功能 | 说明 |
|------|------|
| 🎭 **工具伪装** | 伪装为 OpenCode、OpenClaw 或自定义工具 |
| 🔀 **请求中转** | 将请求中转到 Coding Plan API 端点 |
| 🔌 **通用兼容** | 兼容任何支持 OpenAI API 的客户端 |
| 🌐 **多供应商** | 支持 6+ 主流大模型供应商 |
| 📊 **用量统计** | 实时追踪 Token 消耗，SQLite 持久化存储 |
| 🔒 **本地认证** | 用自定义密钥保护你的代理 |
| ⚡ **高性能** | Go 语言构建，极致效率 |
| 🔧 **灵活配置** | 支持 TOML 配置文件、环境变量和自定义 API URL |
| 📈 **速率限制** | 内置速率限制防止滥用 |

### 🚀 快速开始

#### 1. 安装

```bash
# 从 Release 下载或从源码编译
git clone https://github.com/systemime/coding-plan-mask.git
cd coding-plan-mask
make build
sudo make install
```

#### 2. 配置

编辑 `/opt/project/coding-plan-mask/config/config.toml`：

```toml
[server]
listen_host = "127.0.0.1"
listen_port = 8787
timeout = 120                       # 请求超时(秒)
rate_limit_requests = 100           # 每5分钟请求限制

[auth]
provider = "zhipu"                  # 你的 Coding Plan 供应商
api_key = "your-coding-plan-api-key"  # 你的 Coding Plan API Key
local_api_key = "sk-local-secret"   # 你的工具使用的密钥

[endpoint]
use_coding_endpoint = true
disguise_tool = "claudecode"        # 伪装为 Claude Code (推荐)
# custom_user_agent = ""            # 自定义 User-Agent（当 disguise_tool = "custom" 时）

[api]
# 可选：自定义 API URL
base_url = ""                       # 自定义基础 URL
coding_url = ""                     # 自定义 Coding 端点 URL
```

#### 3. 启动

```bash
mask-ctl start
```

#### 4. 配置你的 AI 工具

```json
{
    "base_url": "http://127.0.0.1:8787/v1",
    "api_key": "sk-local-secret",
    "model": "glm-4-flash"
}
```

### 🤖 支持的供应商

| 供应商 | 标识符 | 支持模型 |
|--------|--------|----------|
| **智谱 GLM** | `zhipu` | glm-4-flash, glm-4-plus, glm-4-air, glm-4-long |
| **智谱 GLM v2** | `zhipu_v2` | glm-4-flash, glm-4-plus, glm-4-air, glm-4-long, glm-4.7, glm-5 |
| **阿里云百炼** | `aliyun` | qwen-turbo, qwen-plus, qwen-max, qwen2.5-coder-32b-instruct |
| **MiniMax** | `minimax` | abab6.5s-chat, abab6.5g-chat, abab6.5-chat |
| **DeepSeek** | `deepseek` | deepseek-chat, deepseek-coder |
| **Moonshot (Kimi)** | `moonshot` | moonshot-v1-8k, moonshot-v1-32k, moonshot-v1-128k |
| **自定义** | `custom` | 使用 `[api]` 配置段自定义 URL |

### 🎭 工具伪装选项

```toml
[endpoint]
# 伪装为官方支持的工具
disguise_tool = "claudecode"  # Claude Code（推荐）
# disguise_tool = "cursor"     # Cursor IDE
# disguise_tool = "cline"      # Cline (VS Code 插件)
# disguise_tool = "opencode"   # OpenCode (已归档)
# disguise_tool = "openclaw"   # OpenClaw
# disguise_tool = "copilot"    # GitHub Copilot
# disguise_tool = "custom"     # 使用自定义 User-Agent
# custom_user_agent = "YourCustomTool/1.0"
```

| 工具 | 标识符 | User-Agent | 说明 |
|------|--------|------------|------|
| **Claude Code** | `claudecode` | `claude-code/2.0.64` | Anthropic 官方终端编程助手 (推荐) |
| **Cursor** | `cursor` | `cursor/0.45.0` | AI 代码编辑器 |
| **Cline** | `cline` | `cline/3.0.0` | VS Code AI 编程助手 |
| **OpenCode** | `opencode` | `opencode/0.3.0 (linux)` | 开源编程助手 (已归档) |
| **OpenClaw** | `openclaw` | `OpenClaw-Gateway/1.0` | AI 编程工具 |
| **GitHub Copilot** | `copilot` | `GithubCopilot/1.0` | GitHub AI 编程助手 |
| **自定义** | `custom` | (自定义) | 使用 `custom_user_agent` 配置 |

### 📡 API 端点

| 端点 | 方法 | 说明 |
|------|------|------|
| `/` | GET | 服务信息 |
| `/v1/chat/completions` | POST | 聊天补全（支持流式） |
| `/v1/embeddings` | POST | 文本向量嵌入 |
| `/v1/models` | GET | 可用模型列表 |
| `/health` | GET | 健康检查 |
| `/ready` | GET | 就绪检查 |
| `/stats` | GET | 使用统计（JSON） |

### 📊 统计与管理

```bash
# 查看服务信息和连接配置
mask-ctl info

# 查看服务状态
mask-ctl status

# 查看实时日志
mask-ctl logs

# 开启/关闭开机自启
mask-ctl enable
mask-ctl disable

# 通过 API 查看使用统计
curl http://127.0.0.1:8787/stats
```

### 🔧 环境变量配置

你也可以通过环境变量进行配置：

| 变量 | 说明 |
|------|------|
| `PROVIDER` | 供应商标识符 |
| `API_KEY` | Coding Plan API Key |
| `LOCAL_API_KEY` | 本地认证 API Key |
| `HOST` | 监听地址 |
| `PORT` | 监听端口 |
| `DEBUG` | 启用调试模式 (true/false) |
| `API_BASE_URL` | 自定义 API 基础 URL |
| `API_CODING_URL` | 自定义 Coding 端点 URL |

### 📦 项目结构

```
coding-plan-mask/
├── cmd/
│   └── coding-plan-mask/       # 主程序入口
│       └── main.go
├── internal/
│   ├── cmd/                     # 命令行工具
│   │   └── stats/               # 统计命令
│   ├── config/                  # 配置管理
│   │   └── config.go
│   ├── proxy/                   # 代理核心逻辑
│   │   └── proxy.go
│   ├── server/                  # HTTP 服务器
│   │   └── server.go
│   ├── storage/                 # 数据存储（SQLite）
│   │   └── storage.go
│   └── ratelimit/               # 速率限制
│       └── ratelimit.go
├── deploy/                      # 部署相关
│   ├── config.example.toml      # 配置示例
│   ├── config.example.json      # JSON 配置示例
│   ├── mask-ctl.sh              # 控制脚本
│   └── coding-plan-mask.service # systemd 服务文件
├── Makefile                     # 构建脚本
├── go.mod                       # Go 模块定义
├── go.sum                       # 依赖校验
└── README.md                    # 项目文档
```

### ⚠️ 风险预警

> **重要提示：使用前请仔细阅读**

本项目仅供**学习和研究目的**。

| 风险 | 说明 |
|------|------|
| 🔴 **服务条款** | 可能违反你供应商的服务条款 |
| 🔴 **账户风险** | 不当使用可能导致 API 密钥被吊销或账户被暂停 |
| 🟡 **无担保** | 软件按"现状"提供，不提供任何担保 |
| 🟡 **安全风险** | 将代理暴露到公共网络可能导致未授权访问 |
| 🟢 **自负责任** | 用户需对遵守适用法律法规承担全部责任 |

**使用本软件即表示您同意：**
- 自行承担使用风险
- 遵守所有适用法律和供应商条款
- 对任何后果承担全部责任

**建议：**
1. 仅在您拥有合法访问权限的 API 上使用
2. 始终设置 `local_api_key` 以防止未授权访问
3. 除非您完全了解安全影响，否则绑定到 `127.0.0.1`
4. 审查您供应商的服务条款

---

## 📚 参考资料 / References

- [智谱 AI 开放文档 - Coding Plan FAQ](https://docs.bigmodel.cn/cn/coding-plan/faq)
- [阿里云百炼 - Coding Plan 接入工具](https://help.aliyun.com/zh/model-studio/other-tools-coding-plan)
- [Coding Plan 能当 API 用吗？各家限制一览](https://help.apiyi.com/coding-plan-api-restrictions-openai-codex-exception.html)

---

## 🛠️ Development

### Build from Source

```bash
# Clone the repository
git clone https://github.com/systemime/coding-plan-mask.git
cd coding-plan-mask

# Install dependencies
make deps

# Build
make build

# Build for specific platform
make build-linux    # Linux amd64
make build-arm64    # Linux arm64

# Run tests
make test

# Run locally
make run
```

### Tech Stack

- **Language**: Go 1.21+
- **HTTP Server**: net/http
- **Configuration**: TOML (github.com/BurntSushi/toml)
- **Logging**: Zap (go.uber.org/zap)
- **Storage**: SQLite3 (github.com/mattn/go-sqlite3)
- **Rate Limiting**: golang.org/x/time/rate

---

<div align="center">

**⭐ If this project helps you, please give it a star! ⭐**

Made with ❤️ by the community

</div>
