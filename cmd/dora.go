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

var (
	owner     string
	repo      string
	timeRange string
	jsonOut   bool
)

var doraCmd = &cobra.Command{
	Use:   "dora",
	Short: "DORA metrics commands",
	Long:  `Retrieve and display DORA (DevOps Research and Assessment) metrics for repositories`,
}

var doraGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get DORA metrics for a repository",
	Long: `Retrieve DORA metrics including:
- Deployment Frequency
- Lead Time for Changes
- Time to Restore Service
- Change Failure Rate`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if owner == "" || repo == "" {
			return fmt.Errorf("both --owner and --repo flags are required")
		}

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

		path := fmt.Sprintf("/metrics/dora?owner=%s&repo=%s", owner, repo)
		if timeRange != "" {
			path += fmt.Sprintf("&timeRange=%s", timeRange)
		}

		output.Header(fmt.Sprintf("DORA Metrics: %s/%s", owner, repo))
		output.Info("Fetching metrics...")

		resp, err := c.Get(path)
		if err != nil {
			return fmt.Errorf("failed to fetch DORA metrics: %w", err)
		}

		if jsonOut {
			return output.JSON(resp)
		}

		var metrics types.DORAMetrics
		if err := json.Unmarshal(resp.Data, &metrics); err != nil {
			return fmt.Errorf("failed to parse metrics: %w", err)
		}

		// Display metrics in a pretty format
		fmt.Println()
		if metrics.DeploymentFrequency != nil {
			output.Info("📦 Deployment Frequency")
			fmt.Printf("   %.2f deployments/day - %s %s\n",
				metrics.DeploymentFrequency.DeploymentsPerDay,
				getRatingEmoji(metrics.DeploymentFrequency.Rating),
				metrics.DeploymentFrequency.Rating)
			fmt.Println()
		}

		if metrics.LeadTimeForChanges != nil {
			output.Info("⏱️  Lead Time for Changes")
			fmt.Printf("   %.2f hours - %s %s\n",
				metrics.LeadTimeForChanges.AverageHours,
				getRatingEmoji(metrics.LeadTimeForChanges.Rating),
				metrics.LeadTimeForChanges.Rating)
			fmt.Println()
		}

		if metrics.TimeToRestoreService != nil {
			output.Info("🔧 Time to Restore Service")
			fmt.Printf("   %.2f hours - %s %s\n",
				metrics.TimeToRestoreService.AverageHours,
				getRatingEmoji(metrics.TimeToRestoreService.Rating),
				metrics.TimeToRestoreService.Rating)
			fmt.Println()
		}

		if metrics.ChangeFailureRate != nil {
			output.Info("❌ Change Failure Rate")
			fmt.Printf("   %.2f%% - %s %s\n",
				metrics.ChangeFailureRate.Percentage,
				getRatingEmoji(metrics.ChangeFailureRate.Rating),
				metrics.ChangeFailureRate.Rating)
			fmt.Println()
		}

		// Show metadata
		if resp.Metadata != nil {
			if resp.Metadata.Source == "cache" {
				output.Success("⚡ Loaded from cache")
			} else {
				output.Info("🔐 Loaded from GitHub API")
			}
		}

		return nil
	},
}

func getRatingEmoji(rating string) string {
	switch rating {
	case "Elite":
		return "🏆"
	case "High":
		return "🌟"
	case "Medium":
		return "⭐"
	case "Low":
		return "📉"
	default:
		return "❓"
	}
}

func init() {
	rootCmd.AddCommand(doraCmd)
	doraCmd.AddCommand(doraGetCmd)

	doraGetCmd.Flags().StringVarP(&owner, "owner", "o", "", "Repository owner (required)")
	doraGetCmd.Flags().StringVarP(&repo, "repo", "r", "", "Repository name (required)")
	doraGetCmd.Flags().StringVarP(&timeRange, "time-range", "t", "30d", "Time range (7d, 30d, 90d)")
	doraGetCmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output raw JSON")
}
