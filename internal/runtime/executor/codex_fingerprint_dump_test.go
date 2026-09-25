package executor

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
)

// TestCodexFingerprintDump 打印 CPA 为一次 Codex 请求计算出的上游头，
// 用于和官方 codex 客户端的真实请求头对比。
func TestCodexFingerprintDump(t *testing.T) {
	cfg := &config.Config{}
	cfg.CodexHeaderDefaults.UserAgent = "codex_exec/0.157.0 (Ubuntu 20.4.0; x86_64) xterm-256color (codex_exec; 0.157.0)"
	cfg.CodexHeaderDefaults.BetaFeatures = "remote_compaction_v2"

	auth := &cliproxyauth.Auth{Provider: "codex"}

	// 模拟官方 codex 客户端下发的头（与抓包一致）
	down := http.Header{}
	down.Set("User-Agent", "codex_exec/0.157.0 (Ubuntu 20.4.0; x86_64) xterm-256color (codex_exec; 0.157.0)")
	down.Set("X-Codex-Beta-Features", "remote_compaction_v2")
	down.Set("X-Codex-Window-Id", "01a0d3f0-test:0")
	down.Set("X-Codex-Turn-Metadata", `{"session_id":"01a0d3f0-sess","thread_id":"01a0d3f0-thread","turn_trigger":"exec","model":"gpt-5.6-terra"}`)
	down.Set("X-Client-Request-Id", "01a0d3f0-req")
	down.Set("Session-Id", "01a0d3f0-sess")
	down.Set("Thread-Id", "01a0d3f0-thread")
	down.Set("X-Openai-Internal-Codex-Responses-Lite", "true")
	down.Set("Originator", "codex_exec")

	r := httptest.NewRequest("POST", "https://api.openai.com/v1/responses", nil)
	applyCodexHeadersFromSources(r, auth, "test-token", true, cfg, down)
	applyCodexCloakingHeaders(r.Header, cfg)

	t.Logf("=== CPA 计算出的上游头 ===")
	for k, vs := range r.Header {
		for _, v := range vs {
			t.Logf("  %s: %s", k, v)
		}
	}
}
