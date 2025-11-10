package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/armyknifelabs-platform/armyknife-cli/internal/client"
	"github.com/armyknifelabs-platform/armyknife-cli/internal/config"
	"github.com/armyknifelabs-platform/armyknife-cli/internal/types"
	"github.com/armyknifelabs-platform/armyknife-cli/pkg/output"
	"github.com/spf13/cobra"
)

var githubCmd = &cobra.Command{
	Use:   "github",
	Short: "GitHub operations",
	Long:  `Interact with GitHub repositories and data`,
}

var reposCmd = &cobra.Command{
	Use:   "repos",
	Short: "List user repositories",
	Long:  `List all repositories accessible to the authenticated user`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if !cfg.IsAuthenticated() {
			return fmt.Errorf("not authenticated. Run 'armyknife auth login' first")
		}

		if apiURL != "" {
			cfg.APIURL = apiURL
		}

		c := client.NewClient(cfg)

		output.Header("User Repositories")
		output.Info("Fetching repositories...")

		resp, err := c.Get("/github/user/repositories")
		if err != nil {
			return fmt.Errorf("failed to fetch repositories: %w", err)
		}

		var repos []types.Repository
		if err := json.Unmarshal(resp.Data, &repos); err != nil {
			return fmt.Errorf("failed to parse repositories: %w", err)
		}

		fmt.Println()
		for _, repo := range repos {
			fmt.Printf("📦 %s/%s (ID: %d)\n", repo.Owner, repo.Repo, repo.ID)
		}

		output.Info(fmt.Sprintf("\nTotal: %d repositories", len(repos)))
		return nil
	},
}

var syncCmd = &cobra.Command{
	Use:   "sync [repo-id]",
	Short: "Sync repository data",
	Long:  `Synchronize repository data from GitHub`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoID := args[0]

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if !cfg.IsAuthenticated() {
			return fmt.Errorf("not authenticated. Run 'armyknife auth login' first")
		}

		if apiURL != "" {
			cfg.APIURL = apiURL
		}

		c := client.NewClient(cfg)

		output.Header(fmt.Sprintf("Syncing Repository ID: %s", repoID))
		output.Info("Initiating sync...")

		resp, err := c.Post(fmt.Sprintf("/github/repos/%s/sync", repoID), nil)
		if err != nil {
			return fmt.Errorf("failed to sync repository: %w", err)
		}

		output.Success("✅ Sync completed successfully")
		if jsonOut {
			return output.JSON(resp)
		}

		return nil
	},
}

var rateLimitCmd = &cobra.Command{
	Use:   "rate-limit",
	Short: "Check GitHub API rate limit",
	Long:  `Display current GitHub API rate limit status`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if !cfg.IsAuthenticated() {
			return fmt.Errorf("not authenticated. Run 'armyknife auth login' first")
		}

		if apiURL != "" {
			cfg.APIURL = apiURL
		}

		c := client.NewClient(cfg)

		output.Header("GitHub API Rate Limit")

		resp, err := c.Get("/github/rate-limit")
		if err != nil {
			return fmt.Errorf("failed to fetch rate limit: %w", err)
		}

		var rateLimit types.RateLimitStatus
		if err := json.Unmarshal(resp.Data, &rateLimit); err != nil {
			return fmt.Errorf("failed to parse rate limit: %w", err)
		}

		fmt.Println()
		output.Table(map[string]string{
			"Remaining": fmt.Sprintf("%d", rateLimit.Remaining),
			"Limit":     fmt.Sprintf("%d", rateLimit.Limit),
			"Reset At":  rateLimit.ResetAt,
			"Reset In":  fmt.Sprintf("%d seconds", rateLimit.ResetIn),
		})

		percentUsed := rateLimit.PercentUsed
		if percentUsed < 50 {
			output.Success(fmt.Sprintf("\n✅ %.1f%% used - Healthy", percentUsed))
		} else if percentUsed < 80 {
			output.Warning(fmt.Sprintf("\n⚠️  %.1f%% used - Moderate", percentUsed))
		} else {
			output.Error(fmt.Sprintf("\n❌ %.1f%% used - Critical", percentUsed))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(githubCmd)
	githubCmd.AddCommand(reposCmd)
	githubCmd.AddCommand(syncCmd)
	githubCmd.AddCommand(rateLimitCmd)

	syncCmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output raw JSON")
}
