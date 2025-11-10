package cmd

import (
	"fmt"

	"github.com/armyknifelabs-platform/seip-cli/internal/client"
	"github.com/armyknifelabs-platform/seip-cli/internal/config"
	"github.com/armyknifelabs-platform/seip-cli/pkg/output"
	"github.com/spf13/cobra"
)

var (
	codeContent string
	language    string
	query       string
)

var aiCmd = &cobra.Command{
	Use:   "ai",
	Short: "AI-powered commands",
	Long:  `Interact with AI services including code analysis and RAG queries`,
}

var copilotCmd = &cobra.Command{
	Use:   "copilot",
	Short: "AI code assistance",
	Long:  `Get AI-powered code suggestions and assistance`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if codeContent == "" {
			return fmt.Errorf("--code flag is required")
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

		output.Header("AI Code Copilot")
		output.Info("Analyzing code...")

		reqBody := map[string]interface{}{
			"code":     codeContent,
			"language": language,
		}

		resp, err := c.Post("/ai/code/copilot", reqBody)
		if err != nil {
			return fmt.Errorf("failed to get code assistance: %w", err)
		}

		if jsonOut {
			return output.JSON(resp)
		}

		output.Success("\n✅ Analysis complete:")
		fmt.Println()
		return output.JSON(resp.Data)
	},
}

var ragCmd = &cobra.Command{
	Use:   "rag",
	Short: "RAG (Retrieval Augmented Generation) queries",
	Long:  `Query the RAG system for context-aware AI responses`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if query == "" {
			return fmt.Errorf("--query flag is required")
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

		output.Header("RAG Query")
		output.Info(fmt.Sprintf("Query: %s", query))
		output.Info("Processing...")

		reqBody := map[string]interface{}{
			"query": query,
		}

		resp, err := c.Post("/ai/rag/query", reqBody)
		if err != nil {
			return fmt.Errorf("failed to execute RAG query: %w", err)
		}

		if jsonOut {
			return output.JSON(resp)
		}

		output.Success("\n✅ Response:")
		fmt.Println()
		return output.JSON(resp.Data)
	},
}

var aiHealthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check AI service health",
	Long:  `Check the health status of AI services`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if apiURL != "" {
			cfg.APIURL = apiURL
		}

		c := client.NewClient(cfg)

		output.Header("AI Service Health")

		resp, err := c.Get("/ai/health")
		if err != nil {
			output.Error(fmt.Sprintf("❌ AI service health check failed: %v", err))
			return err
		}

		output.Success("✅ AI service is healthy")
		if jsonOut {
			return output.JSON(resp)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(aiCmd)
	aiCmd.AddCommand(copilotCmd)
	aiCmd.AddCommand(ragCmd)
	aiCmd.AddCommand(aiHealthCmd)

	copilotCmd.Flags().StringVarP(&codeContent, "code", "c", "", "Code to analyze (required)")
	copilotCmd.Flags().StringVarP(&language, "language", "l", "javascript", "Programming language")
	copilotCmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output raw JSON")

	ragCmd.Flags().StringVarP(&query, "query", "q", "", "Query string (required)")
	ragCmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output raw JSON")

	aiHealthCmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output raw JSON")
}
