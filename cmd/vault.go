package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/armyknifelabs-platform/armyknife-cli/internal/client"
	"github.com/armyknifelabs-platform/armyknife-cli/internal/config"
	"github.com/armyknifelabs-platform/armyknife-cli/pkg/output"
	"github.com/spf13/cobra"
)

var vaultCmd = &cobra.Command{
	Use:   "vault",
	Short: "Manage secrets in HashiCorp Vault",
	Long: `Vault commands for managing secrets in the SEIP platform.
Supports listing, getting, setting, and syncing secrets from local .env files.`,
}

// vaultHealthCmd checks vault health
var vaultHealthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check Vault health and connection status",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if apiURL != "" {
			cfg.APIURL = apiURL
		}

		c := client.NewClient(cfg)

		output.Header("Vault Health Check")

		resp, err := c.Get("/vault/health")
		if err != nil {
			output.Error(fmt.Sprintf("❌ Vault health check failed: %v", err))
			return err
		}

		var result struct {
			Status    string `json:"status"`
			Connected bool   `json:"connected"`
			Message   string `json:"message"`
		}
		if err := json.Unmarshal(resp.Data, &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if result.Connected {
			output.Success(fmt.Sprintf("✅ Vault: %s", result.Status))
		} else {
			output.Error(fmt.Sprintf("❌ Vault: %s - %s", result.Status, result.Message))
		}

		return nil
	},
}

// vaultListCmd lists secrets at a path
var vaultListCmd = &cobra.Command{
	Use:   "list [path]",
	Short: "List secrets at a path",
	Long:  `List secret keys at a given path. If no path provided, lists at root.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if apiURL != "" {
			cfg.APIURL = apiURL
		}

		c := client.NewClient(cfg)

		path := ""
		if len(args) > 0 {
			path = args[0]
		}

		endpoint := "/vault/secrets"
		if path != "" {
			endpoint = fmt.Sprintf("/vault/secrets/%s", path)
		}

		output.Header(fmt.Sprintf("Secrets at: %s", path))

		resp, err := c.Get(endpoint)
		if err != nil {
			output.Error(fmt.Sprintf("❌ Failed to list secrets: %v", err))
			return err
		}

		var result struct {
			Path    string   `json:"path"`
			Secrets []string `json:"secrets"`
		}
		if err := json.Unmarshal(resp.Data, &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if len(result.Secrets) == 0 {
			output.Info("No secrets found at this path")
			return nil
		}

		for _, secret := range result.Secrets {
			if strings.HasSuffix(secret, "/") {
				output.Info(fmt.Sprintf("📁 %s", secret))
			} else {
				output.Info(fmt.Sprintf("🔐 %s", secret))
			}
		}

		output.Info(fmt.Sprintf("\nTotal: %d items", len(result.Secrets)))
		return nil
	},
}

// vaultGetCmd gets a specific secret
var vaultGetCmd = &cobra.Command{
	Use:   "get <path>",
	Short: "Get a secret's key-value pairs",
	Long:  `Retrieve and display all key-value pairs for a secret at the given path.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if apiURL != "" {
			cfg.APIURL = apiURL
		}

		c := client.NewClient(cfg)
		path := args[0]

		showValues, _ := cmd.Flags().GetBool("show-values")

		output.Header(fmt.Sprintf("Secret: %s", path))

		resp, err := c.Get(fmt.Sprintf("/vault/secret/%s", path))
		if err != nil {
			output.Error(fmt.Sprintf("❌ Failed to get secret: %v", err))
			return err
		}

		var result struct {
			Path   string            `json:"path"`
			Secret map[string]string `json:"secret"`
		}
		if err := json.Unmarshal(resp.Data, &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if len(result.Secret) == 0 {
			output.Info("No keys found in this secret")
			return nil
		}

		for key, value := range result.Secret {
			if showValues {
				output.Info(fmt.Sprintf("  %s = %s", key, value))
			} else {
				// Mask the value
				maskedValue := "****"
				if len(value) > 4 {
					maskedValue = value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
				}
				output.Info(fmt.Sprintf("  %s = %s", key, maskedValue))
			}
		}

		output.Info(fmt.Sprintf("\nTotal: %d keys", len(result.Secret)))
		if !showValues {
			output.Info("(use --show-values to reveal full values)")
		}
		return nil
	},
}

// vaultSetCmd sets a secret
var vaultSetCmd = &cobra.Command{
	Use:   "set <path> <key>=<value> [key2=value2 ...]",
	Short: "Set secret key-value pairs",
	Long:  `Create or update a secret with the provided key-value pairs.`,
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if apiURL != "" {
			cfg.APIURL = apiURL
		}

		c := client.NewClient(cfg)
		path := args[0]

		// Parse key=value pairs
		data := make(map[string]string)
		for _, arg := range args[1:] {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid format: %s (expected KEY=VALUE)", arg)
			}
			data[parts[0]] = parts[1]
		}

		patch, _ := cmd.Flags().GetBool("patch")

		output.Header(fmt.Sprintf("Setting secret: %s", path))

		body := map[string]interface{}{
			"data": data,
		}
		bodyBytes, _ := json.Marshal(body)

		var resp *client.APIResponse
		if patch {
			resp, err = c.Patch(fmt.Sprintf("/vault/secret/%s", path), bodyBytes)
		} else {
			resp, err = c.Post(fmt.Sprintf("/vault/secret/%s", path), bodyBytes)
		}

		if err != nil {
			output.Error(fmt.Sprintf("❌ Failed to set secret: %v", err))
			return err
		}

		var result struct {
			Path    string `json:"path"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(resp.Data, &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		output.Success(fmt.Sprintf("✅ %s", result.Message))
		for key := range data {
			output.Info(fmt.Sprintf("  - %s", key))
		}
		return nil
	},
}

// vaultDeleteCmd deletes a secret
var vaultDeleteCmd = &cobra.Command{
	Use:   "delete <path>",
	Short: "Delete a secret",
	Long:  `Delete a secret at the given path.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if apiURL != "" {
			cfg.APIURL = apiURL
		}

		c := client.NewClient(cfg)
		path := args[0]

		force, _ := cmd.Flags().GetBool("force")

		if !force {
			output.Warning(fmt.Sprintf("⚠️  Are you sure you want to delete secret at '%s'?", path))
			output.Info("Use --force to skip this confirmation")
			return nil
		}

		output.Header(fmt.Sprintf("Deleting secret: %s", path))

		resp, err := c.Delete(fmt.Sprintf("/vault/secret/%s", path))
		if err != nil {
			output.Error(fmt.Sprintf("❌ Failed to delete secret: %v", err))
			return err
		}

		var result struct {
			Path    string `json:"path"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(resp.Data, &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		output.Success(fmt.Sprintf("✅ %s", result.Message))
		return nil
	},
}

// vaultPushCmd pushes local .env file to vault
var vaultPushCmd = &cobra.Command{
	Use:   "push <env-file> <vault-path>",
	Short: "Push local .env file to Vault",
	Long: `Parse a local .env file and push all key-value pairs to a Vault secret path.
This is useful for syncing local development secrets to the platform.

Example:
  armyknife vault push .env.local production/myapp
  armyknife vault push ~/.secrets/api-keys production/api-keys --patch`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if apiURL != "" {
			cfg.APIURL = apiURL
		}

		c := client.NewClient(cfg)
		envFile := args[0]
		vaultPath := args[1]

		patch, _ := cmd.Flags().GetBool("patch")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		prefix, _ := cmd.Flags().GetString("prefix")
		exclude, _ := cmd.Flags().GetStringSlice("exclude")

		// Resolve file path
		if !filepath.IsAbs(envFile) {
			cwd, _ := os.Getwd()
			envFile = filepath.Join(cwd, envFile)
		}

		output.Header(fmt.Sprintf("Pushing %s → %s", filepath.Base(envFile), vaultPath))

		// Parse .env file
		secrets, err := parseEnvFile(envFile)
		if err != nil {
			output.Error(fmt.Sprintf("❌ Failed to parse env file: %v", err))
			return err
		}

		if len(secrets) == 0 {
			output.Warning("No secrets found in file")
			return nil
		}

		// Apply prefix filter
		if prefix != "" {
			filtered := make(map[string]string)
			for key, value := range secrets {
				if strings.HasPrefix(key, prefix) {
					filtered[key] = value
				}
			}
			secrets = filtered
			output.Info(fmt.Sprintf("Filtered to %d keys with prefix '%s'", len(secrets), prefix))
		}

		// Apply exclusions
		if len(exclude) > 0 {
			for _, pattern := range exclude {
				for key := range secrets {
					matched, _ := filepath.Match(pattern, key)
					if matched {
						delete(secrets, key)
					}
				}
			}
		}

		output.Info(fmt.Sprintf("Found %d secrets to push:", len(secrets)))
		for key := range secrets {
			output.Info(fmt.Sprintf("  • %s", key))
		}

		if dryRun {
			output.Warning("\n--dry-run: No changes made")
			return nil
		}

		// Push to vault
		body := map[string]interface{}{
			"data": secrets,
		}
		bodyBytes, _ := json.Marshal(body)

		var pushResp *client.APIResponse
		if patch {
			output.Info("\nUsing PATCH (merge with existing secrets)...")
			pushResp, err = c.Patch(fmt.Sprintf("/vault/secret/%s", vaultPath), bodyBytes)
		} else {
			output.Info("\nUsing POST (replace entire secret)...")
			pushResp, err = c.Post(fmt.Sprintf("/vault/secret/%s", vaultPath), bodyBytes)
		}

		if err != nil {
			output.Error(fmt.Sprintf("❌ Failed to push secrets: %v", err))
			return err
		}

		var pushResult struct {
			Path    string `json:"path"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(pushResp.Data, &pushResult); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		output.Success(fmt.Sprintf("\n✅ %s", pushResult.Message))
		output.Info(fmt.Sprintf("Pushed %d secrets to %s", len(secrets), vaultPath))
		return nil
	},
}

// vaultPullCmd pulls vault secrets to local .env file
var vaultPullCmd = &cobra.Command{
	Use:   "pull <vault-path> [output-file]",
	Short: "Pull Vault secrets to local .env file",
	Long: `Retrieve secrets from Vault and save them as a local .env file.
If no output file is specified, prints to stdout.

Example:
  armyknife vault pull production/myapp .env.local
  armyknife vault pull production/api-keys > api-keys.env`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if apiURL != "" {
			cfg.APIURL = apiURL
		}

		c := client.NewClient(cfg)
		vaultPath := args[0]
		outputFile := ""
		if len(args) > 1 {
			outputFile = args[1]
		}

		prefix, _ := cmd.Flags().GetString("prefix")

		resp, err := c.Get(fmt.Sprintf("/vault/secret/%s", vaultPath))
		if err != nil {
			output.Error(fmt.Sprintf("❌ Failed to pull secrets: %v", err))
			return err
		}

		var result struct {
			Path   string            `json:"path"`
			Secret map[string]string `json:"secret"`
		}
		if err := json.Unmarshal(resp.Data, &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if len(result.Secret) == 0 {
			output.Warning("No secrets found at this path")
			return nil
		}

		// Build .env content
		var envContent strings.Builder
		envContent.WriteString(fmt.Sprintf("# Pulled from Vault: %s\n", vaultPath))
		envContent.WriteString("# Generated by armyknife vault pull\n\n")

		for key, value := range result.Secret {
			// Apply prefix filter
			if prefix != "" && !strings.HasPrefix(key, prefix) {
				continue
			}
			// Quote values that contain special characters
			if strings.ContainsAny(value, " \t\n\"'$`\\") {
				value = fmt.Sprintf("\"%s\"", strings.ReplaceAll(value, "\"", "\\\""))
			}
			envContent.WriteString(fmt.Sprintf("%s=%s\n", key, value))
		}

		if outputFile == "" {
			// Print to stdout
			fmt.Print(envContent.String())
		} else {
			// Write to file
			if err := os.WriteFile(outputFile, []byte(envContent.String()), 0600); err != nil {
				output.Error(fmt.Sprintf("❌ Failed to write file: %v", err))
				return err
			}
			output.Success(fmt.Sprintf("✅ Pulled %d secrets to %s", len(result.Secret), outputFile))
		}

		return nil
	},
}

// parseEnvFile parses a .env file and returns key-value pairs
func parseEnvFile(filename string) (map[string]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	secrets := make(map[string]string)
	scanner := bufio.NewScanner(file)

	// Regex for KEY=VALUE, KEY="VALUE", KEY='VALUE'
	lineRegex := regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)=(.*)$`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		matches := lineRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		key := matches[1]
		value := matches[2]

		// Remove surrounding quotes
		if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
			(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
			value = value[1 : len(value)-1]
		}

		secrets[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return secrets, nil
}

func init() {
	rootCmd.AddCommand(vaultCmd)

	// Add subcommands
	vaultCmd.AddCommand(vaultHealthCmd)
	vaultCmd.AddCommand(vaultListCmd)
	vaultCmd.AddCommand(vaultGetCmd)
	vaultCmd.AddCommand(vaultSetCmd)
	vaultCmd.AddCommand(vaultDeleteCmd)
	vaultCmd.AddCommand(vaultPushCmd)
	vaultCmd.AddCommand(vaultPullCmd)

	// Flags for get command
	vaultGetCmd.Flags().Bool("show-values", false, "Show actual secret values (default is masked)")

	// Flags for set command
	vaultSetCmd.Flags().Bool("patch", false, "Patch existing secret instead of replacing")

	// Flags for delete command
	vaultDeleteCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")

	// Flags for push command
	vaultPushCmd.Flags().Bool("patch", false, "Merge with existing secrets instead of replacing")
	vaultPushCmd.Flags().Bool("dry-run", false, "Show what would be pushed without making changes")
	vaultPushCmd.Flags().String("prefix", "", "Only push keys with this prefix")
	vaultPushCmd.Flags().StringSlice("exclude", []string{}, "Exclude keys matching these patterns")

	// Flags for pull command
	vaultPullCmd.Flags().String("prefix", "", "Only pull keys with this prefix")
}
