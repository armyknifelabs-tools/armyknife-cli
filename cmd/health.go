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

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check system health",
	Long:  `Check the health status of the ArmyKnife platform and its services`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if apiURL != "" {
			cfg.APIURL = apiURL
		}

		c := client.NewClient(cfg)

		output.Header("System Health Check")

		// Check backend health
		resp, err := c.Get("/health")
		if err != nil {
			output.Error(fmt.Sprintf("❌ Backend health check failed: %v", err))
			return err
		}

		var health types.HealthStatus
		if err := json.Unmarshal(resp.Data, &health); err != nil {
			return fmt.Errorf("failed to parse health response: %w", err)
		}

		if health.Status == "healthy" {
			output.Success("✅ Backend: Healthy")
		} else {
			output.Warning(fmt.Sprintf("⚠️  Backend: %s", health.Status))
		}

		// Check AI service health
		aiResp, err := c.Get("/ai/health")
		if err != nil {
			output.Warning(fmt.Sprintf("⚠️  AI Service: %v", err))
		} else {
			var aiHealth types.HealthStatus
			if err := json.Unmarshal(aiResp.Data, &aiHealth); err == nil {
				if aiHealth.Status == "healthy" {
					output.Success("✅ AI Service: Healthy")
				} else {
					output.Warning(fmt.Sprintf("⚠️  AI Service: %s", aiHealth.Status))
				}
			}
		}

		// Check rate limit status
		rateResp, err := c.Get("/github/rate-limit")
		if err != nil {
			output.Warning(fmt.Sprintf("⚠️  GitHub Rate Limit: %v", err))
		} else {
			var rateLimit types.RateLimitStatus
			if err := json.Unmarshal(rateResp.Data, &rateLimit); err == nil {
				output.Info(fmt.Sprintf("\n📊 GitHub Rate Limit: %d/%d remaining (%.1f%% used)",
					rateLimit.Remaining, rateLimit.Limit, rateLimit.PercentUsed))
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(healthCmd)
}
