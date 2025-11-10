package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/armyknifelabs-platform/armyknife-cli/internal/config"
	"github.com/armyknifelabs-platform/armyknife-cli/pkg/output"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication commands",
	Long:  `Manage authentication with the ArmyKnife platform using API keys`,
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with the ArmyKnife platform",
	Long: `Authenticate with the ArmyKnife platform using an API key or GitHub PAT.

Option 1: Use an existing API key (from web dashboard)
  armyknife auth login
  armyknife auth login --api-key "ak_your_key_here"

Option 2: Use GitHub Personal Access Token (for CI/CD)
  armyknife auth login --github-pat "ghp_your_pat_here"
  export GITHUB_PAT="ghp_your_pat_here" && armyknife auth login

The GitHub PAT will be exchanged for an API key automatically.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Override API URL if provided via flag
		if apiURL != "" {
			cfg.APIURL = apiURL
		}

		// Check if GitHub PAT is provided (for CI/CD flow)
		githubPAT, _ := cmd.Flags().GetString("github-pat")
		if githubPAT == "" {
			// Check environment variable
			githubPAT = os.Getenv("GITHUB_PAT")
		}

		if githubPAT != "" {
			// Exchange GitHub PAT for API key
			return exchangePATForAPIKey(cfg, githubPAT)
		}

		// Otherwise, use direct API key flow
		apiKey, _ := cmd.Flags().GetString("api-key")
		if apiKey == "" {
			output.Header("ArmyKnife CLI Authentication")
			output.Info("To get an API key:")
			output.Info("1. Log in to https://test.armyknifelabs.com")
			output.Info("2. Go to Settings > API Keys")
			output.Info("3. Generate a new API key")
			output.Info("4. Copy and paste it below")
			output.Info("")
			output.Info("Or use GitHub PAT: armyknife auth login --github-pat <pat>")
			output.Info("")
			fmt.Print("Enter your API key: ")
			fmt.Scanln(&apiKey)
		}

		if apiKey == "" {
			return fmt.Errorf("API key is required")
		}

		// Validate API key format
		if len(apiKey) < 10 || !isValidAPIKey(apiKey) {
			return fmt.Errorf("invalid API key format")
		}

		// Save API key to config
		cfg.AccessToken = apiKey
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		output.Success("✅ API key saved successfully")
		output.Info("You can now use the ArmyKnife CLI to interact with the platform")
		return nil
	},
}

func isValidAPIKey(key string) bool {
	// API keys should start with "ak_"
	return len(key) > 3 && key[:3] == "ak_"
}

func exchangePATForAPIKey(cfg *config.Config, githubPAT string) error {
	output.Header("Exchanging GitHub PAT for API Key")
	output.Info("Validating GitHub PAT...")

	// Prepare request body
	reqBody := map[string]string{
		"github_pat": githubPAT,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to prepare request: %w", err)
	}

	// Make request to PAT exchange endpoint
	url := fmt.Sprintf("%s/auth/pat/exchange", cfg.APIURL)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to contact API: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var result struct {
		Success bool `json:"success"`
		Data    struct {
			APIKey    string `json:"api_key"`
			KeyID     string `json:"key_id"` // UUID string, not int
			KeyName   string `json:"key_name"`
			ExpiresAt string `json:"expires_at"`
			User      struct {
				ID       int    `json:"id"`
				Username string `json:"username"`
			} `json:"user"`
		} `json:"data"`
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("PAT exchange failed: %s - %s", result.Error.Code, result.Error.Message)
	}

	// Save API key and expiry to config
	cfg.AccessToken = result.Data.APIKey
	cfg.TokenExpiry = result.Data.ExpiresAt
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	output.Success("✅ Successfully authenticated!")
	output.Info(fmt.Sprintf("User: %s", result.Data.User.Username))
	output.Info(fmt.Sprintf("API Key ID: %s", result.Data.KeyID))
	output.Info(fmt.Sprintf("Key Name: %s", result.Data.KeyName))
	output.Info(fmt.Sprintf("Expires: %s", formatExpiryDate(result.Data.ExpiresAt)))
	output.Info("")
	output.Info("You can now use the ArmyKnife CLI to interact with the platform")

	return nil
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout and clear stored credentials",
	Long:  `Remove stored authentication credentials from local config`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		cfg.AccessToken = ""
		cfg.RefreshToken = ""
		cfg.TokenExpiry = ""

		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		output.Success("✅ Logged out successfully")
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check authentication status",
	Long:  `Display current authentication status and token information`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		output.Header("Authentication Status")

		if cfg.IsAuthenticated() {
			output.Success("✅ Authenticated")
			output.Table(map[string]string{
				"API URL":      cfg.APIURL,
				"Token Expiry": formatExpiryDate(cfg.TokenExpiry),
			})
		} else {
			output.Warning("❌ Not authenticated")
			output.Info("\nRun 'armyknife auth login' to authenticate")
		}

		return nil
	},
}

// formatExpiryDate formats an ISO 8601 date string into a human-readable format
// Input: "2026-11-09T19:45:30.123Z"
// Output: "November 9, 2026 at 7:45 PM UTC (363 days from now)"
func formatExpiryDate(isoDate string) string {
	if isoDate == "" {
		return "Unknown"
	}

	// Parse the ISO 8601 date
	expiryTime, err := time.Parse(time.RFC3339, isoDate)
	if err != nil {
		return isoDate // Return original if parsing fails
	}

	// Format: "January 2, 2006 at 3:04 PM MST"
	formatted := expiryTime.Format("January 2, 2006 at 3:04 PM MST")

	// Calculate days until expiry
	now := time.Now()
	duration := expiryTime.Sub(now)
	days := int(duration.Hours() / 24)

	if days < 0 {
		return fmt.Sprintf("%s (EXPIRED)", formatted)
	} else if days == 0 {
		return fmt.Sprintf("%s (expires today)", formatted)
	} else if days == 1 {
		return fmt.Sprintf("%s (1 day from now)", formatted)
	} else {
		return fmt.Sprintf("%s (%d days from now)", formatted, days)
	}
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(logoutCmd)
	authCmd.AddCommand(statusCmd)

	// Add flags for login command
	loginCmd.Flags().StringP("api-key", "k", "", "API key for authentication")
	loginCmd.Flags().StringP("github-pat", "p", "", "GitHub Personal Access Token (for CI/CD)")
}
