package support

import (
	"errors"
	"fmt"
	"io"
	"runtime"
	"strings"

	"main/utils/structs"
)

func WrapBackendConnectionError(kind string, addr string, err error, cfg structs.ConfigSet) error {
	if err == nil {
		return nil
	}

	base := fmt.Sprintf("%s unavailable at %s", kind, addr)
	lower := strings.ToLower(err.Error())
	if strings.Contains(lower, "actively refused") || strings.Contains(lower, "connection refused") {
		switch strings.ToLower(strings.TrimSpace(cfg.BackendMode)) {
		case "docker":
			if runtime.GOOS == "windows" {
				return fmt.Errorf("%s: start Docker Desktop, run .\\wrapper-start.ps1, then verify with .\\amdl.exe doctor: %w", base, err)
			}
			return fmt.Errorf("%s: start the wrapper container and verify the port is exposed: %w", base, err)
		case "wsl":
			return fmt.Errorf("%s: start the wrapper in WSL with -H 0.0.0.0 and verify Windows can reach %s: %w", base, addr, err)
		default:
			return fmt.Errorf("%s: start the wrapper backend and verify %s is reachable: %w", base, addr, err)
		}
	}

	return fmt.Errorf("%s: %w", base, err)
}

func WrapDecryptRuntimeError(err error) error {
	if err == nil {
		return nil
	}

	lower := strings.ToLower(err.Error())
	if errors.Is(err, io.EOF) || strings.Contains(lower, "connection reset by peer") {
		return fmt.Errorf("decrypt backend closed the connection unexpectedly (wrapper often logs 'Invalid CKC error'; common causes are expired login/session, storefront mismatch/preview-only availability, or proxy/VPN interference). Re-run .\\amdl.exe login, restart the wrapper, and confirm the track is available in your Apple Music storefront/account: %w", err)
	}

	return err
}
