// Package misc provides miscellaneous utility functions for the CLI Proxy API server.
package misc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	// codexFallbackVersion is reported when the npm registry has not been
	// fetched yet or cannot be reached. Bump it whenever a new codex release
	// lands so a failed fetch never leaves the upstream UA on a stale version.
	codexFallbackVersion = "0.160.0"
	codexNpmLatestURL    = "https://registry.npmjs.org/@openai/codex/latest"
	codexVersionCacheTTL = 6 * time.Hour
	codexVersionFetchTO  = 10 * time.Second
)

var (
	cachedCodexVersion = codexFallbackVersion
	codexVersionMu     sync.RWMutex
	codexUpdaterOnce   sync.Once
)

type codexNpmManifest struct {
	Version string `json:"version"`
}

// StartCodexVersionUpdater starts a background goroutine that periodically
// refreshes the cached official codex CLI version from the npm registry, so
// the upstream Codex User-Agent follows codex releases without a rebuild.
func StartCodexVersionUpdater(ctx context.Context) {
	codexUpdaterOnce.Do(func() {
		go runCodexVersionUpdater(ctx)
	})
}

func runCodexVersionUpdater(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	ticker := time.NewTicker(codexVersionCacheTTL / 2)
	defer ticker.Stop()
	log.Infof("periodic codex version refresh started (interval=%s)", codexVersionCacheTTL/2)
	refreshCodexVersion(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			refreshCodexVersion(ctx)
		}
	}
}

func refreshCodexVersion(ctx context.Context) {
	version, err := fetchCodexLatestVersion(ctx)
	codexVersionMu.Lock()
	defer codexVersionMu.Unlock()
	if err == nil && strings.TrimSpace(version) != "" {
		cachedCodexVersion = strings.TrimSpace(version)
		log.WithField("version", version).Info("fetched latest codex version")
	} else if err != nil {
		log.WithError(err).Debug("failed to fetch latest codex version; keeping cached/fallback value")
	}
}

func fetchCodexLatestVersion(ctx context.Context) (string, error) {
	reqCtx, cancel := context.WithTimeout(ctx, codexVersionFetchTO)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, codexNpmLatestURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	var manifest codexNpmManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return "", err
	}
	return manifest.Version, nil
}

// CodexVersion returns the currently cached official codex CLI version.
func CodexVersion() string {
	codexVersionMu.RLock()
	defer codexVersionMu.RUnlock()
	return cachedCodexVersion
}

// CodexUserAgent builds a codex_exec User-Agent matching the official codex
// CLI format: codex_exec/<ver> (<os> <ver>; <arch>) <term> (codex_exec; <ver>).
// The version is refreshed dynamically; OS/arch/terminal are detected at runtime.
func CodexUserAgent() string {
	v := CodexVersion()
	return fmt.Sprintf("codex_exec/%s (%s; %s) %s (codex_exec; %s)",
		v, codexOSName(), codexArch(), codexTerminal(), v)
}

func codexOSName() string {
	id, versionID := "", ""
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			switch {
			case strings.HasPrefix(line, "ID="):
				id = strings.Trim(strings.TrimPrefix(line, "ID="), `"`)
			case strings.HasPrefix(line, "VERSION_ID="):
				versionID = strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), `"`)
			}
		}
	}
	name := id
	switch id {
	case "ubuntu":
		name = "Ubuntu"
	case "debian":
		name = "Debian"
	case "centos":
		name = "CentOS"
	case "rhel":
		name = "RedHat"
	case "fedora":
		name = "Fedora"
	case "arch":
		name = "Arch"
	case "alpine":
		name = "Alpine"
	}
	if name == "" {
		name = "Linux"
	}
	// VERSION_ID like "22.04" renders as "22.4.0", matching the official client.
	parts := strings.Split(versionID, ".")
	major := "0"
	if len(parts) > 0 && parts[0] != "" {
		major = parts[0]
	}
	minor := "0"
	if len(parts) > 1 {
		if m := strings.TrimLeft(parts[1], "0"); m != "" {
			minor = m
		}
	}
	return fmt.Sprintf("%s %s.%s.0", name, major, minor)
}

func codexArch() string {
	switch runtime.GOARCH {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "aarch64"
	default:
		return runtime.GOARCH
	}
}

func codexTerminal() string {
	if term := os.Getenv("TERM"); term != "" {
		return term
	}
	return "xterm-256color"
}
