package cmd

import (
	"fmt"

	"github.com/armyknifelabs-platform/armyknife-cli/internal/client"
	"github.com/armyknifelabs-platform/armyknife-cli/internal/config"
	"github.com/armyknifelabs-platform/armyknife-cli/pkg/output"
	"github.com/spf13/cobra"
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Cache management commands",
	Long:  `Manage and monitor the application cache (Redis and PostgreSQL)`,
}

var cacheStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Display cache statistics",
	Long:  `Show cache hit/miss rates and performance metrics`,
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

		output.Header("Cache Statistics")
		output.Info("Fetching cache stats...")

		resp, err := c.Get("/cache/stats")
		if err != nil {
			return fmt.Errorf("failed to fetch cache stats: %w", err)
		}

		return output.JSON(resp.Data)
	},
}

var cacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear cache",
	Long:  `Clear all cached data (use with caution)`,
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

		output.Warning("⚠️  This will clear all cached data")
		output.Info("Clearing cache...")

		resp, err := c.Post("/cache/clear", nil)
		if err != nil {
			return fmt.Errorf("failed to clear cache: %w", err)
		}

		output.Success("✅ Cache cleared successfully")
		if jsonOut {
			return output.JSON(resp)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(cacheCmd)
	cacheCmd.AddCommand(cacheStatsCmd)
	cacheCmd.AddCommand(cacheClearCmd)

	cacheClearCmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output raw JSON")
}
