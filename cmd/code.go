package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	repositoryID int
	queryLimit   int
)

// codeCmd represents the rag command
var codeCmd = &cobra.Command{
	Use:   "code",
	Short: "Code Intelligence and RAG operations",
	Long: `Retrieval-Augmented Generation (RAG) commands for semantic code search,
natural language queries, and AI-powered code analysis.

Examples:
  armyknife code index /path/to/repo --repo-id 1
  armyknife code query "How does authentication work?" --repo-id 1
  armyknife code stats --repo-id 1`,
}

// codeIndexCmd indexes a repository
var codeIndexCmd = &cobra.Command{
	Use:   "index <path>",
	Short: "Index a repository for code search",
	Long: `Index all code files in a repository for semantic search and AI analysis.
Supports: TypeScript, JavaScript, Go, Python, Rust, Java, C/C++, Ruby, PHP.

The path must be accessible from the backend server (mounted volume or network path).`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repositoryPath := args[0]

		// Convert to absolute path
		absPath, err := filepath.Abs(repositoryPath)
		if err != nil {
			fmt.Printf("Error: Invalid path: %v\n", err)
			os.Exit(1)
		}

		// Check if path exists
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			fmt.Printf("Error: Path does not exist: %s\n", absPath)
			os.Exit(1)
		}

		fmt.Printf("📂 Indexing repository: %s\n", absPath)
		fmt.Printf("🔢 Repository ID: %d\n", repositoryID)

		// Call API
		reqBody := map[string]interface{}{
			"repository_path": absPath,
			"repository_id":   repositoryID,
		}

		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			fmt.Printf("Error creating request: %v\n", err)
			os.Exit(1)
		}

		resp, err := http.Post(
			fmt.Sprintf("%s/code/index", apiURL),
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			fmt.Printf("Error calling API: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Error reading response: %v\n", err)
			os.Exit(1)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			fmt.Printf("Error parsing response: %v\n", err)
			fmt.Printf("Raw response: %s\n", string(body))
			os.Exit(1)
		}

		if success, ok := result["success"].(bool); ok && success {
			data := result["data"].(map[string]interface{})
			fmt.Printf("\n✅ Indexing Complete!\n")
			fmt.Printf("   Files Indexed: %.0f\n", data["files_indexed"])
			fmt.Printf("   Functions Extracted: %.0f\n", data["functions_extracted"])
			fmt.Printf("   Classes Extracted: %.0f\n", data["classes_extracted"])
			fmt.Printf("   Embeddings Created: %.0f\n", data["embeddings_created"])
			fmt.Printf("   Duration: %.0fms\n", data["duration_ms"])
		} else {
			fmt.Printf("❌ Indexing Failed\n")
			errorData := result["error"].(map[string]interface{})
			fmt.Printf("   Error: %s\n", errorData["message"])
			if details, ok := errorData["details"]; ok {
				fmt.Printf("   Details: %s\n", details)
			}
			os.Exit(1)
		}
	},
}

// codeQueryCmd queries code using natural language
var codeQueryCmd = &cobra.Command{
	Use:   "query <question>",
	Short: "Query code using natural language",
	Long: `Ask questions about your codebase in natural language.
The AI will search through indexed code and provide relevant snippets with explanations.

Examples:
  armyknife code query "How does authentication work?"
  armyknife code query "Where are API routes defined?" --repo-id 1
  armyknife code query "How do I handle errors?" --limit 3`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		question := args[0]

		fmt.Printf("🔍 Query: %s\n", question)
		if repositoryID > 0 {
			fmt.Printf("🔢 Repository ID: %d\n", repositoryID)
		}
		fmt.Printf("📊 Limit: %d results\n\n", queryLimit)

		// Call API
		reqBody := map[string]interface{}{
			"query": question,
			"limit": queryLimit,
		}
		if repositoryID > 0 {
			reqBody["repository_id"] = repositoryID
		}

		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			fmt.Printf("Error creating request: %v\n", err)
			os.Exit(1)
		}

		resp, err := http.Post(
			fmt.Sprintf("%s/code/query", apiURL),
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			fmt.Printf("Error calling API: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Error reading response: %v\n", err)
			os.Exit(1)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			fmt.Printf("Error parsing response: %v\n", err)
			fmt.Printf("Raw response: %s\n", string(body))
			os.Exit(1)
		}

		if success, ok := result["success"].(bool); ok && success {
			data := result["data"].(map[string]interface{})
			results := data["results"].([]interface{})

			if len(results) == 0 {
				fmt.Printf("❌ No results found\n")
				fmt.Printf("   Try indexing your repository first: armyknife code index <path>\n")
				return
			}

			fmt.Printf("✅ Found %d results:\n\n", len(results))

			for i, r := range results {
				res := r.(map[string]interface{})
				fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
				fmt.Printf("Result #%d (Score: %.2f)\n", i+1, res["score"])
				fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
				fmt.Printf("📁 File: %s\n", res["filePath"])
				if functionName, ok := res["functionName"].(string); ok && functionName != "" {
					fmt.Printf("🔧 Function: %s\n", functionName)
				}
				if className, ok := res["className"].(string); ok && className != "" {
					fmt.Printf("📦 Class: %s\n", className)
				}
				fmt.Printf("\n💡 Explanation:\n%s\n\n", res["snippet"])
			}
		} else {
			fmt.Printf("❌ Query Failed\n")
			errorData := result["error"].(map[string]interface{})
			fmt.Printf("   Error: %s\n", errorData["message"])
			if details, ok := errorData["details"]; ok {
				fmt.Printf("   Details: %s\n", details)
			}
			os.Exit(1)
		}
	},
}

// codeHybridCmd uses hybrid search (vector + keyword)
var codeHybridCmd = &cobra.Command{
	Use:   "hybrid <question>",
	Short: "Query code using hybrid search (vector + keyword)",
	Long: `Ask questions about your codebase using hybrid search combining:
  - Vector search (semantic similarity via embeddings)
  - Keyword search (PostgreSQL full-text search)

Hybrid search provides better accuracy, especially for exact keyword matches.
Scoring: 0.7 * vector_similarity + 0.3 * keyword_relevance

Examples:
  armyknife code hybrid "authentication login function"
  armyknife code hybrid "getUserById method" --repo-id 1`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		question := args[0]

		fmt.Printf("🔀 Hybrid Query: %s\n", question)
		if repositoryID > 0 {
			fmt.Printf("🔢 Repository ID: %d\n", repositoryID)
		}
		fmt.Printf("📊 Limit: %d results\n\n", queryLimit)

		// Call API
		reqBody := map[string]interface{}{
			"query": question,
			"limit": queryLimit,
		}
		if repositoryID > 0 {
			reqBody["repository_id"] = repositoryID
		}

		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			fmt.Printf("Error creating request: %v\n", err)
			os.Exit(1)
		}

		resp, err := http.Post(
			fmt.Sprintf("%s/code/query/hybrid", apiURL),
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			fmt.Printf("Error calling API: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Error reading response: %v\n", err)
			os.Exit(1)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			fmt.Printf("Error parsing response: %v\n", err)
			fmt.Printf("Raw response: %s\n", string(body))
			os.Exit(1)
		}

		if success, ok := result["success"].(bool); ok && success {
			data := result["data"].(map[string]interface{})
			results := data["results"].([]interface{})

			if len(results) == 0 {
				fmt.Printf("❌ No results found\n")
				return
			}

			searchType := data["search_type"].(string)
			fmt.Printf("✅ Found %d results (%s search):\n\n", len(results), searchType)

			for i, r := range results {
				res := r.(map[string]interface{})
				fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
				fmt.Printf("Result #%d (Score: %.2f)\n", i+1, res["score"])
				fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
				fmt.Printf("📁 File: %s\n", res["filePath"])
				if functionName, ok := res["functionName"].(string); ok && functionName != "" {
					fmt.Printf("🔧 Function: %s\n", functionName)
				}
				if className, ok := res["className"].(string); ok && className != "" {
					fmt.Printf("📦 Class: %s\n", className)
				}
				if lineStart, ok := res["lineStart"].(float64); ok && lineStart > 0 {
					fmt.Printf("📍 Lines: %.0f", lineStart)
					if lineEnd, ok := res["lineEnd"].(float64); ok && lineEnd > 0 {
						fmt.Printf("-%.0f\n", lineEnd)
					} else {
						fmt.Printf("\n")
					}
				}
				fmt.Printf("\n💡 Snippet:\n%s\n\n", res["snippet"])
			}
		} else {
			fmt.Printf("❌ Hybrid Query Failed\n")
			errorData := result["error"].(map[string]interface{})
			fmt.Printf("   Error: %s\n", errorData["message"])
			if details, ok := errorData["details"]; ok {
				fmt.Printf("   Details: %s\n", details)
			}
			os.Exit(1)
		}
	},
}

// codeMetricsCmd gets performance metrics
var codeMetricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Get code intelligence performance metrics",
	Long: `Display performance metrics for the code intelligence system:
  - Cache hit rates and efficiency
  - Query latency percentiles (p50, p95, p99)
  - Index statistics (repositories, embeddings, files)

Useful for monitoring system performance and optimization.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("📊 Fetching performance metrics...\n\n")

		resp, err := http.Get(fmt.Sprintf("%s/code/metrics", apiURL))
		if err != nil {
			fmt.Printf("Error calling API: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Error reading response: %v\n", err)
			os.Exit(1)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			fmt.Printf("Error parsing response: %v\n", err)
			os.Exit(1)
		}

		if success, ok := result["success"].(bool); ok && success {
			data := result["data"].(map[string]interface{})

			// Cache metrics
			cache := data["cache"].(map[string]interface{})
			fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
			fmt.Printf("💾 Cache Performance\n")
			fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
			fmt.Printf("   Hits: %.0f\n", cache["hits"])
			fmt.Printf("   Misses: %.0f\n", cache["misses"])
			fmt.Printf("   Hit Rate: %.2f%%\n", cache["hitRate"])
			fmt.Printf("   Total Queries: %.0f\n\n", cache["totalQueries"])

			// Query latency
			latency := data["queryLatency"].(map[string]interface{})
			fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
			fmt.Printf("⚡ Query Latency (milliseconds)\n")
			fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
			fmt.Printf("   p50 (median): %.0fms\n", latency["p50"])
			fmt.Printf("   p95: %.0fms\n", latency["p95"])
			fmt.Printf("   p99: %.0fms\n\n", latency["p99"])

			// Index statistics
			stats := data["indexStats"].(map[string]interface{})
			fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
			fmt.Printf("📚 Index Statistics\n")
			fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
			fmt.Printf("   Repositories: %.0f\n", stats["totalRepositories"])
			fmt.Printf("   Total Embeddings: %.0f\n", stats["totalEmbeddings"])
			fmt.Printf("   Total Files: %.0f\n", stats["totalFiles"])
			fmt.Printf("   Avg Embeddings/File: %.2f\n", stats["avgEmbeddingsPerFile"])

			fmt.Printf("\n✅ System is healthy and operational\n")
		} else {
			fmt.Printf("❌ Failed to get metrics\n")
			errorData := result["error"].(map[string]interface{})
			fmt.Printf("   Error: %s\n", errorData["message"])
			os.Exit(1)
		}
	},
}

// codeStatsCmd gets indexing statistics
var codeStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Get code indexing statistics",
	Long:  `Display statistics about indexed code including total embeddings, repositories, and files.`,
	Run: func(cmd *cobra.Command, args []string) {
		url := fmt.Sprintf("%s/code/stats", apiURL)
		if repositoryID > 0 {
			url = fmt.Sprintf("%s?repository_id=%d", url, repositoryID)
		}

		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf("Error calling API: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Error reading response: %v\n", err)
			os.Exit(1)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			fmt.Printf("Error parsing response: %v\n", err)
			os.Exit(1)
		}

		if success, ok := result["success"].(bool); ok && success {
			data := result["data"].(map[string]interface{})
			fmt.Printf("\n📊 Code Indexing Statistics\n")
			fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
			fmt.Printf("   Total Embeddings: %s\n", data["total_embeddings"])
			fmt.Printf("   Total Repositories: %s\n", data["total_repositories"])
			fmt.Printf("   Total Files: %s\n", data["total_files"])
			fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")
		} else {
			fmt.Printf("❌ Failed to get stats\n")
			errorData := result["error"].(map[string]interface{})
			fmt.Printf("   Error: %s\n", errorData["message"])
			os.Exit(1)
		}
	},
}

// ============================================================
// Repository Management Commands (Phase 2)
// ============================================================

// codeRepoCmd represents the repository management parent command
var codeRepoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Repository management commands",
	Long: `Manage code repositories for code intelligence indexing.

Examples:
  armyknife code repo register armyknifelabs-platform armyknifelabs-idp-seip-platform
  armyknife code repo list
  armyknife code repo get 1
  armyknife code repo delete 2`,
}

// codeRepoRegisterCmd registers a new repository
var codeRepoRegisterCmd = &cobra.Command{
	Use:   "register <owner> <repo>",
	Short: "Register a new repository for code intelligence",
	Long: `Register a new repository in the code intelligence system.
This creates a repository record that can be indexed for semantic code search.

The repository will be marked as 'pending' until it is indexed.`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		owner := args[0]
		repo := args[1]
		githubURL, _ := cmd.Flags().GetString("github-url")

		fmt.Printf("📝 Registering repository: %s/%s\n", owner, repo)

		// Call API
		reqBody := map[string]interface{}{
			"owner": owner,
			"repo":  repo,
		}
		if githubURL != "" {
			reqBody["github_url"] = githubURL
		}

		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			fmt.Printf("Error creating request: %v\n", err)
			os.Exit(1)
		}

		resp, err := http.Post(
			fmt.Sprintf("%s/code/repositories", apiURL),
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			fmt.Printf("Error calling API: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Error reading response: %v\n", err)
			os.Exit(1)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			fmt.Printf("Error parsing response: %v\n", err)
			fmt.Printf("Raw response: %s\n", string(body))
			os.Exit(1)
		}

		if success, ok := result["success"].(bool); ok && success {
			data := result["data"].(map[string]interface{})
			fmt.Printf("\n✅ Repository Registered!\n")
			fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
			fmt.Printf("   ID: %.0f\n", data["id"])
			fmt.Printf("   Owner: %s\n", data["owner"])
			fmt.Printf("   Repo: %s\n", data["repo"])
			fmt.Printf("   Status: %s\n", data["status"])
			if githubURLVal, ok := data["githubUrl"].(string); ok && githubURLVal != "" {
				fmt.Printf("   GitHub URL: %s\n", githubURLVal)
			}
			fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")
			fmt.Printf("Next: Index the repository with:\n")
			fmt.Printf("  armyknife code index /path/to/repo --repo-id %.0f\n\n", data["id"])
		} else {
			fmt.Printf("❌ Registration Failed\n")
			errorData := result["error"].(map[string]interface{})
			fmt.Printf("   Error: %s\n", errorData["message"])
			if details, ok := errorData["details"]; ok {
				fmt.Printf("   Details: %s\n", details)
			}
			os.Exit(1)
		}
	},
}

// codeRepoListCmd lists all repositories
var codeRepoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered repositories",
	Long: `List all repositories registered in the code intelligence system.

You can filter by status: pending, indexing, indexed, or failed.`,
	Run: func(cmd *cobra.Command, args []string) {
		status, _ := cmd.Flags().GetString("status")

		url := fmt.Sprintf("%s/code/repositories", apiURL)
		if status != "" {
			url = fmt.Sprintf("%s?status=%s", url, status)
		}

		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf("Error calling API: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Error reading response: %v\n", err)
			os.Exit(1)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			fmt.Printf("Error parsing response: %v\n", err)
			os.Exit(1)
		}

		if success, ok := result["success"].(bool); ok && success {
			data := result["data"].([]interface{})

			if len(data) == 0 {
				fmt.Printf("❌ No repositories found\n")
				if status != "" {
					fmt.Printf("   (Filtered by status: %s)\n", status)
				}
				fmt.Printf("\n   Register a repository with:\n")
				fmt.Printf("   armyknife code repo register <owner> <repo>\n\n")
				return
			}

			fmt.Printf("\n📚 Repositories (%d total)\n", len(data))
			fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

			for _, r := range data {
				repo := r.(map[string]interface{})
				fmt.Printf("\n   ID: %.0f\n", repo["id"])
				fmt.Printf("   Repository: %s/%s\n", repo["owner"], repo["repo"])
				fmt.Printf("   Status: %s\n", repo["status"])
				fmt.Printf("   Files: %.0f\n", repo["fileCount"])
				fmt.Printf("   Embeddings: %.0f\n", repo["embeddingCount"])
				if lastIndexed, ok := repo["lastIndexedAt"].(string); ok && lastIndexed != "" {
					fmt.Printf("   Last Indexed: %s\n", lastIndexed)
				}
			}

			fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")
		} else {
			fmt.Printf("❌ Failed to list repositories\n")
			errorData := result["error"].(map[string]interface{})
			fmt.Printf("   Error: %s\n", errorData["message"])
			os.Exit(1)
		}
	},
}

// codeRepoGetCmd gets repository details
var codeRepoGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get repository details and statistics",
	Long:  `Get detailed information about a specific repository including indexing statistics.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repoID := args[0]

		resp, err := http.Get(fmt.Sprintf("%s/code/repositories/%s", apiURL, repoID))
		if err != nil {
			fmt.Printf("Error calling API: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Error reading response: %v\n", err)
			os.Exit(1)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			fmt.Printf("Error parsing response: %v\n", err)
			os.Exit(1)
		}

		if success, ok := result["success"].(bool); ok && success {
			data := result["data"].(map[string]interface{})
			stats := data["stats"].(map[string]interface{})

			fmt.Printf("\n📦 Repository Details\n")
			fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
			fmt.Printf("   ID: %.0f\n", data["id"])
			fmt.Printf("   Repository: %s/%s\n", data["owner"], data["repo"])
			fmt.Printf("   Status: %s\n", data["status"])

			if githubURL, ok := data["githubUrl"].(string); ok && githubURL != "" {
				fmt.Printf("   GitHub URL: %s\n", githubURL)
			}

			if lastIndexed, ok := data["lastIndexedAt"].(string); ok && lastIndexed != "" {
				fmt.Printf("   Last Indexed: %s\n", lastIndexed)
			}

			fmt.Printf("\n📊 Statistics:\n")
			fmt.Printf("   Files: %.0f\n", stats["fileCount"])
			fmt.Printf("   Embeddings: %.0f\n", stats["embeddingCount"])
			fmt.Printf("   Functions: %.0f\n", stats["functionCount"])
			fmt.Printf("   Classes: %.0f\n", stats["classCount"])

			if languages, ok := stats["languages"].([]interface{}); ok && len(languages) > 0 {
				fmt.Printf("   Languages: ")
				for i, lang := range languages {
					if i > 0 {
						fmt.Printf(", ")
					}
					fmt.Printf("%s", lang)
				}
				fmt.Printf("\n")
			}

			if errorMsg, ok := data["errorMessage"].(string); ok && errorMsg != "" {
				fmt.Printf("\n❌ Error: %s\n", errorMsg)
			}

			fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")
		} else {
			fmt.Printf("❌ Failed to get repository\n")
			errorData := result["error"].(map[string]interface{})
			fmt.Printf("   Error: %s\n", errorData["message"])
			os.Exit(1)
		}
	},
}

// codeRepoDeleteCmd deletes a repository
var codeRepoDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a repository and all its embeddings",
	Long: `Delete a repository from the code intelligence system.

⚠️  WARNING: This will delete all embeddings for this repository.
This action cannot be undone.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repoID := args[0]
		confirm, _ := cmd.Flags().GetBool("confirm")

		if !confirm {
			fmt.Printf("⚠️  WARNING: This will delete repository %s and ALL its embeddings.\n", repoID)
			fmt.Printf("   This action cannot be undone.\n\n")
			fmt.Printf("   To confirm deletion, add the --confirm flag:\n")
			fmt.Printf("   armyknife code repo delete %s --confirm\n\n", repoID)
			os.Exit(1)
		}

		fmt.Printf("🗑️  Deleting repository %s...\n", repoID)

		client := &http.Client{}
		req, err := http.NewRequest(
			"DELETE",
			fmt.Sprintf("%s/code/repositories/%s", apiURL, repoID),
			nil,
		)
		if err != nil {
			fmt.Printf("Error creating request: %v\n", err)
			os.Exit(1)
		}

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Error calling API: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Error reading response: %v\n", err)
			os.Exit(1)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			fmt.Printf("Error parsing response: %v\n", err)
			os.Exit(1)
		}

		if success, ok := result["success"].(bool); ok && success {
			data := result["data"].(map[string]interface{})
			fmt.Printf("\n✅ %s\n\n", data["message"])
		} else {
			fmt.Printf("❌ Failed to delete repository\n")
			errorData := result["error"].(map[string]interface{})
			fmt.Printf("   Error: %s\n", errorData["message"])
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(codeCmd)

	// Add subcommands
	codeCmd.AddCommand(codeIndexCmd)
	codeCmd.AddCommand(codeQueryCmd)
	codeCmd.AddCommand(codeHybridCmd)
	codeCmd.AddCommand(codeMetricsCmd)
	codeCmd.AddCommand(codeStatsCmd)
	codeCmd.AddCommand(codeRepoCmd)

	// Add repository management subcommands
	codeRepoCmd.AddCommand(codeRepoRegisterCmd)
	codeRepoCmd.AddCommand(codeRepoListCmd)
	codeRepoCmd.AddCommand(codeRepoGetCmd)
	codeRepoCmd.AddCommand(codeRepoDeleteCmd)

	// Flags for index command
	codeIndexCmd.Flags().IntVar(&repositoryID, "repo-id", 1, "Repository ID")

	// Flags for query command
	codeQueryCmd.Flags().IntVar(&repositoryID, "repo-id", 0, "Repository ID (optional, searches all if not specified)")
	codeQueryCmd.Flags().IntVar(&queryLimit, "limit", 5, "Maximum number of results")

	// Flags for hybrid command
	codeHybridCmd.Flags().IntVar(&repositoryID, "repo-id", 0, "Repository ID (optional, searches all if not specified)")
	codeHybridCmd.Flags().IntVar(&queryLimit, "limit", 5, "Maximum number of results")

	// Flags for stats command
	codeStatsCmd.Flags().IntVar(&repositoryID, "repo-id", 0, "Repository ID (optional, shows all if not specified)")

	// Flags for repository register command
	codeRepoRegisterCmd.Flags().String("github-url", "", "GitHub URL for the repository (optional)")

	// Flags for repository list command
	codeRepoListCmd.Flags().String("status", "", "Filter by status: pending, indexing, indexed, or failed")

	// Flags for repository delete command
	codeRepoDeleteCmd.Flags().Bool("confirm", false, "Confirm deletion (required)")
}
