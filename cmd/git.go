package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/armyknifelabs-platform/armyknife-cli/internal/client"
	"github.com/armyknifelabs-platform/armyknife-cli/internal/config"
	"github.com/armyknifelabs-platform/armyknife-cli/internal/types"
	"github.com/armyknifelabs-platform/armyknife-cli/pkg/output"
	"github.com/spf13/cobra"
)

// Provider icons/colors for display
var providerDisplay = map[types.GitProvider]struct {
	icon  string
	color string
}{
	types.ProviderGitHub:      {"🐙", "#24292e"},
	types.ProviderGitLab:      {"🦊", "#fc6d26"},
	types.ProviderBitbucket:   {"🪣", "#0052cc"},
	types.ProviderAzureDevOps: {"☁️", "#0078d4"},
}

var gitCmd = &cobra.Command{
	Use:   "git",
	Short: "Multi-provider Git operations",
	Long: `Interact with multiple Git providers (GitHub, GitLab, Bitbucket, Azure DevOps).

This command group provides unified access to repositories, pull requests, commits,
and pipelines across all connected Git providers.`,
}

// ============================================================
// PROVIDER CONNECTION COMMANDS
// ============================================================

var providersCmd = &cobra.Command{
	Use:   "providers",
	Short: "List available and connected Git providers",
	Long:  `Display all supported Git providers and their connection status`,
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

		output.Header("Git Providers")
		output.Info("Fetching provider status...")

		resp, err := c.Get("/git/providers")
		if err != nil {
			return fmt.Errorf("failed to fetch providers: %w", err)
		}

		var providers []types.ProviderInfo
		if err := json.Unmarshal(resp.Data, &providers); err != nil {
			return fmt.Errorf("failed to parse providers: %w", err)
		}

		fmt.Println()
		for _, p := range providers {
			display := providerDisplay[p.ID]
			status := "❌ Not Connected"
			if p.IsConnected {
				status = "✅ Connected"
			}
			fmt.Printf("%s %s (%s)\n", display.icon, p.DisplayName, status)
			fmt.Printf("   Capabilities: %s\n", strings.Join(p.Capabilities, ", "))
			fmt.Println()
		}

		return nil
	},
}

var connectionsCmd = &cobra.Command{
	Use:   "connections",
	Short: "List active provider connections",
	Long:  `Display all active connections to Git providers`,
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

		output.Header("Provider Connections")

		resp, err := c.Get("/git/connections")
		if err != nil {
			return fmt.Errorf("failed to fetch connections: %w", err)
		}

		var connections []types.ProviderConnection
		if err := json.Unmarshal(resp.Data, &connections); err != nil {
			return fmt.Errorf("failed to parse connections: %w", err)
		}

		if len(connections) == 0 {
			output.Warning("No provider connections found.")
			output.Info("Use 'armyknife git connect <provider>' to connect a provider.")
			return nil
		}

		fmt.Println()
		for _, conn := range connections {
			display := providerDisplay[conn.Provider]
			status := "🔴 Inactive"
			if conn.IsActive {
				status = "🟢 Active"
			}

			fmt.Printf("%s %s - %s\n", display.icon, conn.DisplayName, status)
			fmt.Printf("   ID: %d | Type: %s | Created: %s\n",
				conn.ID, conn.ConnectionType, conn.CreatedAt)
			if conn.BaseURL != "" {
				fmt.Printf("   URL: %s\n", conn.BaseURL)
			}
			fmt.Println()
		}

		output.Info(fmt.Sprintf("Total: %d connection(s)", len(connections)))
		return nil
	},
}

var connectCmd = &cobra.Command{
	Use:   "connect <provider>",
	Short: "Connect a Git provider",
	Long: `Initiate OAuth flow to connect a Git provider.

Supported providers:
  - github      GitHub (cloud or Enterprise)
  - gitlab      GitLab (cloud or self-hosted)
  - bitbucket   Bitbucket Cloud
  - azure       Azure DevOps`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		providerArg := strings.ToLower(args[0])

		// Map short names to provider IDs
		providerMap := map[string]types.GitProvider{
			"github":    types.ProviderGitHub,
			"gh":        types.ProviderGitHub,
			"gitlab":    types.ProviderGitLab,
			"gl":        types.ProviderGitLab,
			"bitbucket": types.ProviderBitbucket,
			"bb":        types.ProviderBitbucket,
			"azure":     types.ProviderAzureDevOps,
			"ado":       types.ProviderAzureDevOps,
			"azdo":      types.ProviderAzureDevOps,
		}

		provider, ok := providerMap[providerArg]
		if !ok {
			return fmt.Errorf("unknown provider: %s. Supported: github, gitlab, bitbucket, azure", providerArg)
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

		display := providerDisplay[provider]
		output.Header(fmt.Sprintf("Connect %s %s", display.icon, provider))

		// Get connection type flag
		connType, _ := cmd.Flags().GetString("type")
		baseURL, _ := cmd.Flags().GetString("base-url")

		reqBody := types.ConnectProviderRequest{
			Provider:       provider,
			ConnectionType: connType,
			BaseURL:        baseURL,
		}

		resp, err := c.Post("/git/connect", reqBody)
		if err != nil {
			return fmt.Errorf("failed to initiate connection: %w", err)
		}

		var result struct {
			AuthURL string `json:"authUrl"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(resp.Data, &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		fmt.Println()
		output.Info("To complete the connection, visit this URL in your browser:")
		fmt.Println()
		fmt.Printf("  🔗 %s\n", result.AuthURL)
		fmt.Println()
		output.Info("After authorizing, you'll be redirected back to complete the setup.")

		return nil
	},
}

var disconnectCmd = &cobra.Command{
	Use:   "disconnect <provider>",
	Short: "Disconnect a Git provider",
	Long:  `Remove connection to a Git provider`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		providerArg := strings.ToLower(args[0])

		providerMap := map[string]types.GitProvider{
			"github":    types.ProviderGitHub,
			"gitlab":    types.ProviderGitLab,
			"bitbucket": types.ProviderBitbucket,
			"azure":     types.ProviderAzureDevOps,
		}

		provider, ok := providerMap[providerArg]
		if !ok {
			return fmt.Errorf("unknown provider: %s", providerArg)
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

		display := providerDisplay[provider]
		output.Header(fmt.Sprintf("Disconnect %s %s", display.icon, provider))

		_, err = c.Delete(fmt.Sprintf("/git/connections/%s", provider))
		if err != nil {
			return fmt.Errorf("failed to disconnect: %w", err)
		}

		output.Success(fmt.Sprintf("✅ Successfully disconnected from %s", provider))
		return nil
	},
}

// ============================================================
// UNIFIED REPOSITORY COMMANDS
// ============================================================

var gitReposCmd = &cobra.Command{
	Use:   "repos",
	Short: "List repositories across all providers",
	Long:  `List all repositories from all connected Git providers`,
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

		// Get filter flags
		providerFilter, _ := cmd.Flags().GetString("provider")
		limit, _ := cmd.Flags().GetInt("limit")

		output.Header("Repositories (All Providers)")
		output.Info("Fetching repositories...")

		path := "/git/repos"
		if providerFilter != "" {
			path += "?provider=" + providerFilter
		}
		if limit > 0 {
			if strings.Contains(path, "?") {
				path += fmt.Sprintf("&limit=%d", limit)
			} else {
				path += fmt.Sprintf("?limit=%d", limit)
			}
		}

		resp, err := c.Get(path)
		if err != nil {
			return fmt.Errorf("failed to fetch repositories: %w", err)
		}

		var result struct {
			Items      []types.UnifiedRepository   `json:"items"`
			TotalCount int                         `json:"totalCount"`
			ByProvider map[types.GitProvider]int   `json:"byProvider"`
		}
		if err := json.Unmarshal(resp.Data, &result); err != nil {
			return fmt.Errorf("failed to parse repositories: %w", err)
		}

		if jsonOut {
			return output.JSON(resp)
		}

		fmt.Println()
		for _, repo := range result.Items {
			display := providerDisplay[repo.Provider]
			visibility := "🔓"
			if repo.IsPrivate {
				visibility = "🔒"
			}
			fmt.Printf("%s %s %s\n", display.icon, visibility, repo.FullName)
			if repo.Description != "" {
				fmt.Printf("   📝 %s\n", truncate(repo.Description, 60))
			}
			fmt.Printf("   🌿 %s | ⭐ %d | 🍴 %d\n",
				repo.DefaultBranch, repo.StarCount, repo.ForkCount)
			fmt.Println()
		}

		// Summary by provider
		output.Info("Summary by Provider:")
		for provider, count := range result.ByProvider {
			display := providerDisplay[provider]
			fmt.Printf("  %s %s: %d repositories\n", display.icon, provider, count)
		}
		fmt.Printf("\nTotal: %d repositories\n", result.TotalCount)

		return nil
	},
}

// ============================================================
// UNIFIED PULL REQUEST COMMANDS
// ============================================================

var gitPRsCmd = &cobra.Command{
	Use:   "prs",
	Short: "List pull requests across all providers",
	Long:  `List pull requests/merge requests from all connected Git providers`,
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

		// Get filter flags
		state, _ := cmd.Flags().GetString("state")
		providerFilter, _ := cmd.Flags().GetString("provider")
		limit, _ := cmd.Flags().GetInt("limit")

		output.Header("Pull Requests (All Providers)")

		path := "/git/pull-requests"
		params := []string{}
		if state != "" {
			params = append(params, "state="+state)
		}
		if providerFilter != "" {
			params = append(params, "provider="+providerFilter)
		}
		if limit > 0 {
			params = append(params, fmt.Sprintf("limit=%d", limit))
		}
		if len(params) > 0 {
			path += "?" + strings.Join(params, "&")
		}

		resp, err := c.Get(path)
		if err != nil {
			return fmt.Errorf("failed to fetch pull requests: %w", err)
		}

		var result struct {
			Items      []types.UnifiedPullRequest `json:"items"`
			TotalCount int                        `json:"totalCount"`
			ByProvider map[types.GitProvider]int  `json:"byProvider"`
		}
		if err := json.Unmarshal(resp.Data, &result); err != nil {
			return fmt.Errorf("failed to parse pull requests: %w", err)
		}

		if jsonOut {
			return output.JSON(resp)
		}

		fmt.Println()
		for _, pr := range result.Items {
			display := providerDisplay[pr.Provider]
			stateIcon := "🟢"
			if pr.State == "merged" {
				stateIcon = "🟣"
			} else if pr.State == "closed" {
				stateIcon = "🔴"
			}

			draftIndicator := ""
			if pr.IsDraft {
				draftIndicator = " [DRAFT]"
			}

			fmt.Printf("%s %s #%d: %s%s\n", display.icon, stateIcon, pr.Number, pr.Title, draftIndicator)
			fmt.Printf("   📦 %s | 👤 %s\n", pr.RepoFullName, pr.Author)
			fmt.Printf("   🌿 %s → %s\n", pr.SourceBranch, pr.TargetBranch)
			if pr.Additions > 0 || pr.Deletions > 0 {
				fmt.Printf("   📊 +%d/-%d in %d files\n", pr.Additions, pr.Deletions, pr.ChangedFiles)
			}
			fmt.Println()
		}

		// Summary
		output.Info("Summary by Provider:")
		for provider, count := range result.ByProvider {
			display := providerDisplay[provider]
			fmt.Printf("  %s %s: %d PRs\n", display.icon, provider, count)
		}
		fmt.Printf("\nTotal: %d pull requests\n", result.TotalCount)

		return nil
	},
}

// ============================================================
// UNIFIED PIPELINE COMMANDS
// ============================================================

var gitPipelinesCmd = &cobra.Command{
	Use:   "pipelines",
	Short: "List CI/CD pipelines across all providers",
	Long:  `List CI/CD pipelines/workflows from all connected Git providers`,
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

		// Get filter flags
		status, _ := cmd.Flags().GetString("status")
		providerFilter, _ := cmd.Flags().GetString("provider")
		limit, _ := cmd.Flags().GetInt("limit")

		output.Header("CI/CD Pipelines (All Providers)")

		path := "/git/pipelines"
		params := []string{}
		if status != "" {
			params = append(params, "status="+status)
		}
		if providerFilter != "" {
			params = append(params, "provider="+providerFilter)
		}
		if limit > 0 {
			params = append(params, fmt.Sprintf("limit=%d", limit))
		}
		if len(params) > 0 {
			path += "?" + strings.Join(params, "&")
		}

		resp, err := c.Get(path)
		if err != nil {
			return fmt.Errorf("failed to fetch pipelines: %w", err)
		}

		var result struct {
			Items      []types.UnifiedPipeline   `json:"items"`
			TotalCount int                       `json:"totalCount"`
			ByProvider map[types.GitProvider]int `json:"byProvider"`
		}
		if err := json.Unmarshal(resp.Data, &result); err != nil {
			return fmt.Errorf("failed to parse pipelines: %w", err)
		}

		if jsonOut {
			return output.JSON(resp)
		}

		fmt.Println()
		for _, p := range result.Items {
			display := providerDisplay[p.Provider]
			statusIcon := "⏳"
			switch p.Status {
			case "success":
				statusIcon = "✅"
			case "failure":
				statusIcon = "❌"
			case "running":
				statusIcon = "🔄"
			case "cancelled":
				statusIcon = "⏹️"
			case "skipped":
				statusIcon = "⏭️"
			}

			name := p.Name
			if name == "" {
				name = p.Branch
			}

			fmt.Printf("%s %s %s\n", display.icon, statusIcon, name)
			fmt.Printf("   📦 %s | 🌿 %s\n", p.RepoFullName, p.Branch)
			fmt.Printf("   📝 %s | ⏱️ %ds\n", p.CommitSHA[:7], p.Duration)
			fmt.Println()
		}

		// Summary
		output.Info("Summary by Provider:")
		for provider, count := range result.ByProvider {
			display := providerDisplay[provider]
			fmt.Printf("  %s %s: %d pipelines\n", display.icon, provider, count)
		}
		fmt.Printf("\nTotal: %d pipelines\n", result.TotalCount)

		return nil
	},
}

// ============================================================
// PROVIDER SUMMARY COMMAND
// ============================================================

var gitSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Show summary across all providers",
	Long:  `Display an overview of all connected Git providers including repository counts,
open PRs, recent activity, and pipeline status.`,
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

		output.Header("Provider Summary")
		output.Info("Aggregating data from all providers...")

		resp, err := c.Get("/git/summary")
		if err != nil {
			return fmt.Errorf("failed to fetch summary: %w", err)
		}

		var summaries []types.ProviderSummary
		if err := json.Unmarshal(resp.Data, &summaries); err != nil {
			return fmt.Errorf("failed to parse summary: %w", err)
		}

		if jsonOut {
			return output.JSON(resp)
		}

		fmt.Println()

		totalRepos := 0
		totalPRs := 0
		totalCommits := 0

		for _, s := range summaries {
			display := providerDisplay[s.Provider]
			connectionStatus := "❌ Not Connected"
			if s.IsConnected {
				connectionStatus = "✅ Connected"
			}

			fmt.Printf("%s %s - %s\n", display.icon, strings.ToUpper(string(s.Provider)), connectionStatus)

			if s.IsConnected {
				if s.Error != "" {
					output.Error(fmt.Sprintf("   ⚠️ Error: %s", s.Error))
				} else {
					fmt.Printf("   📦 Repositories: %d\n", s.RepositoryCount)
					fmt.Printf("   🔀 Open PRs: %d\n", s.OpenPullRequests)
					fmt.Printf("   📝 Recent Commits (7d): %d\n", s.RecentCommits)
					fmt.Printf("   🔧 Pipelines: ✅%d ❌%d 🔄%d\n",
						s.PipelineStatus.Success,
						s.PipelineStatus.Failed,
						s.PipelineStatus.Running)

					totalRepos += s.RepositoryCount
					totalPRs += s.OpenPullRequests
					totalCommits += s.RecentCommits
				}
			}
			fmt.Println()
		}

		// Grand totals
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("📊 TOTALS\n")
		fmt.Printf("   📦 Total Repositories: %d\n", totalRepos)
		fmt.Printf("   🔀 Total Open PRs: %d\n", totalPRs)
		fmt.Printf("   📝 Total Recent Commits: %d\n", totalCommits)

		return nil
	},
}

// ============================================================
// INITIALIZATION
// ============================================================

func init() {
	rootCmd.AddCommand(gitCmd)

	// Provider management
	gitCmd.AddCommand(providersCmd)
	gitCmd.AddCommand(connectionsCmd)
	gitCmd.AddCommand(connectCmd)
	gitCmd.AddCommand(disconnectCmd)

	// Unified data commands
	gitCmd.AddCommand(gitReposCmd)
	gitCmd.AddCommand(gitPRsCmd)
	gitCmd.AddCommand(gitPipelinesCmd)
	gitCmd.AddCommand(gitSummaryCmd)

	// Connect command flags
	connectCmd.Flags().StringP("type", "t", "user", "Connection type: 'user' or 'organization'")
	connectCmd.Flags().StringP("base-url", "u", "", "Base URL for self-hosted providers")

	// Repos command flags
	gitReposCmd.Flags().StringP("provider", "p", "", "Filter by provider (github, gitlab, bitbucket, azure)")
	gitReposCmd.Flags().IntP("limit", "l", 50, "Maximum repositories to return")
	gitReposCmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output raw JSON")

	// PRs command flags
	gitPRsCmd.Flags().StringP("provider", "p", "", "Filter by provider")
	gitPRsCmd.Flags().StringP("state", "s", "open", "Filter by state: open, merged, closed, all")
	gitPRsCmd.Flags().IntP("limit", "l", 20, "Maximum PRs to return")
	gitPRsCmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output raw JSON")

	// Pipelines command flags
	gitPipelinesCmd.Flags().StringP("provider", "p", "", "Filter by provider")
	gitPipelinesCmd.Flags().StringP("status", "s", "", "Filter by status: success, failure, running, pending")
	gitPipelinesCmd.Flags().IntP("limit", "l", 20, "Maximum pipelines to return")
	gitPipelinesCmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output raw JSON")

	// Summary command flags
	gitSummaryCmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output raw JSON")
}

// Helper functions

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
