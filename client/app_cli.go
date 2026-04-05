package main

import (
	"fmt"
	"main/internal/appconfig"
	"main/internal/session"
	"main/internal/support"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"gopkg.in/yaml.v2"
)

type doctorResult struct {
	Name    string
	Status  string
	Details string
}

func main() {
	args := os.Args[1:]
	if err := runApp(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runApp(args []string) error {
	if len(args) == 0 {
		return runInteractiveHome()
	}

	switch args[0] {
	case "help", "--help", "-h":
		printRootUsage()
		return nil
	}

	if isLegacyInvocation(args) {
		runLegacyWithArgs(args)
		return nil
	}

	switch args[0] {
	case "setup":
		return runSetupWizard()
	case "doctor":
		return runDoctorCommand()
	case "search":
		return runSearchCommand(args[1:])
	case "download":
		return runDownloadCommand(args[1:])
	case "login":
		return runLoginCommand()
	case "logout":
		confirm := true
		if len(args) > 1 && (args[1] == "--yes" || args[1] == "-y") {
			confirm = false
		}
		return runLogoutCommand(confirm)
	case "backend":
		return runBackendCommand(args[1:])
	case "config":
		return runConfigCommand(args[1:])
	case "token":
		return runTokenCommand(args[1:])
	default:
		printRootUsage()
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func isLegacyInvocation(args []string) bool {
	if len(args) == 0 {
		return false
	}
	first := args[0]
	return strings.HasPrefix(first, "-") || looksLikeAppleMusicURL(first)
}

func looksLikeAppleMusicURL(value string) bool {
	return strings.Contains(value, "music.apple.com/")
}

func runLegacyWithArgs(args []string) {
	originalArgs := os.Args
	os.Args = append([]string{originalArgs[0]}, args...)
	defer func() {
		os.Args = originalArgs
	}()
	legacyMain()
}

func printRootUsage() {
	printBanner()
	fmt.Println("amdl - Apple Music Downloader")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  amdl")
	fmt.Println("  amdl setup")
	fmt.Println("  amdl doctor")
	fmt.Println("  amdl search [song|album|artist] [query]")
	fmt.Println("  amdl download <apple-music-url>")
	fmt.Println("  amdl login")
	fmt.Println("  amdl logout [--yes]")
	fmt.Println("  amdl token set")
	fmt.Println("  amdl token clear")
	fmt.Println("  amdl backend status")
	fmt.Println("  amdl backend guide")
	fmt.Println("  amdl config show")
	fmt.Println("  amdl config reset")
	fmt.Println("")
	fmt.Println("Legacy compatibility:")
	fmt.Println("  amdl --search album \"Taylor Swift\"")
	fmt.Println("  amdl --atmos <url>")
	fmt.Println("  amdl --song <url>")
}

func runInteractiveHome() error {
	printBanner()
	printHomeSummary()

	options := []string{
		"Search & Download",
		"Download from URL",
		"Setup Wizard",
		"Login to Apple Music",
		"Logout / Reset Session",
		"Set media-user-token",
		"Doctor Check",
		"Backend Status",
		"Backend Guide",
		"Show Config",
		"Reset Config",
		"Exit",
	}

	choice := ""
	prompt := &survey.Select{
		Message:  "Choose an action:",
		Options:  options,
		PageSize: len(options),
	}
	if err := survey.AskOne(prompt, &choice); err != nil {
		return nil
	}

	switch choice {
	case "Search & Download":
		return runInteractiveSearchDownload()
	case "Download from URL":
		url := ""
		if err := survey.AskOne(&survey.Input{Message: "Apple Music URL:"}, &url, survey.WithValidator(survey.Required)); err != nil {
			return nil
		}
		runLegacyWithArgs([]string{strings.TrimSpace(url)})
		return nil
	case "Setup Wizard":
		return runSetupWizard()
	case "Login to Apple Music":
		return runLoginCommand()
	case "Logout / Reset Session":
		return runLogoutCommand(true)
	case "Set media-user-token":
		return runTokenCommand([]string{"set"})
	case "Doctor Check":
		return runDoctorCommand()
	case "Backend Status":
		return printBackendStatus()
	case "Backend Guide":
		printBackendGuide()
		return nil
	case "Show Config":
		return runConfigCommand([]string{"show"})
	case "Reset Config":
		return runConfigCommand([]string{"reset"})
	default:
		return nil
	}
}

func runInteractiveSearchDownload() error {
	searchType := "album"
	query := ""
	if err := survey.AskOne(&survey.Select{
		Message: "Search type:",
		Options: []string{"album", "song", "artist"},
	}, &searchType); err != nil {
		return nil
	}
	if err := survey.AskOne(&survey.Input{Message: "Search query:"}, &query, survey.WithValidator(survey.Required)); err != nil {
		return nil
	}
	runLegacyWithArgs([]string{"--search", searchType, strings.TrimSpace(query)})
	return nil
}

func runSearchCommand(args []string) error {
	if len(args) == 0 {
		return runInteractiveSearchDownload()
	}
	searchType := args[0]
	if searchType != "album" && searchType != "song" && searchType != "artist" {
		return fmt.Errorf("invalid search type: %s", searchType)
	}
	if len(args) < 2 {
		return fmt.Errorf("search query is required")
	}
	query := strings.Join(args[1:], " ")
	runLegacyWithArgs([]string{"--search", searchType, query})
	return nil
}

func runDownloadCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("download URL is required")
	}
	runLegacyWithArgs(args)
	return nil
}

func runBackendCommand(args []string) error {
	if len(args) == 0 {
		return printBackendStatus()
	}
	switch args[0] {
	case "status":
		return printBackendStatus()
	case "guide":
		printBackendGuide()
		return nil
	default:
		return fmt.Errorf("unknown backend command: %s", args[0])
	}
}

func runConfigCommand(args []string) error {
	if len(args) == 0 || args[0] == "show" {
		cfg, err := appconfig.Load()
		if err != nil {
			return err
		}
		body, err := yaml.Marshal(&cfg)
		if err != nil {
			return err
		}
		fmt.Printf("Config path: %s\n\n%s", appconfig.Path(), string(body))
		return nil
	}
	if args[0] == "reset" {
		cfg := appconfig.Default()
		if err := appconfig.Save(cfg); err != nil {
			return err
		}
		fmt.Printf("Config reset: %s\n", appconfig.Path())
		return nil
	}
	return fmt.Errorf("unknown config command: %s", args[0])
}

func runTokenCommand(args []string) error {
	mode := "set"
	if len(args) > 0 {
		mode = args[0]
	}
	cfg := appconfig.LoadOrDefault()
	switch mode {
	case "set":
		value := ""
		if cfg.MediaUserToken != "your-media-user-token" {
			value = cfg.MediaUserToken
		}
		prompt := &survey.Password{Message: "Paste media-user-token (input hidden):"}
		if value != "" {
			prompt.Message = "Paste new media-user-token (leave empty to keep current):"
		}
		if err := survey.AskOne(prompt, &value); err != nil {
			return nil
		}
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			if cfg.MediaUserToken != "your-media-user-token" {
				fmt.Println("Existing media-user-token kept.")
				return nil
			}
			return fmt.Errorf("media-user-token cannot be empty")
		}
		cfg.MediaUserToken = trimmed
		if err := appconfig.Save(cfg); err != nil {
			return err
		}
		fmt.Println("media-user-token saved to local config.yaml")
		return nil
	case "clear":
		cfg.MediaUserToken = "your-media-user-token"
		if err := appconfig.Save(cfg); err != nil {
			return err
		}
		fmt.Println("media-user-token cleared from local config.yaml")
		return nil
	default:
		return fmt.Errorf("unknown token command: %s", mode)
	}
}

func runSetupWizard() error {
	cfg := appconfig.LoadOrDefault()

	modeLabel := map[string]string{
		"alac":  "Lossless (ALAC)",
		"aac":   "AAC",
		"atmos": "Dolby Atmos",
	}[cfg.DefaultDownloadMode]
	if modeLabel == "" {
		modeLabel = "Lossless (ALAC)"
	}
	backendLabel := map[string]string{
		"docker": "Docker",
		"wsl":    "WSL",
		"manual": "Manual",
	}[cfg.BackendMode]
	if backendLabel == "" {
		backendLabel = "Docker"
	}

	outDir := cfg.AlacSaveFolder
	if err := survey.AskOne(&survey.Input{Message: "ALAC download folder:", Default: outDir}, &outDir, survey.WithValidator(survey.Required)); err != nil {
		return nil
	}
	modeChoice := modeLabel
	if err := survey.AskOne(&survey.Select{Message: "Default download quality:", Options: []string{"Lossless (ALAC)", "AAC", "Dolby Atmos"}, Default: modeLabel}, &modeChoice); err != nil {
		return nil
	}
	storefront := cfg.Storefront
	if err := survey.AskOne(&survey.Input{Message: "Storefront (2-letter code):", Default: storefront}, &storefront, survey.WithValidator(survey.Required)); err != nil {
		return nil
	}
	mediaToken := cfg.MediaUserToken
	if mediaToken == "your-media-user-token" {
		mediaToken = ""
	}
	if err := survey.AskOne(&survey.Input{Message: "media-user-token (optional, needed for MV/AAC-LC):", Default: mediaToken}, &mediaToken); err != nil {
		return nil
	}
	backendChoice := backendLabel
	if err := survey.AskOne(&survey.Select{Message: "Backend mode:", Options: []string{"Docker", "WSL", "Manual"}, Default: backendLabel}, &backendChoice); err != nil {
		return nil
	}
	decryptHost := cfg.DecryptM3u8Port
	if err := survey.AskOne(&survey.Input{Message: "Decrypt backend host:port:", Default: decryptHost}, &decryptHost, survey.WithValidator(survey.Required)); err != nil {
		return nil
	}
	m3u8Host := cfg.GetM3u8Port
	if err := survey.AskOne(&survey.Input{Message: "M3U8 backend host:port:", Default: m3u8Host}, &m3u8Host, survey.WithValidator(survey.Required)); err != nil {
		return nil
	}

	cfg.AlacSaveFolder = strings.TrimSpace(outDir)
	cfg.Storefront = strings.ToLower(strings.TrimSpace(storefront))
	cfg.MediaUserToken = strings.TrimSpace(mediaToken)
	cfg.DecryptM3u8Port = strings.TrimSpace(decryptHost)
	cfg.GetM3u8Port = strings.TrimSpace(m3u8Host)
	if cfg.MediaUserToken == "" {
		cfg.MediaUserToken = "your-media-user-token"
	}

	switch modeChoice {
	case "AAC":
		cfg.DefaultDownloadMode = "aac"
	case "Dolby Atmos":
		cfg.DefaultDownloadMode = "atmos"
	default:
		cfg.DefaultDownloadMode = "alac"
	}
	switch backendChoice {
	case "WSL":
		cfg.BackendMode = "wsl"
	case "Manual":
		cfg.BackendMode = "manual"
	default:
		cfg.BackendMode = "docker"
	}

	if err := os.MkdirAll(cfg.AlacSaveFolder, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create output folder: %w", err)
	}
	if err := appconfig.Save(cfg); err != nil {
		return err
	}

	fmt.Println("Setup saved.")
	fmt.Printf("- config: %s\n", appconfig.Path())
	fmt.Printf("- backend mode: %s\n", cfg.BackendMode)
	fmt.Printf("- default quality: %s\n", cfg.DefaultDownloadMode)
	fmt.Printf("- output folder: %s\n", cfg.AlacSaveFolder)

	backendOK := testTCP(cfg.DecryptM3u8Port) == nil && testTCP(cfg.GetM3u8Port) == nil
	if backendOK {
		fmt.Println("- backend check: OK")
	} else {
		fmt.Println("- backend check: not reachable yet")
	}

	state := session.Detect()
	if state.Cached {
		fmt.Println("- login session: cached locally")
	} else {
		fmt.Println("- login session: not found")
		loginNow := false
		if err := survey.AskOne(&survey.Confirm{Message: "Log in to Apple Music now?", Default: true}, &loginNow); err == nil && loginNow {
			if err := runLoginCommand(); err != nil {
				return err
			}
		}
	}

	return nil
}

func runDoctorCommand() error {
	results := runDoctorChecks()
	hasFailure := false
	for _, result := range results {
		prefix := "✅"
		switch result.Status {
		case "warn":
			prefix = "⚠"
		case "fail":
			prefix = "❌"
			hasFailure = true
		}
		fmt.Printf("%s %s: %s\n", prefix, result.Name, result.Details)
	}
	if hasFailure {
		return fmt.Errorf("doctor found blocking issues")
	}
	return nil
}

func runDoctorChecks() []doctorResult {
	results := []doctorResult{}

	if err := appconfig.Validate(); err != nil {
		results = append(results, doctorResult{Name: "Config", Status: "fail", Details: err.Error()})
		cfg := appconfig.Default()
		results = append(results, doctorResult{Name: "Backend decrypt port", Status: statusFromError(testTCP(cfg.DecryptM3u8Port)), Details: tcpDetail(cfg.DecryptM3u8Port)})
		results = append(results, doctorResult{Name: "Backend m3u8 port", Status: statusFromError(testTCP(cfg.GetM3u8Port)), Details: tcpDetail(cfg.GetM3u8Port)})
	} else {
		cfg, _ := appconfig.Load()
		results = append(results, doctorResult{Name: "Config", Status: "ok", Details: fmt.Sprintf("Loaded %s", appconfig.Path())})
		results = append(results, doctorResult{Name: "Output folder", Status: outputFolderStatus(cfg.AlacSaveFolder), Details: outputFolderDetail(cfg.AlacSaveFolder)})
		results = append(results, doctorResult{Name: "Backend decrypt port", Status: statusFromError(testTCP(cfg.DecryptM3u8Port)), Details: tcpDetail(cfg.DecryptM3u8Port)})
		results = append(results, doctorResult{Name: "Backend m3u8 port", Status: statusFromError(testTCP(cfg.GetM3u8Port)), Details: tcpDetail(cfg.GetM3u8Port)})
		results = append(results, doctorResult{Name: "Backend account port", Status: statusFromError(testTCP("127.0.0.1:30020")), Details: tcpDetail("127.0.0.1:30020")})
	}

	state := session.Detect()
	if state.Cached {
		results = append(results, doctorResult{Name: "Login session", Status: "ok", Details: fmt.Sprintf("cached locally: %s", state.CacheDir)})
	} else {
		results = append(results, doctorResult{Name: "Login session", Status: "warn", Details: "missing; run `amdl login`"})
	}

	cfg := appconfig.LoadOrDefault()
	mediaTokenReady := len(strings.TrimSpace(cfg.MediaUserToken)) > 50 && cfg.MediaUserToken != "your-media-user-token"
	if mediaTokenReady {
		results = append(results, doctorResult{Name: "media-user-token", Status: "ok", Details: "configured for MV/AAC-LC features"})
	} else {
		results = append(results, doctorResult{Name: "media-user-token", Status: "warn", Details: "missing or placeholder; run `amdl token set`"})
	}
	if _, err := support.ResolveExecutable("mp4decrypt"); err == nil && mediaTokenReady {
		results = append(results, doctorResult{Name: "Music Video readiness", Status: "ok", Details: "mp4decrypt and media-user-token are available"})
	} else {
		results = append(results, doctorResult{Name: "Music Video readiness", Status: "warn", Details: "requires both mp4decrypt and a valid media-user-token (`amdl token set`)"})
	}

	results = append(results, executableCheck("MP4Box", true))
	results = append(results, executableCheck("ffmpeg", false))
	results = append(results, executableCheck("mp4decrypt", false))
	results = append(results, executableCheck("docker", false))

	if runtime.GOOS == "windows" {
		results = append(results, executableCheck("wsl", false))
	}

	return results
}

func executableCheck(name string, required bool) doctorResult {
	path, err := support.ResolveExecutable(name)
	if err == nil {
		return doctorResult{Name: name, Status: "ok", Details: fmt.Sprintf("available: %s", path)}
	}
	status := "warn"
	if required {
		status = "fail"
	}
	if name == "mp4decrypt" {
		return doctorResult{Name: name, Status: status, Details: "not found in PATH, beside amdl.exe, or in tools\\mp4decrypt.exe"}
	}
	return doctorResult{Name: name, Status: status, Details: "not found in PATH or local tools folder"}
}

func outputFolderStatus(path string) string {
	if path == "" {
		return "fail"
	}
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return "fail"
	}
	probe := filepath.Join(path, ".amdl-write-test")
	if err := os.WriteFile(probe, []byte("ok"), 0644); err != nil {
		return "fail"
	}
	_ = os.Remove(probe)
	return "ok"
}

func outputFolderDetail(path string) string {
	if path == "" {
		return "folder not configured"
	}
	return fmt.Sprintf("writable: %s", path)
}

func statusFromError(err error) string {
	if err == nil {
		return "ok"
	}
	return "warn"
}

func tcpDetail(addr string) string {
	if err := testTCP(addr); err != nil {
		return fmt.Sprintf("not reachable: %s", addr)
	}
	return fmt.Sprintf("reachable: %s", addr)
}

func testTCP(addr string) error {
	conn, err := net.DialTimeout("tcp", addr, 1200*time.Millisecond)
	if err != nil {
		return err
	}
	_ = conn.Close()
	return nil
}

func printBackendStatus() error {
	cfg := appconfig.LoadOrDefault()
	checks := []string{cfg.DecryptM3u8Port, cfg.GetM3u8Port, "127.0.0.1:30020"}
	labels := []string{"Decrypt", "M3U8", "Account"}
	for i, addr := range checks {
		if err := testTCP(addr); err != nil {
			fmt.Printf("❌ %s: %s unreachable\n", labels[i], addr)
		} else {
			fmt.Printf("✅ %s: %s reachable\n", labels[i], addr)
		}
	}
	fmt.Printf("Backend mode: %s\n", cfg.BackendMode)
	state := session.Detect()
	if state.Cached {
		fmt.Printf("Login session: cached (%s)\n", state.CacheDir)
	} else {
		fmt.Println("Login session: missing")
	}
	return nil
}

func printBackendGuide() {
	fmt.Println("Supported backend target: WorldObservationLog/wrapper")
	fmt.Println("")
	fmt.Println("Docker flow:")
	fmt.Println("  1. Start Docker Desktop")
	fmt.Println("  2. Run `amdl login` once to create a local session")
	fmt.Println("  3. Start wrapper via `wrapper-start.ps1` or your helper flow")
	fmt.Println("  4. Ensure ports 10020, 20020, and 30020 are exposed")
	fmt.Println("")
	fmt.Println("WSL flow:")
	fmt.Println("  1. Open WSL")
	fmt.Println("  2. Start wrapper with -H 0.0.0.0")
	fmt.Println("  3. Confirm Windows can reach 127.0.0.1:10020 and 127.0.0.1:20020")
	fmt.Println("")
	fmt.Println("Then run: amdl doctor")
}

func printBanner() {
	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║   amdl - Apple Music Downloader     ║")
	fmt.Println("╚══════════════════════════════════════╝")
}

func printHomeSummary() {
	cfg := appconfig.LoadOrDefault()
	backend := "offline"
	if testTCP(cfg.DecryptM3u8Port) == nil && testTCP(cfg.GetM3u8Port) == nil {
		backend = "online"
	}
	sessionStatus := "missing"
	if session.Detect().Cached {
		sessionStatus = "cached"
	}

	fmt.Printf("Storefront: %s | Default mode: %s | Backend mode: %s\n", strings.ToUpper(cfg.Storefront), describeMode(cfg.DefaultDownloadMode), cfg.BackendMode)
	fmt.Printf("Config: %s | Wrapper: %s | Session: %s\n\n", appconfig.Path(), backend, sessionStatus)
}

func describeMode(mode string) string {
	switch mode {
	case "aac":
		return "AAC"
	case "atmos":
		return "Dolby Atmos"
	default:
		return "Lossless (ALAC)"
	}
}

func runLoginCommand() error {
	scriptPath, err := session.ScriptPath("wrapper-login.ps1")
	if err != nil {
		return err
	}
	fmt.Println("Starting secure Apple Music login flow...")
	args := []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File", scriptPath}
	loginUser := strings.TrimSpace(os.Getenv("AMDL_LOGIN_USER"))
	loginPass := os.Getenv("AMDL_LOGIN_PASSWORD")
	if loginUser != "" && loginPass != "" {
		fmt.Println("Using one-time environment credentials for non-interactive login.")
		args = append(args, "-Username", loginUser, "-Password", loginPass, "-NonInteractive", "-NoPause")
	} else {
		args = append(args, "-NoPause")
	}
	cmd := exec.Command(powerShellBinary(), args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	if !session.Detect().Cached {
		return fmt.Errorf("login finished but no local session was detected")
	}
	fmt.Println("Local session cached successfully.")
	return nil
}

func runLogoutCommand(confirm bool) error {
	state := session.Detect()
	if confirm {
		approved := false
		message := "Delete the local Apple Music session cache and stop wrapper?"
		if state.Cached {
			message = fmt.Sprintf("Delete local session cache at %s and stop wrapper?", state.CacheDir)
		}
		if err := survey.AskOne(&survey.Confirm{Message: message, Default: false}, &approved); err != nil || !approved {
			return nil
		}
	}
	if err := session.StopWrapperContainer(); err != nil {
		return err
	}
	if err := session.ClearLocalSession(); err != nil {
		return err
	}
	fmt.Println("Local session removed. Run `amdl login` to sign in again.")
	return nil
}

func powerShellBinary() string {
	if _, err := exec.LookPath("pwsh"); err == nil {
		return "pwsh"
	}
	return "powershell"
}
