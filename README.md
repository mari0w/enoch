# enoch

Go 进程负责轮询 Telegram bot，并把消息转发给本地 Codex CLI。Codex 会读取根目录 `skills/` 来完成操作。

## 结构
- `cmd/enoch`：Go 入口
- `internal/telegram`：Telegram 轮询
- `internal/codex`：Codex CLI 调用
- `internal/logging`：日志模块（控制台 + 文件）
- `skills/`：技能目录（由 Codex 读取）

## 快速开始
1. 通过 @BotFather 创建 Telegram bot，拿到 token。
2. 复制 `.env.example` 为 `.env` 并填写配置。
3. 运行：

```bash
go run ./cmd/enoch
```

如果你不使用 `run.sh`，也不需要设置 `CODEX_HOME`。我们通过仓库内的 `.codex/skills` 来确保只加载项目技能。

## Codex 认证
Codex CLI 需要认证才能调用模型。你可以选择以下方式之一：
- 交互式登录：`codex login`（无界面环境可用 `codex login --device-auth`）
- 非交互式（推荐用于机器人）：在 `.env` 中设置 `CODEX_API_KEY`
  
注意：Codex 默认把认证缓存写在 `~/.codex/auth.json` 或系统凭据库中；如果你改了 `CODEX_HOME` 并且使用的是文件缓存，会导致找不到登录信息，从而出现 401。

## 配置说明
- `TELEGRAM_BOT_TOKEN`：Bot token（必填）
- `TELEGRAM_ALLOWED_CHAT_ID`：限制只允许该 chat id 使用（建议填写）
- `TELEGRAM_POLL_INTERVAL`：轮询间隔秒数
- `TELEGRAM_TYPING_INTERVAL`：发送“正在输入”的间隔秒数（0 关闭）
- `TELEGRAM_CONTEXT_SIZE`：每个 chat 保留最近 N 条上下文（0 关闭）

- `CODEX_COMMAND`：Codex CLI 命令，默认 `codex`
- `CODEX_ARGS`：额外参数，支持 `{prompt}` 占位符（默认 `exec {prompt}`，非交互）
- `CODEX_PROMPT_MODE`：`stdin` 或 `arg`（默认 `arg`）
- `CODEX_USE_TTY`：是否使用 `script(1)` 提供伪终端（默认 `false`，仅在交互式 CLI 需要时开启）
- `CODEX_DISABLE_CPR`：禁用终端光标位置读取（解决部分 CLI 的 `cursor position` 错误）
- `CODEX_TIMEOUT`：超时时间（秒）
- `CODEX_WORKDIR`：Codex 工作目录（默认 `.`，用于读取 `skills/`）
- `CODEX_PROGRESS_INTERVAL`：Codex 执行超过该时间后每隔该秒数输出“仍在运行”日志（0 表示关闭）；同时用于 Telegram 的“仍在处理中”提示（不会高于 30 秒一次）
- `CODEX_HOME`：Codex 的 Home 目录（默认 `~/.codex`）。只有在你确实要隔离配置/凭据时才设置；否则建议保持默认值以复用已有登录缓存。

- `LOG_LEVEL`：`debug|info|warn|error`
- `LOG_FILE`：日志文件路径（为空表示不写文件）
- `LOG_CONSOLE`：是否输出到控制台
- `LOG_COLOR`：控制台彩色输出
- `LOG_TIME_FORMAT`：时间格式（默认 `2006-01-02 15:04:05`）

## Telegram 指令
- `/status`：查看运行状态、队列长度与上下文统计
- `/stop`：暂停处理新任务（接收继续，排队不执行）
- `/resume`：恢复处理
- `/reset`：清空该 chat 的上下文

## 依赖说明
- 如果 `CODEX_USE_TTY=true`，系统需要可用的 `script` 命令。
  - macOS 默认自带 `script`
  - Linux 通常来自 `util-linux`

## 技能目录
当前示例技能：`skills/system/SKILL.md`。

注意：`SKILL.md` 需要 YAML frontmatter（`---` 包裹的 `name`/`description`），否则 Codex 会拒绝加载。

Codex 会按优先级加载团队配置（含 `skills/`），其中包括仓库内的 `.codex/skills`，其优先级高于全局配置。

本项目使用 `.codex/skills -> skills/` 的符号链接，保证：
- Codex 只读取当前项目技能
- 你的登录缓存不受 `CODEX_HOME` 变更影响

如果你的 Codex skill 规范与此不同，请给我一个样例，我会按规范调整。
