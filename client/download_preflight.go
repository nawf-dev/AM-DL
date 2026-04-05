package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"main/internal/session"
)

const backendAccountAddr = "127.0.0.1:30020"

type downloadPreflightFinding struct {
	Status  string
	Message string
}

type wrapperAccountInfo struct {
	StorefrontID string `json:"storefront_id"`
	DevToken     string `json:"dev_token"`
	MusicToken   string `json:"music_token"`
}

func runDownloadPreflight(args []string) error {
	urls := collectAppleMusicURLs(args)
	if len(urls) == 0 {
		return nil
	}

	findings := []downloadPreflightFinding{}
	hasBlocking := false
	comparisonStorefront := strings.ToLower(strings.TrimSpace(Config.Storefront))
	comparisonLabel := "config"

	needsWrapper := downloadNeedsWrapper(urls)
	if needsWrapper {
		state := session.Detect()
		if state.Cached {
			findings = append(findings, downloadPreflightFinding{Status: "ok", Message: fmt.Sprintf("Login session cache found: %s", state.CacheDir)})
		} else {
			findings = append(findings, downloadPreflightFinding{Status: "fail", Message: "Login session cache missing. Run `amdl.exe login` before downloading lossless tracks."})
			hasBlocking = true
		}

		for _, endpoint := range []struct {
			name string
			addr string
		}{
			{name: "Decrypt backend", addr: Config.DecryptM3u8Port},
			{name: "M3U8 backend", addr: Config.GetM3u8Port},
			{name: "Account backend", addr: backendAccountAddr},
		} {
			if err := testTCP(endpoint.addr); err != nil {
				findings = append(findings, downloadPreflightFinding{Status: "fail", Message: fmt.Sprintf("%s unreachable at %s. Start the wrapper before downloading lossless tracks.", endpoint.name, endpoint.addr)})
				hasBlocking = true
			} else {
				findings = append(findings, downloadPreflightFinding{Status: "ok", Message: fmt.Sprintf("%s reachable: %s", endpoint.name, endpoint.addr)})
			}
		}

		accountInfo, err := fetchWrapperAccountInfo()
		if err != nil {
			findings = append(findings, downloadPreflightFinding{Status: "fail", Message: fmt.Sprintf("Wrapper account info check failed: %v", err)})
			hasBlocking = true
		} else {
			if strings.TrimSpace(accountInfo.StorefrontID) == "" || strings.TrimSpace(accountInfo.MusicToken) == "" {
				findings = append(findings, downloadPreflightFinding{Status: "fail", Message: "Wrapper account info is incomplete. Re-run `amdl.exe login` and restart the wrapper."})
				hasBlocking = true
			} else {
				findings = append(findings, downloadPreflightFinding{Status: "ok", Message: fmt.Sprintf("Wrapper account info available (storefront_id: %s)", accountInfo.StorefrontID)})
				if accountStorefront := storefrontCodeFromStorefrontID(accountInfo.StorefrontID); accountStorefront != "" {
					comparisonStorefront = accountStorefront
					comparisonLabel = "account"
				}
			}
		}
	}

	seenWarnings := map[string]bool{}
	for _, rawURL := range urls {
		urlStorefront := requestedStorefront(rawURL)
		if urlStorefront == "" || comparisonStorefront == "" || urlStorefront == comparisonStorefront {
			continue
		}
		message := fmt.Sprintf("URL storefront %q differs from %s storefront %q. This often causes lyrics gaps and can lead to CKC/decrypt failures if the release is not available to your account storefront.", strings.ToUpper(urlStorefront), comparisonLabel, strings.ToUpper(comparisonStorefront))
		if !seenWarnings[message] {
			findings = append(findings, downloadPreflightFinding{Status: "warn", Message: message})
			seenWarnings[message] = true
		}
	}

	if len(findings) == 0 {
		return nil
	}

	fmt.Println("Pre-download check:")
	for _, finding := range findings {
		prefix := "✅"
		switch finding.Status {
		case "warn":
			prefix = "⚠"
		case "fail":
			prefix = "❌"
		}
		fmt.Printf("%s %s\n", prefix, finding.Message)
	}

	if hasBlocking {
		return fmt.Errorf("pre-download check failed; fix the blocking issue(s) above before downloading")
	}

	return nil
}

func collectAppleMusicURLs(args []string) []string {
	urls := make([]string, 0, len(args))
	seen := map[string]bool{}
	for _, arg := range args {
		trimmed := strings.TrimSpace(arg)
		if !looksLikeAppleMusicURL(trimmed) || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		urls = append(urls, trimmed)
	}
	return urls
}

func downloadNeedsWrapper(urls []string) bool {
	if dl_aac && Config.AacType == "aac-lc" {
		for _, rawURL := range urls {
			if strings.Contains(rawURL, "/music-video/") {
				return true
			}
		}
		return false
	}

	for _, rawURL := range urls {
		if strings.Contains(rawURL, "/music-video/") {
			continue
		}
		return true
	}
	return false
}

func requestedStorefront(rawURL string) string {
	parts := strings.Split(strings.Trim(rawURL, "/"), "/")
	for i, part := range parts {
		if part == "music.apple.com" && i+1 < len(parts) {
			candidate := strings.ToLower(strings.TrimSpace(parts[i+1]))
			if len(candidate) == 2 {
				return candidate
			}
		}
	}
	return ""
}

func fetchWrapperAccountInfo() (wrapperAccountInfo, error) {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://" + backendAccountAddr)
	if err != nil {
		return wrapperAccountInfo{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return wrapperAccountInfo{}, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var info wrapperAccountInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return wrapperAccountInfo{}, err
	}
	return info, nil
}
