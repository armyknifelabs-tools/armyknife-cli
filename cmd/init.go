package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type DiskSpace struct {
	MountPoint string
	Available  uint64
	Total      uint64
	Filesystem string
}

type InitConfig struct {
	ModelsPath      string   `json:"models_path"`
	VoiceServerPort int      `json:"voice_server_port"`
	AutoStartServer bool     `json:"auto_start_server"`
	DownloadModels  []string `json:"download_models,omitempty"`
}

var (
	initSkipPrompts   bool
	initModelsPath    string
	initAutoDownload  bool
	initServerPort    int
	initAutoStart     bool
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize ArmyKnife CLI with optimal configuration",
	Long: `Initialize ArmyKnife CLI for first-time setup:

1. Discovers largest available disk space for AI models
2. Offers to download recommended AI models from Hugging Face
3. Creates configuration file with optimal settings
4. Injects environment variables into shell config (.bashrc/.zshrc)
5. Sets up auto-start voice server on macOS boot (via launchd)

This command automates the entire developer setup process, eliminating
manual configuration and server management.

Examples:
  # Interactive setup (recommended for first-time)
  armyknife init

  # Auto-accept defaults and download all recommended models
  armyknife init --auto-download

  # Specify custom models path
  armyknife init --models-path /Volumes/External/.armyknife/models

  # Set up without auto-start (manual server control)
  armyknife init --no-auto-start`,
	Run: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().BoolVar(&initSkipPrompts, "skip-prompts", false, "Skip all prompts and use defaults")
	initCmd.Flags().StringVar(&initModelsPath, "models-path", "", "Custom path for AI models (auto-detected if not specified)")
	initCmd.Flags().BoolVar(&initAutoDownload, "auto-download", false, "Automatically download all recommended models")
	initCmd.Flags().IntVar(&initServerPort, "server-port", 8765, "Port for voice server")
	initCmd.Flags().BoolVar(&initAutoStart, "no-auto-start", false, "Do not set up auto-start on boot")
}

func runInit(cmd *cobra.Command, args []string) {
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("  🎯 ArmyKnife CLI - First-Time Setup Wizard")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()

	// Step 1: Discover disk space
	fmt.Println("📊 Step 1/5: Disk Space Discovery")
	fmt.Println(strings.Repeat("─", 60))

	diskSpaces, err := discoverDiskSpaces()
	if err != nil {
		fmt.Printf("⚠️  Warning: Could not analyze disk space: %v\n", err)
		diskSpaces = []DiskSpace{}
	}

	var modelsPath string
	if initModelsPath != "" {
		modelsPath = initModelsPath
		fmt.Printf("Using specified models path: %s\n", modelsPath)
	} else if len(diskSpaces) > 0 {
		modelsPath = selectModelsPath(diskSpaces, initSkipPrompts)
	} else {
		// Fallback to home directory
		homeDir, _ := os.UserHomeDir()
		modelsPath = filepath.Join(homeDir, ".armyknife", "models")
		fmt.Printf("Using default models path: %s\n", modelsPath)
	}

	// Create models directory
	if err := os.MkdirAll(modelsPath, 0755); err != nil {
		fmt.Printf("❌ Failed to create models directory: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Models directory created: %s\n\n", modelsPath)

	// Step 2: Model Download
	fmt.Println("🦜 Step 2/5: AI Model Setup")
	fmt.Println(strings.Repeat("─", 60))

	recommendedModels := getRecommendedModels()
	selectedModels := selectModels(recommendedModels, initAutoDownload, initSkipPrompts)

	if len(selectedModels) > 0 {
		fmt.Printf("\n📥 Downloading %d models to %s\n", len(selectedModels), modelsPath)
		fmt.Println("This may take some time depending on your internet connection...")
		fmt.Println()

		for i, model := range selectedModels {
			fmt.Printf("[%d/%d] Downloading %s (%s)...\n", i+1, len(selectedModels), model.Name, model.Size)
			if err := downloadModel(model, modelsPath); err != nil {
				fmt.Printf("   ❌ Failed: %v\n", err)
			} else {
				fmt.Printf("   ✅ Downloaded successfully\n")
			}
			fmt.Println()
		}
	} else {
		fmt.Println("⏭️  Skipping model downloads (can be done later with `armyknife voice models download`)")
		fmt.Println()
	}

	// Step 3: Configuration File
	fmt.Println("📝 Step 3/5: Configuration File")
	fmt.Println(strings.Repeat("─", 60))

	config := InitConfig{
		ModelsPath:      modelsPath,
		VoiceServerPort: initServerPort,
		AutoStartServer: !initAutoStart,
		DownloadModels:  getModelNames(selectedModels),
	}

	if err := saveInitConfig(config); err != nil {
		fmt.Printf("❌ Failed to save configuration: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Configuration saved to ~/.armyknife/config.yaml")
	fmt.Println()

	// Step 4: Shell Environment Variables
	fmt.Println("🐚 Step 4/5: Shell Environment Setup")
	fmt.Println(strings.Repeat("─", 60))

	shellType, shellConfigPath := detectShell()
	if shellConfigPath != "" {
		fmt.Printf("Detected shell: %s\n", shellType)
		fmt.Printf("Config file: %s\n", shellConfigPath)

		if err := injectEnvVars(shellConfigPath, modelsPath, initServerPort); err != nil {
			fmt.Printf("❌ Failed to update shell config: %v\n", err)
		} else {
			fmt.Println("✅ Environment variables added to shell config")
			fmt.Println()
			fmt.Println("   Added variables:")
			fmt.Printf("   - ARMYKNIFE_MODELS_PATH=%s\n", modelsPath)
			fmt.Printf("   - ARMYKNIFE_VOICE_PORT=%d\n", initServerPort)
			fmt.Println()
			fmt.Printf("   ⚠️  Reload shell config with: source %s\n", shellConfigPath)
		}
	} else {
		fmt.Println("⚠️  Could not detect shell config file")
	}
	fmt.Println()

	// Step 5: macOS Auto-Start (launchd)
	if runtime.GOOS == "darwin" && !initAutoStart {
		fmt.Println("🚀 Step 5/5: macOS Auto-Start Setup")
		fmt.Println(strings.Repeat("─", 60))

		if err := setupLaunchd(modelsPath, initServerPort); err != nil {
			fmt.Printf("❌ Failed to set up auto-start: %v\n", err)
			fmt.Println("   You can manually start the server with: armyknife voice server")
		} else {
			fmt.Println("✅ Voice server configured to start automatically on boot")
			fmt.Println()
			fmt.Println("   launchd service: com.armyknifelabs.voice-server")
			fmt.Println("   Service commands:")
			fmt.Println("     - Start:   launchctl start com.armyknifelabs.voice-server")
			fmt.Println("     - Stop:    launchctl stop com.armyknifelabs.voice-server")
			fmt.Println("     - Status:  launchctl list | grep armyknife")
		}
		fmt.Println()
	} else if !initAutoStart {
		fmt.Println("🚀 Step 5/5: Auto-Start Setup")
		fmt.Println(strings.Repeat("─", 60))
		fmt.Println("⏭️  Auto-start is only supported on macOS (via launchd)")
		fmt.Println("   Start server manually with: armyknife voice server")
		fmt.Println()
	}

	// Final Summary
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("  ✅ Setup Complete!")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Reload your shell config or open a new terminal")
	fmt.Println("  2. Check voice service status: armyknife voice status")
	fmt.Println("  3. Test transcription: armyknife voice transcribe <audio-file>")
	fmt.Println()
	fmt.Println("Configuration:")
	fmt.Printf("  Models: %s\n", modelsPath)
	fmt.Printf("  Server: http://localhost:%d\n", initServerPort)
	if !initAutoStart && runtime.GOOS == "darwin" {
		fmt.Println("  Auto-start: Enabled (launchd)")
	} else {
		fmt.Println("  Auto-start: Manual")
	}
	fmt.Println()
}

// discoverDiskSpaces finds all mounted filesystems and their available space
func discoverDiskSpaces() ([]DiskSpace, error) {
	var diskSpaces []DiskSpace

	switch runtime.GOOS {
	case "darwin", "linux":
		// Use df command to get disk info
		cmd := exec.Command("df", "-k")
		output, err := cmd.Output()
		if err != nil {
			return nil, err
		}

		lines := strings.Split(string(output), "\n")
		for i, line := range lines {
			if i == 0 {
				continue // Skip header
			}
			fields := strings.Fields(line)
			if len(fields) < 6 {
				continue
			}

			// Parse available space (in KB)
			var available, total uint64
			fmt.Sscanf(fields[3], "%d", &available)
			fmt.Sscanf(fields[1], "%d", &total)

			// Convert KB to bytes
			available *= 1024
			total *= 1024

			mountPoint := fields[len(fields)-1]

			// Skip system/virtual filesystems
			if strings.HasPrefix(mountPoint, "/dev") ||
				strings.HasPrefix(mountPoint, "/sys") ||
				strings.HasPrefix(mountPoint, "/proc") ||
				strings.HasPrefix(mountPoint, "/run") ||
				mountPoint == "/boot" {
				continue
			}

			diskSpaces = append(diskSpaces, DiskSpace{
				MountPoint: mountPoint,
				Available:  available,
				Total:      total,
				Filesystem: fields[0],
			})
		}
	default:
		return nil, fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	// Sort by available space (largest first)
	sort.Slice(diskSpaces, func(i, j int) bool {
		return diskSpaces[i].Available > diskSpaces[j].Available
	})

	return diskSpaces, nil
}

// selectModelsPath lets user choose where to store models
func selectModelsPath(diskSpaces []DiskSpace, skipPrompts bool) string {
	fmt.Println("\nAvailable disk spaces:")
	fmt.Println()

	for i, disk := range diskSpaces {
		availableGB := float64(disk.Available) / (1024 * 1024 * 1024)
		totalGB := float64(disk.Total) / (1024 * 1024 * 1024)
		usagePercent := (1 - float64(disk.Available)/float64(disk.Total)) * 100

		fmt.Printf("  %d. %s\n", i+1, disk.MountPoint)
		fmt.Printf("     Available: %.2f GB / %.2f GB (%.1f%% used)\n", availableGB, totalGB, usagePercent)
		fmt.Printf("     Filesystem: %s\n", disk.Filesystem)
		fmt.Println()
	}

	if skipPrompts || len(diskSpaces) == 1 {
		selected := diskSpaces[0]
		modelsPath := filepath.Join(selected.MountPoint, ".armyknife", "models")
		fmt.Printf("Auto-selected: %s (%.2f GB available)\n", modelsPath, float64(selected.Available)/(1024*1024*1024))
		return modelsPath
	}

	// Prompt user
	fmt.Print("Select disk for AI models (1-%d) [1]: ", len(diskSpaces))
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		input = "1"
	}

	var choice int
	fmt.Sscanf(input, "%d", &choice)

	if choice < 1 || choice > len(diskSpaces) {
		choice = 1
	}

	selected := diskSpaces[choice-1]
	modelsPath := filepath.Join(selected.MountPoint, ".armyknife", "models")
	return modelsPath
}

type ModelInfo struct {
	Name        string
	Description string
	URL         string
	Filename    string
	Size        string
}

// getRecommendedModels returns list of recommended models for voice AI
func getRecommendedModels() []ModelInfo {
	return []ModelInfo{
		{
			Name:        "Whisper Medium Q5",
			Description: "Fast, high-quality English STT (recommended)",
			URL:         "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-medium-q5_0.bin",
			Filename:    "whisper-medium-q5_0.bin",
			Size:        "515 MB",
		},
		{
			Name:        "Whisper Large V3 Q8",
			Description: "Best quality English/multilingual STT",
			URL:         "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3-q8_0.bin",
			Filename:    "whisper-large-v3-q8_0.bin",
			Size:        "1.66 GB",
		},
		{
			Name:        "Parakeet TDT 0.6B v2",
			Description: "NVIDIA Parakeet - English STT (high accuracy)",
			URL:         "https://api.ngc.nvidia.com/v2/models/nvidia/nemo/parakeet_tdt_0_6b_v2/versions/1.0.0/files/parakeet_tdt_0.6b_v2.nemo",
			Filename:    "parakeet-tdt-0.6b-v2.nemo",
			Size:        "2.3 GB",
		},
		{
			Name:        "Parakeet TDT 0.6B v3",
			Description: "NVIDIA Parakeet - Multilingual (25 languages)",
			URL:         "https://api.ngc.nvidia.com/v2/models/nvidia/nemo/parakeet_tdt_0_6b_v3/versions/1.0.0/files/parakeet_tdt_0.6b_v3.nemo",
			Filename:    "parakeet-tdt-0.6b-v3.nemo",
			Size:        "2.3 GB",
		},
		{
			Name:        "Parakeet RNNT 1.1B",
			Description: "NVIDIA Parakeet - Large model (best quality)",
			URL:         "https://api.ngc.nvidia.com/v2/models/nvidia/nemo/parakeet_rnnt_1_1b/versions/1.0.0/files/parakeet_rnnt_1.1b.nemo",
			Filename:    "parakeet-rnnt-1.1b.nemo",
			Size:        "4.0 GB",
		},
	}
}

// selectModels lets user choose which models to download
func selectModels(models []ModelInfo, autoDownload, skipPrompts bool) []ModelInfo {
	if autoDownload {
		fmt.Println("Auto-download enabled: Downloading all recommended models")
		return models
	}

	if skipPrompts {
		// Default: download first 2 models (Whisper Medium and Parakeet TDT v2)
		fmt.Println("Auto-selected: Whisper Medium Q5 + Parakeet TDT 0.6B v2")
		return models[:2]
	}

	fmt.Println("\nRecommended AI Models:")
	fmt.Println()

	for i, model := range models {
		fmt.Printf("  %d. %s (%s)\n", i+1, model.Name, model.Size)
		fmt.Printf("     %s\n", model.Description)
		fmt.Println()
	}

	fmt.Println("Options:")
	fmt.Println("  1. Download all models (~10.6 GB)")
	fmt.Println("  2. Download recommended (Whisper Medium + Parakeet TDT v2) (~2.8 GB)")
	fmt.Println("  3. Choose specific models")
	fmt.Println("  4. Skip downloads (can download later)")
	fmt.Println()
	fmt.Print("Select option (1-4) [2]: ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		input = "2"
	}

	switch input {
	case "1":
		return models
	case "2":
		return models[:2]
	case "3":
		return selectSpecificModels(models)
	case "4":
		return []ModelInfo{}
	default:
		return models[:2]
	}
}

func selectSpecificModels(models []ModelInfo) []ModelInfo {
	fmt.Println("\nEnter model numbers to download (space-separated, e.g., 1 3 5):")
	fmt.Print("> ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return []ModelInfo{}
	}

	choices := strings.Fields(input)
	var selected []ModelInfo

	for _, choice := range choices {
		var idx int
		fmt.Sscanf(choice, "%d", &idx)
		if idx >= 1 && idx <= len(models) {
			selected = append(selected, models[idx-1])
		}
	}

	return selected
}

// downloadModel downloads a model from Hugging Face or NGC
func downloadModel(model ModelInfo, destDir string) error {
	destPath := filepath.Join(destDir, model.Filename)

	// Check if already exists
	if _, err := os.Stat(destPath); err == nil {
		return fmt.Errorf("already exists, skipping")
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Minute, // Large models need time
	}

	resp, err := client.Get(model.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// Create destination file
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Copy with progress
	_, err = io.Copy(out, resp.Body)
	return err
}

// saveInitConfig saves the initialization configuration
func saveInitConfig(config InitConfig) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(homeDir, ".armyknife")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	configPath := filepath.Join(configDir, "config.yaml")

	// Also update the JSON config if it exists
	jsonConfigPath := filepath.Join(configDir, "config.json")
	if _, err := os.Stat(jsonConfigPath); err == nil {
		// Read existing config
		data, _ := os.ReadFile(jsonConfigPath)
		var jsonConfig map[string]interface{}
		json.Unmarshal(data, &jsonConfig)

		// Add models_path
		jsonConfig["models_path"] = config.ModelsPath
		jsonConfig["voice_server_port"] = config.VoiceServerPort

		// Save updated config
		updatedData, _ := json.MarshalIndent(jsonConfig, "", "  ")
		os.WriteFile(jsonConfigPath, updatedData, 0600)
	}

	// Save YAML config
	yamlContent := fmt.Sprintf(`# ArmyKnife CLI Configuration (generated by init)
models_path: %s
voice_server_port: %d
auto_start_server: %t

# Downloaded models:
`, config.ModelsPath, config.VoiceServerPort, config.AutoStartServer)

	for _, model := range config.DownloadModels {
		yamlContent += fmt.Sprintf("# - %s\n", model)
	}

	return os.WriteFile(configPath, []byte(yamlContent), 0600)
}

// detectShell detects user's shell and returns config file path
func detectShell() (string, string) {
	shell := os.Getenv("SHELL")
	homeDir, _ := os.UserHomeDir()

	if strings.Contains(shell, "zsh") {
		return "zsh", filepath.Join(homeDir, ".zshrc")
	} else if strings.Contains(shell, "bash") {
		// Check for .bash_profile first (macOS), then .bashrc (Linux)
		bashProfile := filepath.Join(homeDir, ".bash_profile")
		bashrc := filepath.Join(homeDir, ".bashrc")

		if _, err := os.Stat(bashProfile); err == nil {
			return "bash", bashProfile
		}
		return "bash", bashrc
	}

	return "unknown", ""
}

// injectEnvVars adds environment variables to shell config
func injectEnvVars(shellConfigPath, modelsPath string, serverPort int) error {
	// Read existing config
	content, err := os.ReadFile(shellConfigPath)
	if err != nil {
		// File doesn't exist, create it
		content = []byte{}
	}

	configStr := string(content)

	// Check if already configured
	if strings.Contains(configStr, "ARMYKNIFE_MODELS_PATH") {
		return nil // Already configured
	}

	// Prepare new content
	newContent := fmt.Sprintf(`

# ===================================================
# ArmyKnife CLI Configuration (added by: armyknife init)
# ===================================================
export ARMYKNIFE_MODELS_PATH="%s"
export ARMYKNIFE_VOICE_PORT=%d

# Optional: Add armyknife to PATH if installed globally
# export PATH="$PATH:/usr/local/bin/armyknife"

`, modelsPath, serverPort)

	// Append to config
	f, err := os.OpenFile(shellConfigPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(newContent)
	return err
}

// setupLaunchd creates macOS launchd plist for auto-start
func setupLaunchd(modelsPath string, serverPort int) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	launchAgentsDir := filepath.Join(homeDir, "Library", "LaunchAgents")
	if err := os.MkdirAll(launchAgentsDir, 0755); err != nil {
		return err
	}

	plistPath := filepath.Join(launchAgentsDir, "com.armyknifelabs.voice-server.plist")

	// Get armyknife binary path
	armyknifePath, err := os.Executable()
	if err != nil {
		return err
	}

	// Create plist content
	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.armyknifelabs.voice-server</string>

	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
		<string>voice</string>
		<string>server</string>
		<string>--port</string>
		<string>%d</string>
		<string>--daemon</string>
	</array>

	<key>RunAtLoad</key>
	<true/>

	<key>KeepAlive</key>
	<dict>
		<key>SuccessfulExit</key>
		<false/>
	</dict>

	<key>StandardOutPath</key>
	<string>%s/Library/Logs/armyknife-voice-server.log</string>

	<key>StandardErrorPath</key>
	<string>%s/Library/Logs/armyknife-voice-server-error.log</string>

	<key>EnvironmentVariables</key>
	<dict>
		<key>ARMYKNIFE_MODELS_PATH</key>
		<string>%s</string>
		<key>ARMYKNIFE_VOICE_PORT</key>
		<string>%d</string>
	</dict>

	<key>WorkingDirectory</key>
	<string>%s</string>
</dict>
</plist>
`, armyknifePath, serverPort, homeDir, homeDir, modelsPath, serverPort, homeDir)

	// Write plist file
	if err := os.WriteFile(plistPath, []byte(plistContent), 0644); err != nil {
		return err
	}

	// Load the service
	cmd := exec.Command("launchctl", "load", plistPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to load launchd service: %w", err)
	}

	return nil
}

// getModelNames extracts just the names from ModelInfo slice
func getModelNames(models []ModelInfo) []string {
	names := make([]string, len(models))
	for i, model := range models {
		names[i] = model.Name
	}
	return names
}
