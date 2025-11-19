package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/armyknifelabs-platform/armyknife-cli/internal/client"
	"github.com/armyknifelabs-platform/armyknife-cli/internal/config"
	"github.com/armyknifelabs-platform/armyknife-cli/pkg/output"
	"github.com/spf13/cobra"
)

var (
	ragQuery  string
	ragLimit  int
	useAI     bool
	ragType   string
)

// ragCmd represents the main rag command
var ragRootCmd = &cobra.Command{
	Use:   "rag",
	Short: "RAG (Retrieval Augmented Generation) system",
	Long: `Query and manage the RAG (Retrieval Augmented Generation) system.

The platform has three RAG systems:
- docs: Documentation RAG (internal markdown docs from shared-docs-ready/)
- pdf: PDF RAG (uploaded PDF books and documents)
- code: Code RAG (repository code and documentation)`,
}

// ragDocsCmd queries the documentation RAG
var ragDocsCmd = &cobra.Command{
	Use:   "docs [query]",
	Short: "Query documentation RAG (internal docs)",
	Long:  `Search internal documentation from shared-docs-ready/ using semantic search`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if apiURL != "" {
			cfg.APIURL = apiURL
		}

		c := client.NewClient(cfg)

		output.Header("Documentation RAG Query")
		if useAI {
			output.Info("AI-Enhanced Mode: ON")
		}
		output.Info(fmt.Sprintf("Query: %s", query))
		output.Info("Searching internal documentation...")

		reqBody := map[string]interface{}{
			"query": query,
			"limit": ragLimit,
			"useAI": useAI,
		}

		resp, err := c.Post("/ai/docs/query", reqBody)
		if err != nil {
			return fmt.Errorf("failed to query documentation RAG: %w", err)
		}

		if jsonOut {
			return output.JSON(resp)
		}

		output.Success("\n✅ Documentation Search Results:")
		fmt.Println()

		// Unmarshal response data
		var data map[string]interface{}
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		// Extract and display results
		if results, ok := data["results"].([]interface{}); ok {
			for i, result := range results {
				if r, ok := result.(map[string]interface{}); ok {
					fmt.Printf("%d. %s\n", i+1, r["title"])
					fmt.Printf("   Type: %s | Score: %.2f | Relevance: %s\n",
						r["type"], r["score"], r["relevance"])
					if text, ok := r["text"].(string); ok && len(text) > 200 {
						fmt.Printf("   %s...\n", text[:200])
					} else if ok {
						fmt.Printf("   %s\n", text)
					}
					fmt.Println()
				}
			}
		}

		// Display AI response if available
		if useAI {
			if aiResponse, ok := data["aiResponse"].(string); ok && aiResponse != "" {
				fmt.Println("\n🤖 AI-Enhanced Response:")
				fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
				fmt.Println(aiResponse)
				fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			}
		}

		return nil
	},
}

// ragPdfCmd queries the PDF RAG
var ragPdfCmd = &cobra.Command{
	Use:   "pdf [query]",
	Short: "Query PDF RAG (uploaded PDFs)",
	Long:  `Search uploaded PDF books and documents using semantic search`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if apiURL != "" {
			cfg.APIURL = apiURL
		}

		c := client.NewClient(cfg)

		output.Header("PDF RAG Query")
		output.Info(fmt.Sprintf("Query: %s", query))
		output.Info("Searching PDF documents...")

		reqBody := map[string]interface{}{
			"query": query,
			"limit": ragLimit,
		}

		resp, err := c.Post("/ai/rag/query", reqBody)
		if err != nil {
			return fmt.Errorf("failed to query PDF RAG: %w", err)
		}

		if jsonOut {
			return output.JSON(resp)
		}

		output.Success("\n✅ PDF Search Results:")
		fmt.Println()

		// Unmarshal response data
		var data map[string]interface{}
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		// Extract and display results
		if results, ok := data["results"].([]interface{}); ok {
			for i, result := range results {
				if r, ok := result.(map[string]interface{}); ok {
					fmt.Printf("%d. %s\n", i+1, r["filename"])
					fmt.Printf("   Score: %.2f | Relevance: %s\n", r["score"], r["relevance"])
					if text, ok := r["text"].(string); ok && len(text) > 200 {
						fmt.Printf("   %s...\n", text[:200])
					} else if ok {
						fmt.Printf("   %s\n", text)
					}
					fmt.Println()
				}
			}
		}

		return nil
	},
}

// ragCodeCmd queries the code RAG
var ragCodeCmd = &cobra.Command{
	Use:   "code [query]",
	Short: "Query code RAG (repository code)",
	Long:  `Search repository code and documentation using semantic search`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]

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

		output.Header("Code RAG Query")
		output.Info(fmt.Sprintf("Query: %s", query))
		output.Info("Searching code repositories...")

		reqBody := map[string]interface{}{
			"query": query,
			"limit": ragLimit,
		}

		resp, err := c.Post("/rag/query", reqBody)
		if err != nil {
			return fmt.Errorf("failed to query code RAG: %w", err)
		}

		if jsonOut {
			return output.JSON(resp)
		}

		output.Success("\n✅ Code Search Results:")
		fmt.Println()
		return output.JSON(resp.Data)
	},
}

// ragListCmd lists documents in RAG systems
var ragListCmd = &cobra.Command{
	Use:   "list [type]",
	Short: "List documents in RAG systems",
	Long:  `List all ingested documents. Type can be: docs, pdf, code, or all (default)`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ragType := "all"
		if len(args) > 0 {
			ragType = args[0]
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if apiURL != "" {
			cfg.APIURL = apiURL
		}

		c := client.NewClient(cfg)

		endpoints := map[string]string{
			"docs": "/ai/docs/list",
			"pdf":  "/ai/rag/documents",
			"code": "/rag/status",
		}

		typesToQuery := []string{}
		if ragType == "all" {
			for k := range endpoints {
				typesToQuery = append(typesToQuery, k)
			}
		} else {
			if _, ok := endpoints[ragType]; !ok {
				return fmt.Errorf("invalid type: %s. Use: docs, pdf, code, or all", ragType)
			}
			typesToQuery = []string{ragType}
		}

		output.Header(fmt.Sprintf("RAG Documents (%s)", ragType))

		for _, t := range typesToQuery {
			fmt.Printf("\n=== %s RAG ===\n", t)
			resp, err := c.Get(endpoints[t])
			if err != nil {
				output.Error(fmt.Sprintf("❌ Failed to list %s documents: %v", t, err))
				continue
			}

			if jsonOut {
				output.JSON(resp)
			} else {
				output.JSON(resp.Data)
			}
		}

		return nil
	},
}

// ragStatusCmd checks RAG system status
var ragStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check RAG systems status",
	Long:  `Check the health and status of all RAG systems`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if apiURL != "" {
			cfg.APIURL = apiURL
		}

		c := client.NewClient(cfg)

		output.Header("RAG Systems Status")

		endpoints := []struct {
			name string
			path string
		}{
			{"Documentation RAG", "/ai/docs/status"},
			{"PDF RAG", "/ai/rag/status"},
			{"Code RAG", "/rag/status"},
		}

		for _, ep := range endpoints {
			fmt.Printf("\n=== %s ===\n", ep.name)
			resp, err := c.Get(ep.path)
			if err != nil {
				output.Error(fmt.Sprintf("❌ %s unavailable: %v", ep.name, err))
				continue
			}

			if jsonOut {
				output.JSON(resp)
			} else {
				output.JSON(resp.Data)
			}
		}

		return nil
	},
}

// ragSyncCmd triggers code embedding sync
var ragSyncCmd = &cobra.Command{
	Use:   "sync [owner] [repo]",
	Short: "Sync repository code for embeddings",
	Long:  `Trigger embedding sync to ingest repository code into the RAG system`,
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		owner := args[0]
		repo := args[1]

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

		output.Header(fmt.Sprintf("Code RAG Sync: %s/%s", owner, repo))
		output.Info("Triggering embedding sync...")
		output.Info("⚠️  This may take 10-30 minutes depending on repository size")

		// Get force flag
		force, _ := cmd.Flags().GetBool("force")

		reqBody := map[string]interface{}{
			"owner": owner,
			"repo":  repo,
			"force": force,
		}

		resp, err := c.Post("/rag/sync", reqBody)
		if err != nil {
			return fmt.Errorf("failed to trigger sync: %w", err)
		}

		if jsonOut {
			return output.JSON(resp)
		}

		// Unmarshal response data
		var data map[string]interface{}
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		output.Success("\n✅ Sync Job Queued:")
		fmt.Printf("  Job ID: %s\n", data["jobId"])
		fmt.Printf("  Owner: %s\n", data["owner"])
		fmt.Printf("  Repo: %s\n", data["repo"])
		fmt.Printf("  Status: %s\n", data["status"])
		fmt.Println()
		fmt.Println("Monitor progress with: armyknife rag status")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(ragRootCmd)

	ragRootCmd.AddCommand(ragDocsCmd)
	ragRootCmd.AddCommand(ragPdfCmd)
	ragRootCmd.AddCommand(ragCodeCmd)
	ragRootCmd.AddCommand(ragListCmd)
	ragRootCmd.AddCommand(ragStatusCmd)
	ragRootCmd.AddCommand(ragSyncCmd)

	// Flags for docs command
	ragDocsCmd.Flags().IntVarP(&ragLimit, "limit", "l", 5, "Maximum number of results")
	ragDocsCmd.Flags().BoolVar(&useAI, "ai", false, "Use AI-enhanced responses (Claude 3.5 Sonnet)")
	ragDocsCmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output raw JSON")

	// Flags for pdf command
	ragPdfCmd.Flags().IntVarP(&ragLimit, "limit", "l", 5, "Maximum number of results")
	ragPdfCmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output raw JSON")

	// Flags for code command
	ragCodeCmd.Flags().IntVarP(&ragLimit, "limit", "l", 5, "Maximum number of results")
	ragCodeCmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output raw JSON")

	// Flags for list command
	ragListCmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output raw JSON")

	// Flags for status command
	ragStatusCmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output raw JSON")

	// Flags for sync command
	ragSyncCmd.Flags().Bool("force", false, "Force re-sync even if already synced")
	ragSyncCmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output raw JSON")
}
