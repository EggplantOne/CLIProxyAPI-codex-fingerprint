# Codex 上游指纹对齐（本 Fork 的改动说明）

本仓库是 [router-for-me/CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) **v7.3.12**
的一个 fork，唯一目的是把发往 OpenAI 的 Codex 上游请求指纹对齐到官方 codex CLI，
降低被风控识别为"第三方代理"的概率（防降智/防封）。

## 对齐的 codex 版本

- **`0.157.0`**（`codex_exec/0.157.0`）
- 与官方 `@openai/codex@0.157.0` 实测抓包一致：
  - UA：`codex_exec/0.157.0 (Ubuntu 20.4.0; x86_64) xterm-256color (codex_exec; 0.157.0)`
  - Originator：`codex_exec`
  - Beta 头：`x-codex-beta-features: remote_compaction_v2`

## 相对上游的改动

| 文件 | 改动 |
|---|---|
| `internal/runtime/executor/codex_executor_request.go` | `codexUserAgent`/`codexOriginator` → `codex_exec/0.157.0`；HTTP 路径 `x-codex-beta-features` 增加 config 回退 |
| `internal/registry/models/models.json` | `gpt-5.6-luna` 的按模型 UA 覆盖（4 处）→ `codex_exec/0.157.0` |
| `internal/api/handlers/management/api_tools.go` 等 3 处 | 移除上游硬编码的 Google OAuth client secret（`GOCSPX-…`，改回空值） |
| `internal/runtime/executor/codex_fingerprint_dump_test.go` | 指纹回归测试 |
| `.gitignore` | 忽略 `data/` 等本地运行时数据与密钥 |

其余均为上游原样，未改动。

## 配置（`codex-header-defaults`）

```yaml
codex-header-defaults:
  user-agent: 'codex_exec/0.157.0 (Ubuntu 20.4.0; x86_64) xterm-256color (codex_exec; 0.157.0)'
  beta-features: 'remote_compaction_v2'
```

## 构建

```bash
go build -trimpath -ldflags "-s -w -X main.Version=7.3.12 -X main.Commit=<commit> -X main.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o cli-proxy-api ./cmd/server
```

## 更新 codex 版本

官方 codex 升级后，按以下步骤同步（改完重新编译并部署）：

1. 抓官方新版的真实 UA：把 codex 指向本地抓包服务器，记录 `user-agent`；
2. 更新 `internal/runtime/executor/codex_executor_request.go` 的 `codexUserAgent` 常量；
3. 更新 `internal/registry/models/models.json` 中 `gpt-5.6-luna` 的 override；
4. 更新部署侧 `config.yaml` 的 `codex-header-defaults.user-agent`。

> 注意：UA 里带了机器信息（`Ubuntu 20.4.0; x86_64`），如果你的客户端跑在别的
> OS/架构上，请用你自己抓到的真实 UA，而不是照抄这里的字符串。
