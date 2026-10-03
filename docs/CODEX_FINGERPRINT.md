# Codex 上游指纹对齐（动态版）

本仓库是 [router-for-me/CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) **v8.0.12**
的 fork，把发往 OpenAI 的 Codex 上游请求指纹对齐到官方 codex CLI，并**动态跟随 codex 版本更新**。

## 动态指纹机制

- 后台每 **3 小时** 从 npm registry 拉取官方 `@openai/codex` 最新版本；
- UA 在运行时动态拼装：`codex_exec/<最新版> (<OS> <版本>; <架构>) <终端> (codex_exec; <最新版>)`；
- OS / 架构 / 终端在运行时自动探测（读取 `/etc/os-release`、`runtime.GOARCH`、`$TERM`）；
- 拉取失败时回退到内置版本 `0.160.0`，不会中断服务。

因此 **codex 升级后无需改代码、无需重编译**，CPA 会自动跟随。

## 相对上游的改动

| 文件 | 改动 |
|---|---|
| `internal/misc/codex_version.go`（新增） | 动态版本拉取 + UA 拼装 |
| `internal/runtime/executor/codex_executor_request.go` | UA/Originator 改用动态函数；originator → `codex_exec` |
| `internal/runtime/executor/codex_websockets_request.go` | 同上 |
| `internal/registry/models/models.json` | 移除 `gpt-5.6-luna` 的过期 `codex-tui` UA 覆盖 |
| `cmd/server/main.go` | 启动 codex 版本刷新器 |
| `internal/api/handlers/management/api_tools.go` 等 3 处 | 移除硬编码 Google OAuth client secret（`GOCSPX-…`） |
| `.gitignore` | 忽略 `data/` 等本地运行时数据与密钥 |

## 构建

```bash
go build -buildvcs=false -trimpath \
  -ldflags "-s -w -X main.Version=8.0.12 -X main.Commit=<commit> -X main.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o cli-proxy-api ./cmd/server
```

> 建议在内存充足的机器上编译（目标二进制约 66MB，链接阶段内存占用较高，勿在 1GB 以下内存的 VPS 上直接 `go build`）。
