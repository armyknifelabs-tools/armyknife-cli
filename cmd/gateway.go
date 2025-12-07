package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	searchMode           string
	searchLimit          int
	searchLanguage       string
	searchNodeType       string
	embeddingProvider    string
	vectorWeight         float64
	bm25Weight           float64
	enableReranking      bool
	similarityThreshold  float64
)

// gatewayCmd represents the gateway command
var gatewayCmd = &cobra.Command{
	Use:   "gateway",
	Short: "LLM Gateway operations (RAG, Search, Embeddings)",
	Long: `LLM Gateway commands for AI-powered code intelligence and search.

Includes:
- Hybrid Search (vector + BM25 with Reciprocal Rank Fusion)
- RAG operations (search, explain, similar, index)
- Dual embedding pipeline (local + cloud)

Examples:
  armyknife gateway search "authentication middleware" --mode hybrid
  armyknife gateway rag search "How does error handling work?"
  armyknife gateway rag explain "func main() {}"
  armyknife gateway embedding "code snippet" --provider openai
  armyknife gateway status`,
}

// gatewayStatusCmd gets gateway status
var gatewayStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get LLM Gateway status",
	Long:  `Get the status of the LLM Gateway including search, RAG, and embedding services.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🔌 LLM Gateway Status")
		fmt.Println(strings.Repeat("-", 50))

		// Get search status
		searchResp, err := http.Get(fmt.Sprintf("%s/gateway/search/status", apiURL))
		if err != nil {
			fmt.Printf("❌ Search Service: Error - %v\n", err)
		} else {
			defer searchResp.Body.Close()
			body, _ := io.ReadAll(searchResp.Body)
			var result map[string]interface{}
			if err := json.Unmarshal(body, &result); err == nil && result["success"] == true {
				data := result["data"].(map[string]interface{})
				fmt.Printf("✅ Search Service: %v\n", data["status"])
				if providers, ok := data["providers"].(map[string]interface{}); ok {
					fmt.Printf("   Embedding Providers:\n")
					for name, info := range providers {
						if provInfo, ok := info.(map[string]interface{}); ok {
							status := "❌"
							if provInfo["available"] == true {
								status = "✅"
							}
							fmt.Printf("   - %s: %s\n", name, status)
						}
					}
				}
			} else {
				fmt.Printf("⚠️  Search Service: Unable to parse status\n")
			}
		}

		// Get RAG status
		ragResp, err := http.Get(fmt.Sprintf("%s/gateway/rag/status", apiURL))
		if err != nil {
			fmt.Printf("❌ RAG Service: Error - %v\n", err)
		} else {
			defer ragResp.Body.Close()
			body, _ := io.ReadAll(ragResp.Body)
			var result map[string]interface{}
			if err := json.Unmarshal(body, &result); err == nil && result["success"] == true {
				data := result["data"].(map[string]interface{})
				fmt.Printf("✅ RAG Service: %v\n", data["status"])
				if languages, ok := data["supportedLanguages"].([]interface{}); ok {
					fmt.Printf("   Supported Languages: %d\n", len(languages))
				}
			} else {
				fmt.Printf("⚠️  RAG Service: Unable to parse status\n")
			}
		}

		fmt.Println()
	},
}

// hybridSearchCmd performs hybrid search
var hybridSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Hybrid search combining vector and BM25",
	Long: `Perform hybrid search combining vector similarity (semantic) and BM25 (keyword)
with Reciprocal Rank Fusion (RRF) for optimal results.

Search modes:
- hybrid: Combined vector + BM25 (default, best results)
- vector: Semantic search only (good for concept search)
- bm25: Keyword search only (good for exact matches)

Examples:
  armyknife gateway search "authentication flow"
  armyknife gateway search "handleAuth function" --mode bm25
  armyknife gateway search "error handling patterns" --mode vector
  armyknife gateway search "rate limiting" --limit 20 --rerank`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := args[0]

		fmt.Printf("🔍 Searching: %s\n", query)
		fmt.Printf("   Mode: %s | Limit: %d\n", searchMode, searchLimit)
		if enableReranking {
			fmt.Printf("   Reranking: enabled\n")
		}
		fmt.Println()

		reqBody := map[string]interface{}{
			"query":              query,
			"mode":               searchMode,
			"limit":              searchLimit,
			"vectorWeight":       vectorWeight,
			"bm25Weight":         bm25Weight,
			"enableReranking":    enableReranking,
			"similarityThreshold": similarityThreshold,
			"embeddingProvider":  embeddingProvider,
		}

		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			fmt.Printf("Error creating request: %v\n", err)
			os.Exit(1)
		}

		resp, err := http.Post(
			fmt.Sprintf("%s/gateway/search", apiURL),
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			fmt.Printf("Error calling API: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			fmt.Printf("Error parsing response: %v\n", err)
			os.Exit(1)
		}

		if success, ok := result["success"].(bool); ok && success {
			data := result["data"].(map[string]interface{})
			results := data["results"].([]interface{})

			fmt.Printf("📊 Found %d results\n\n", len(results))

			for i, r := range results {
				res := r.(map[string]interface{})
				title := res["title"]
				if title == nil {
					title = res["filePath"]
				}
				fmt.Printf("%d. %s\n", i+1, title)

				if score, ok := res["score"].(float64); ok {
					fmt.Printf("   RRF Score: %.4f", score)
				}
				if vectorScore, ok := res["vectorScore"].(float64); ok {
					fmt.Printf(" | Vector: %.4f", vectorScore)
				}
				if bm25Score, ok := res["bm25Score"].(float64); ok {
					fmt.Printf(" | BM25: %.4f", bm25Score)
				}
				fmt.Println()

				if filePath, ok := res["filePath"].(string); ok && filePath != "" {
					fmt.Printf("   File: %s\n", filePath)
				}
				if nodeType, ok := res["nodeType"].(string); ok && nodeType != "" {
					fmt.Printf("   Type: %s\n", nodeType)
				}
				if content, ok := res["content"].(string); ok && len(content) > 0 {
					preview := content
					if len(preview) > 200 {
						preview = preview[:200] + "..."
					}
					fmt.Printf("   Preview: %s\n", strings.ReplaceAll(preview, "\n", " "))
				}
				fmt.Println()
			}
		} else {
			if errData, ok := result["error"].(map[string]interface{}); ok {
				fmt.Printf("❌ Error: %v\n", errData["message"])
			} else {
				fmt.Printf("❌ Search failed\n")
			}
		}
	},
}

// codeSearchCmd performs code-specific search
var codeSearchCmd = &cobra.Command{
	Use:   "code-search <query>",
	Short: "Code-specific search with AST filters",
	Long: `Search code using hybrid search with optional AST-based filters.

Filter by:
- Language: typescript, python, go, rust, java
- Node Type: function, class, interface, method, struct

Examples:
  armyknife gateway code-search "error handling"
  armyknife gateway code-search "middleware" --language typescript
  armyknife gateway code-search "Service class" --node-type class`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := args[0]

		fmt.Printf("🔍 Code Search: %s\n", query)
		if searchLanguage != "" {
			fmt.Printf("   Language: %s\n", searchLanguage)
		}
		if searchNodeType != "" {
			fmt.Printf("   Node Type: %s\n", searchNodeType)
		}
		fmt.Println()

		reqBody := map[string]interface{}{
			"query":          query,
			"organizationId": 1, // Default org
			"limit":          searchLimit,
			"mode":           searchMode,
		}

		if searchLanguage != "" {
			reqBody["language"] = []string{searchLanguage}
		}
		if searchNodeType != "" {
			reqBody["nodeType"] = []string{searchNodeType}
		}

		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			fmt.Printf("Error creating request: %v\n", err)
			os.Exit(1)
		}

		resp, err := http.Post(
			fmt.Sprintf("%s/gateway/search/code", apiURL),
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			fmt.Printf("Error calling API: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			fmt.Printf("Error parsing response: %v\n", err)
			os.Exit(1)
		}

		if success, ok := result["success"].(bool); ok && success {
			data := result["data"].(map[string]interface{})
			results := data["results"].([]interface{})

			fmt.Printf("📊 Found %d code chunks\n\n", len(results))

			for i, r := range results {
				res := r.(map[string]interface{})
				fmt.Printf("%d. %s", i+1, res["nodeName"])
				if nodeType, ok := res["nodeType"].(string); ok {
					fmt.Printf(" (%s)", nodeType)
				}
				fmt.Println()

				if filePath, ok := res["filePath"].(string); ok {
					fmt.Printf("   File: %s", filePath)
					if startLine, ok := res["startLine"].(float64); ok {
						fmt.Printf(":%d", int(startLine))
					}
					fmt.Println()
				}
				if signature, ok := res["signature"].(string); ok && signature != "" {
					fmt.Printf("   Signature: %s\n", signature)
				}
				if score, ok := res["score"].(float64); ok {
					fmt.Printf("   Score: %.4f\n", score)
				}
				fmt.Println()
			}
		} else {
			if errData, ok := result["error"].(map[string]interface{}); ok {
				fmt.Printf("❌ Error: %v\n", errData["message"])
			} else {
				fmt.Printf("❌ Code search failed\n")
			}
		}
	},
}

// ragCmd represents the rag subcommand group
var gatewayRagCmd = &cobra.Command{
	Use:   "rag",
	Short: "RAG (Retrieval-Augmented Generation) operations",
	Long: `RAG commands for AI-powered code intelligence.

Operations:
- search: Semantic code search
- explain: AI code explanation
- similar: Find similar code
- index: Index repository for RAG

Examples:
  armyknife gateway rag search "How does auth work?"
  armyknife gateway rag explain "func handler(w http.ResponseWriter)"
  armyknife gateway rag similar "defer db.Close()"`,
}

// ragSearchCmd performs RAG search
var ragSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Semantic RAG search",
	Long: `Search codebase using RAG with semantic understanding.

Supports natural language queries like:
- "How does the authentication system work?"
- "Where are errors handled?"
- "What does the rate limiter do?"`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := args[0]

		fmt.Printf("🧠 RAG Search: %s\n\n", query)

		reqBody := map[string]interface{}{
			"query": query,
			"options": map[string]interface{}{
				"limit":      searchLimit,
				"searchMode": searchMode,
			},
		}

		jsonData, _ := json.Marshal(reqBody)

		resp, err := http.Post(
			fmt.Sprintf("%s/gateway/rag/search", apiURL),
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(body, &result)

		if result["success"] == true {
			data := result["data"].(map[string]interface{})
			results := data["results"].([]interface{})

			fmt.Printf("📊 Found %d relevant code chunks\n\n", len(results))

			for i, r := range results {
				res := r.(map[string]interface{})
				fmt.Printf("%d. %s\n", i+1, res["nodeName"])
				if filePath, ok := res["filePath"].(string); ok {
					fmt.Printf("   %s\n", filePath)
				}
				if score, ok := res["score"].(float64); ok {
					fmt.Printf("   Relevance: %.2f%%\n", score*100)
				}
				fmt.Println()
			}
		} else {
			fmt.Printf("❌ RAG search failed\n")
		}
	},
}

// ragExplainCmd explains code
var ragExplainCmd = &cobra.Command{
	Use:   "explain <code>",
	Short: "Get AI explanation of code",
	Long: `Get an AI-powered explanation of code including:
- Purpose and functionality
- Complexity analysis
- Potential improvements
- Related patterns`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		code := args[0]

		fmt.Printf("🤖 Explaining code...\n\n")

		reqBody := map[string]interface{}{
			"code": code,
		}

		if searchLanguage != "" {
			reqBody["context"] = map[string]string{
				"language": searchLanguage,
			}
		}

		jsonData, _ := json.Marshal(reqBody)

		resp, err := http.Post(
			fmt.Sprintf("%s/gateway/rag/explain", apiURL),
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(body, &result)

		if result["success"] == true {
			data := result["data"].(map[string]interface{})

			fmt.Printf("📝 Code Explanation\n")
			fmt.Println(strings.Repeat("-", 50))

			if explanation, ok := data["explanation"].(string); ok {
				fmt.Println(explanation)
			}

			if complexity, ok := data["complexity"].(map[string]interface{}); ok {
				fmt.Printf("\n📊 Complexity\n")
				if level, ok := complexity["level"].(string); ok {
					fmt.Printf("   Level: %s\n", level)
				}
				if factors, ok := complexity["factors"].([]interface{}); ok {
					fmt.Printf("   Factors: %v\n", factors)
				}
			}

			if suggestions, ok := data["suggestions"].([]interface{}); ok && len(suggestions) > 0 {
				fmt.Printf("\n💡 Suggestions\n")
				for _, s := range suggestions {
					fmt.Printf("   • %s\n", s)
				}
			}
		} else {
			fmt.Printf("❌ Code explanation failed\n")
		}
	},
}

// ragSimilarCmd finds similar code
var ragSimilarCmd = &cobra.Command{
	Use:   "similar <code>",
	Short: "Find similar code patterns",
	Long:  `Find semantically similar code patterns in the indexed codebase.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		code := args[0]

		fmt.Printf("🔎 Finding similar code...\n\n")

		reqBody := map[string]interface{}{
			"code":  code,
			"limit": searchLimit,
		}

		jsonData, _ := json.Marshal(reqBody)

		resp, err := http.Post(
			fmt.Sprintf("%s/gateway/rag/similar", apiURL),
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(body, &result)

		if result["success"] == true {
			data := result["data"].(map[string]interface{})
			results := data["results"].([]interface{})

			fmt.Printf("📊 Found %d similar patterns\n\n", len(results))

			for i, r := range results {
				res := r.(map[string]interface{})
				fmt.Printf("%d. %s\n", i+1, res["nodeName"])
				if filePath, ok := res["filePath"].(string); ok {
					fmt.Printf("   File: %s\n", filePath)
				}
				if similarity, ok := res["similarity"].(float64); ok {
					fmt.Printf("   Similarity: %.2f%%\n", similarity*100)
				}
				fmt.Println()
			}
		} else {
			fmt.Printf("❌ Similar search failed\n")
		}
	},
}

// ragIndexCmd indexes a repository
var ragIndexCmd = &cobra.Command{
	Use:   "index <repo-id>",
	Short: "Index a repository for RAG",
	Long: `Index a repository's codebase for RAG operations.

This will:
1. Parse code using Tree-sitter AST
2. Chunk code into semantic units
3. Generate embeddings using dual pipeline
4. Store in vector database for search`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repoId := args[0]

		fmt.Printf("📥 Indexing repository: %s\n\n", repoId)

		reqBody := map[string]interface{}{
			"repoId": repoId,
		}

		jsonData, _ := json.Marshal(reqBody)

		resp, err := http.Post(
			fmt.Sprintf("%s/gateway/rag/index", apiURL),
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(body, &result)

		if result["success"] == true {
			data := result["data"].(map[string]interface{})
			fmt.Printf("✅ Indexing started\n")
			if jobId, ok := data["jobId"].(string); ok {
				fmt.Printf("   Job ID: %s\n", jobId)
			}
			if status, ok := data["status"].(string); ok {
				fmt.Printf("   Status: %s\n", status)
			}
		} else {
			fmt.Printf("❌ Indexing failed\n")
		}
	},
}

// embeddingCmd generates embeddings
var embeddingCmd = &cobra.Command{
	Use:   "embedding <text>",
	Short: "Generate embeddings for text/code",
	Long: `Generate vector embeddings for text or code using the dual embedding pipeline.

Providers:
- auto: Automatically select best provider (default)
- local: Use local model (UniXcoder)
- openai: Use OpenAI text-embedding-3-small
- voyage: Use Voyage AI
- ollama: Use local Ollama instance`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		text := args[0]

		fmt.Printf("🧮 Generating embedding...\n")
		fmt.Printf("   Provider: %s\n\n", embeddingProvider)

		reqBody := map[string]interface{}{
			"text":     text,
			"provider": embeddingProvider,
		}

		jsonData, _ := json.Marshal(reqBody)

		resp, err := http.Post(
			fmt.Sprintf("%s/gateway/rag/embedding", apiURL),
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(body, &result)

		if result["success"] == true {
			data := result["data"].(map[string]interface{})
			fmt.Printf("✅ Embedding generated\n")
			if dims, ok := data["dimensions"].(float64); ok {
				fmt.Printf("   Dimensions: %d\n", int(dims))
			}
			if model, ok := data["model"].(string); ok {
				fmt.Printf("   Model: %s\n", model)
			}
			if embedding, ok := data["embedding"].([]interface{}); ok {
				fmt.Printf("   Preview: [%.4f, %.4f, %.4f, ...]\n",
					embedding[0], embedding[1], embedding[2])
			}
		} else {
			fmt.Printf("❌ Embedding generation failed\n")
		}
	},
}

// ingestCmd represents the ingest subcommand group
var ingestCmd = &cobra.Command{
	Use:   "ingest",
	Short: "Ingest repositories for RAG indexing",
	Long: `Ingest repository code and documentation into the RAG pipeline.

Workflow: ingest → index → analyze → search

Operations:
- repo: Ingest a single repository
- org: Ingest all repos in an organization
- status: Check ingestion job status
- history: View ingestion history

Examples:
  armyknife gateway ingest repo --owner myorg --repo myrepo
  armyknife gateway ingest org --owner myorg --schedule-daily
  armyknife gateway ingest status job-123`,
}

var (
	ingestOwner         string
	ingestRepo          string
	ingestIncludeCode   bool
	ingestIncludeDocs   bool
	ingestIncludeTests  bool
	ingestScheduleDaily bool
	ingestMaxFileSizeKB int
)

// ingestRepoCmd ingests a single repository
var ingestRepoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Ingest a single repository",
	Long: `Ingest a single repository's code and documentation for RAG.

By default, only documentation files (*.md, README, etc.) are ingested.
Use flags to include source code and test files.

Examples:
  armyknife gateway ingest repo --owner armyknifelabs --repo backend
  armyknife gateway ingest repo --owner myorg --repo myrepo --include-code
  armyknife gateway ingest repo --owner myorg --repo myrepo --include-code --include-tests`,
	Run: func(cmd *cobra.Command, args []string) {
		if ingestOwner == "" || ingestRepo == "" {
			fmt.Println("❌ Error: --owner and --repo are required")
			os.Exit(1)
		}

		fmt.Printf("📥 Ingesting repository: %s/%s\n", ingestOwner, ingestRepo)
		fmt.Printf("   Include Code: %v | Include Docs: %v | Include Tests: %v\n\n",
			ingestIncludeCode, ingestIncludeDocs, ingestIncludeTests)

		reqBody := map[string]interface{}{
			"owner":         ingestOwner,
			"repo":          ingestRepo,
			"includeCode":   ingestIncludeCode,
			"includeDocs":   ingestIncludeDocs,
			"includeTests":  ingestIncludeTests,
			"maxFileSizeKB": ingestMaxFileSizeKB,
		}

		jsonData, _ := json.Marshal(reqBody)

		resp, err := http.Post(
			fmt.Sprintf("%s/rag/ingest/repo", apiURL),
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(body, &result)

		if result["success"] == true {
			data := result["data"].(map[string]interface{})
			fmt.Printf("✅ Ingestion queued!\n")
			if jobId, ok := data["jobId"].(string); ok {
				fmt.Printf("   Job ID: %s\n", jobId)
			}
			if status, ok := data["status"].(string); ok {
				fmt.Printf("   Status: %s\n", status)
			}
			if msg, ok := data["message"].(string); ok {
				fmt.Printf("   %s\n", msg)
			}
			if checkUrl, ok := data["checkStatusUrl"].(string); ok {
				fmt.Printf("\n   Check status: armyknife gateway ingest status <jobId>\n")
				fmt.Printf("   API: %s%s\n", apiURL, checkUrl)
			}
		} else {
			if errData, ok := result["error"].(map[string]interface{}); ok {
				fmt.Printf("❌ Error: %v\n", errData["message"])
			} else {
				fmt.Printf("❌ Ingestion failed\n")
			}
		}
	},
}

// ingestOrgCmd ingests an entire organization
var ingestOrgCmd = &cobra.Command{
	Use:   "org",
	Short: "Ingest all repositories in an organization",
	Long: `Ingest all repositories in an organization for RAG.

Can optionally schedule daily re-ingestion at 2 AM.

Examples:
  armyknife gateway ingest org --owner armyknifelabs
  armyknife gateway ingest org --owner myorg --schedule-daily
  armyknife gateway ingest org --owner myorg --include-code --include-docs`,
	Run: func(cmd *cobra.Command, args []string) {
		if ingestOwner == "" {
			fmt.Println("❌ Error: --owner is required")
			os.Exit(1)
		}

		fmt.Printf("📥 Ingesting organization: %s\n", ingestOwner)
		fmt.Printf("   Include Code: %v | Include Docs: %v | Include Tests: %v\n",
			ingestIncludeCode, ingestIncludeDocs, ingestIncludeTests)
		if ingestScheduleDaily {
			fmt.Printf("   ⏰ Daily ingestion scheduled at 2 AM\n")
		}
		fmt.Println()

		reqBody := map[string]interface{}{
			"owner":         ingestOwner,
			"includeCode":   ingestIncludeCode,
			"includeDocs":   ingestIncludeDocs,
			"includeTests":  ingestIncludeTests,
			"maxFileSizeKB": ingestMaxFileSizeKB,
			"scheduleDaily": ingestScheduleDaily,
		}

		jsonData, _ := json.Marshal(reqBody)

		resp, err := http.Post(
			fmt.Sprintf("%s/rag/ingest/org", apiURL),
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(body, &result)

		if result["success"] == true {
			data := result["data"].(map[string]interface{})
			fmt.Printf("✅ Organization ingestion queued!\n")
			if jobId, ok := data["jobId"].(string); ok {
				fmt.Printf("   Job ID: %s\n", jobId)
			}
			if repos, ok := data["reposToProcess"].(float64); ok {
				fmt.Printf("   Repos to process: %d\n", int(repos))
			}
			if msg, ok := data["message"].(string); ok {
				fmt.Printf("   %s\n", msg)
			}
			if est, ok := data["estimatedTime"].(string); ok {
				fmt.Printf("   Estimated time: %s\n", est)
			}
		} else {
			if errData, ok := result["error"].(map[string]interface{}); ok {
				fmt.Printf("❌ Error: %v\n", errData["message"])
			} else {
				fmt.Printf("❌ Organization ingestion failed\n")
			}
		}
	},
}

// ingestStatusCmd checks ingestion job status
var ingestStatusCmd = &cobra.Command{
	Use:   "status <jobId>",
	Short: "Check ingestion job status",
	Long:  `Check the status of an ingestion job by its job ID.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jobId := args[0]

		fmt.Printf("🔍 Checking status for job: %s\n\n", jobId)

		resp, err := http.Get(fmt.Sprintf("%s/rag/ingest/status/%s", apiURL, jobId))
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(body, &result)

		if result["success"] == true {
			data := result["data"].(map[string]interface{})

			status := data["status"].(string)
			statusIcon := "⏳"
			switch status {
			case "completed":
				statusIcon = "✅"
			case "failed":
				statusIcon = "❌"
			case "cancelled":
				statusIcon = "⚪"
			case "processing":
				statusIcon = "🔄"
			}

			fmt.Printf("%s Status: %s\n", statusIcon, status)
			if owner, ok := data["owner"].(string); ok {
				fmt.Printf("   Owner: %s\n", owner)
			}
			if repo, ok := data["repo"].(string); ok {
				fmt.Printf("   Repo: %s\n", repo)
			}
			if files, ok := data["filesIngested"].(float64); ok {
				fmt.Printf("   Files ingested: %d\n", int(files))
			}
			if skipped, ok := data["filesSkipped"].(float64); ok && skipped > 0 {
				fmt.Printf("   Files skipped: %d\n", int(skipped))
			}
			if errors, ok := data["errors"].(float64); ok && errors > 0 {
				fmt.Printf("   Errors: %d\n", int(errors))
			}
			if duration, ok := data["duration"].(float64); ok && duration > 0 {
				fmt.Printf("   Duration: %ds\n", int(duration))
			}
			if msg, ok := data["message"].(string); ok {
				fmt.Printf("\n   %s\n", msg)
			}
		} else {
			if errData, ok := result["error"].(map[string]interface{}); ok {
				fmt.Printf("❌ Error: %v\n", errData["message"])
			} else {
				fmt.Printf("❌ Failed to get job status\n")
			}
		}
	},
}

// ingestHistoryCmd shows ingestion history
var ingestHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "View ingestion history",
	Long: `View history of ingestion jobs.

Examples:
  armyknife gateway ingest history
  armyknife gateway ingest history --owner myorg
  armyknife gateway ingest history --owner myorg --repo myrepo`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("📜 Ingestion History\n")
		fmt.Println(strings.Repeat("-", 60))

		url := fmt.Sprintf("%s/rag/ingest/history?limit=%d", apiURL, searchLimit)
		if ingestOwner != "" {
			url += "&owner=" + ingestOwner
		}
		if ingestRepo != "" {
			url += "&repo=" + ingestRepo
		}

		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(body, &result)

		if result["success"] == true {
			data := result["data"].(map[string]interface{})
			jobs := data["jobs"].([]interface{})

			if len(jobs) == 0 {
				fmt.Println("No ingestion history found.")
				return
			}

			for _, j := range jobs {
				job := j.(map[string]interface{})
				status := job["status"].(string)
				statusIcon := "⏳"
				switch status {
				case "completed":
					statusIcon = "✅"
				case "failed":
					statusIcon = "❌"
				case "cancelled":
					statusIcon = "⚪"
				}

				fmt.Printf("%s %s/%s\n", statusIcon, job["owner"], job["repo"])
				if jobId, ok := job["jobId"].(string); ok {
					fmt.Printf("   Job ID: %s\n", jobId)
				}
				if files, ok := job["filesIngested"].(float64); ok {
					fmt.Printf("   Files: %d ingested", int(files))
					if skipped, ok := job["filesSkipped"].(float64); ok && skipped > 0 {
						fmt.Printf(", %d skipped", int(skipped))
					}
					fmt.Println()
				}
				fmt.Println()
			}

			if pagination, ok := data["pagination"].(map[string]interface{}); ok {
				if total, ok := pagination["total"].(float64); ok {
					fmt.Printf("Total: %d jobs\n", int(total))
				}
			}
		} else {
			fmt.Printf("❌ Failed to get ingestion history\n")
		}
	},
}

// analyzeCmd represents the analyze subcommand group
var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "AI-powered code analysis",
	Long: `AI-powered repository analysis using Claude/GPT.

Analysis types:
- codebaseExplain: Overall codebase explanation
- patterns: Coding patterns detection
- issues: Issues summarization
- wiki: Wiki/Discussions discovery
- copilot: Comprehensive Copilot analysis

Workflow: ingest → index → analyze → search

Examples:
  armyknife gateway analyze run --owner myorg --repo myrepo --type codebaseExplain
  armyknife gateway analyze status job-123
  armyknife gateway analyze results --owner myorg --repo myrepo`,
}

var (
	analyzeType    string
	analyzeForce   bool
)

// analyzeRunCmd runs AI analysis
var analyzeRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run AI analysis on a repository",
	Long: `Queue AI-powered analysis on a repository.

Analysis types:
- codebaseExplain: Overall codebase explanation and architecture
- patterns: Detect coding patterns and best practices
- issues: Summarize open issues and priorities
- wiki: Discover and analyze wiki/docs
- copilot: Comprehensive GitHub Copilot-style analysis

Analysis runs asynchronously - use 'status' to check progress.

Examples:
  armyknife gateway analyze run --owner myorg --repo myrepo --type codebaseExplain
  armyknife gateway analyze run --owner myorg --repo myrepo --type patterns
  armyknife gateway analyze run --owner myorg --repo myrepo --type copilot --force`,
	Run: func(cmd *cobra.Command, args []string) {
		if ingestOwner == "" || ingestRepo == "" {
			fmt.Println("❌ Error: --owner and --repo are required")
			os.Exit(1)
		}

		fmt.Printf("🤖 Queuing AI analysis: %s\n", analyzeType)
		fmt.Printf("   Repository: %s/%s\n", ingestOwner, ingestRepo)
		if analyzeForce {
			fmt.Printf("   Force refresh: yes\n")
		}
		fmt.Println()

		reqBody := map[string]interface{}{
			"owner":        ingestOwner,
			"repo":         ingestRepo,
			"analysisType": analyzeType,
			"forceRefresh": analyzeForce,
		}

		jsonData, _ := json.Marshal(reqBody)

		resp, err := http.Post(
			fmt.Sprintf("%s/github/ai-analyze", apiURL),
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(body, &result)

		if result["success"] == true {
			data := result["data"].(map[string]interface{})
			status := data["status"].(string)

			if status == "cached" {
				fmt.Printf("✅ Analysis cached (returning existing result)\n")
				if analysis, ok := data["analysis"].(string); ok {
					fmt.Println(strings.Repeat("-", 60))
					fmt.Println(analysis)
				}
				if stale, ok := data["stale"].(bool); ok && stale {
					fmt.Printf("\n⚠️  Result is stale - background refresh queued\n")
				}
			} else {
				fmt.Printf("✅ Analysis queued!\n")
				if jobId, ok := data["jobId"].(string); ok {
					fmt.Printf("   Job ID: %s\n", jobId)
					fmt.Printf("\n   Check status: armyknife gateway analyze status %s\n", jobId)
				}
				if msg, ok := data["message"].(string); ok {
					fmt.Printf("   %s\n", msg)
				}
			}
		} else {
			if errData, ok := result["error"].(map[string]interface{}); ok {
				fmt.Printf("❌ Error: %v\n", errData["message"])
			} else {
				fmt.Printf("❌ Analysis failed\n")
			}
		}
	},
}

// analyzeStatusCmd checks analysis job status
var analyzeStatusCmd = &cobra.Command{
	Use:   "status <jobId>",
	Short: "Check AI analysis job status",
	Long:  `Check the status of an AI analysis job by its job ID.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jobId := args[0]

		fmt.Printf("🔍 Checking analysis status: %s\n\n", jobId)

		resp, err := http.Get(fmt.Sprintf("%s/github/ai-analyze/status/%s", apiURL, jobId))
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(body, &result)

		if result["success"] == true {
			data := result["data"].(map[string]interface{})

			status := data["status"].(string)
			statusIcon := "⏳"
			switch status {
			case "completed":
				statusIcon = "✅"
			case "failed":
				statusIcon = "❌"
			case "processing":
				statusIcon = "🔄"
			}

			fmt.Printf("%s Status: %s\n", statusIcon, status)
			if progress, ok := data["progress"].(float64); ok {
				fmt.Printf("   Progress: %.0f%%\n", progress)
			}

			if status == "completed" {
				if analysis, ok := data["analysis"].(string); ok {
					fmt.Println(strings.Repeat("-", 60))
					fmt.Println(analysis)
				}
			}

			if status == "failed" {
				if errMsg, ok := data["error"].(string); ok {
					fmt.Printf("   Error: %s\n", errMsg)
				}
			}
		} else {
			if errData, ok := result["error"].(map[string]interface{}); ok {
				fmt.Printf("❌ Error: %v\n", errData["message"])
			} else {
				fmt.Printf("❌ Failed to get analysis status\n")
			}
		}
	},
}

// analyzeResultsCmd gets all analysis results for a repo
var analyzeResultsCmd = &cobra.Command{
	Use:   "results",
	Short: "Get all AI analysis results for a repository",
	Long: `Get all cached AI analysis results for a repository.

Examples:
  armyknife gateway analyze results --owner myorg --repo myrepo`,
	Run: func(cmd *cobra.Command, args []string) {
		if ingestOwner == "" || ingestRepo == "" {
			fmt.Println("❌ Error: --owner and --repo are required")
			os.Exit(1)
		}

		fmt.Printf("📊 AI Analysis Results: %s/%s\n", ingestOwner, ingestRepo)
		fmt.Println(strings.Repeat("-", 60))

		resp, err := http.Get(fmt.Sprintf("%s/github/ai-analyze/%s/%s", apiURL, ingestOwner, ingestRepo))
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(body, &result)

		if result["success"] == true {
			data := result["data"].(map[string]interface{})

			if analyses, ok := data["analyses"].(map[string]interface{}); ok {
				if len(analyses) == 0 {
					fmt.Println("No analysis results found. Run 'armyknife gateway analyze run' first.")
					return
				}

				for analysisType, analysisData := range analyses {
					fmt.Printf("\n📝 %s\n", analysisType)
					if ad, ok := analysisData.(map[string]interface{}); ok {
						if analysis, ok := ad["analysis"].(string); ok {
							// Truncate long analyses
							preview := analysis
							if len(preview) > 500 {
								preview = preview[:500] + "..."
							}
							fmt.Println(preview)
						}
						if timestamp, ok := ad["generatedAt"].(string); ok {
							fmt.Printf("\n   Generated: %s\n", timestamp)
						}
					}
					fmt.Println()
				}
			}
		} else {
			if errData, ok := result["error"].(map[string]interface{}); ok {
				fmt.Printf("❌ Error: %v\n", errData["message"])
			} else {
				fmt.Printf("❌ Failed to get analysis results\n")
			}
		}
	},
}

// analyzeStatsCmd gets AI analysis statistics
var analyzeStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Get AI analysis job queue statistics",
	Long:  `Get statistics about the AI analysis job queue.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("📊 AI Analysis Statistics\n")
		fmt.Println(strings.Repeat("-", 40))

		resp, err := http.Get(fmt.Sprintf("%s/github/ai-analyze/stats", apiURL))
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(body, &result)

		if result["success"] == true {
			data := result["data"].(map[string]interface{})
			if stats, ok := data["stats"].(map[string]interface{}); ok {
				if waiting, ok := stats["waiting"].(float64); ok {
					fmt.Printf("   Waiting: %d\n", int(waiting))
				}
				if active, ok := stats["active"].(float64); ok {
					fmt.Printf("   Active: %d\n", int(active))
				}
				if completed, ok := stats["completed"].(float64); ok {
					fmt.Printf("   Completed: %d\n", int(completed))
				}
				if failed, ok := stats["failed"].(float64); ok {
					fmt.Printf("   Failed: %d\n", int(failed))
				}
			}
		} else {
			fmt.Printf("❌ Failed to get statistics\n")
		}
	},
}

// explainRankingCmd explains search ranking
var explainRankingCmd = &cobra.Command{
	Use:   "explain-ranking <query>",
	Short: "Debug search ranking algorithm",
	Long: `Get detailed explanation of how results are ranked for a query.

Shows:
- Vector-only results and scores
- BM25-only results and scores
- Hybrid RRF fusion results
- Score breakdown`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := args[0]

		fmt.Printf("🔬 Analyzing ranking for: %s\n\n", query)

		reqBody := map[string]interface{}{
			"query": query,
			"limit": 5,
		}

		jsonData, _ := json.Marshal(reqBody)

		resp, err := http.Post(
			fmt.Sprintf("%s/gateway/search/explain-ranking", apiURL),
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(body, &result)

		if result["success"] == true {
			data := result["data"].(map[string]interface{})
			explanation := data["explanation"].(map[string]interface{})

			// Vector results
			vectorData := explanation["vectorOnly"].(map[string]interface{})
			fmt.Printf("🔵 Vector Search (Semantic)\n")
			fmt.Printf("   Total: %v results\n", vectorData["count"])
			if topResults, ok := vectorData["topResults"].([]interface{}); ok {
				for _, r := range topResults {
					res := r.(map[string]interface{})
					fmt.Printf("   - %s (score: %.4f)\n", res["title"], res["score"])
				}
			}
			fmt.Println()

			// BM25 results
			bm25Data := explanation["bm25Only"].(map[string]interface{})
			fmt.Printf("🟢 BM25 Search (Keyword)\n")
			fmt.Printf("   Total: %v results\n", bm25Data["count"])
			if topResults, ok := bm25Data["topResults"].([]interface{}); ok {
				for _, r := range topResults {
					res := r.(map[string]interface{})
					fmt.Printf("   - %s (score: %.4f)\n", res["title"], res["score"])
				}
			}
			fmt.Println()

			// Hybrid results
			hybridData := explanation["hybrid"].(map[string]interface{})
			fmt.Printf("🟣 Hybrid Search (RRF Fusion)\n")
			fmt.Printf("   Total: %v results\n", hybridData["count"])
			fmt.Printf("   RRF k: %v\n", hybridData["rrfFusionK"])
			if topResults, ok := hybridData["topResults"].([]interface{}); ok {
				for _, r := range topResults {
					res := r.(map[string]interface{})
					fmt.Printf("   - %s\n", res["title"])
					fmt.Printf("     RRF: %.4f | Vector: %.4f | BM25: %.4f\n",
						res["rrfScore"], res["vectorScore"], res["bm25Score"])
				}
			}
		} else {
			fmt.Printf("❌ Ranking explanation failed\n")
		}
	},
}

func init() {
	rootCmd.AddCommand(gatewayCmd)

	// Gateway subcommands
	gatewayCmd.AddCommand(gatewayStatusCmd)
	gatewayCmd.AddCommand(hybridSearchCmd)
	gatewayCmd.AddCommand(codeSearchCmd)
	gatewayCmd.AddCommand(gatewayRagCmd)
	gatewayCmd.AddCommand(embeddingCmd)
	gatewayCmd.AddCommand(explainRankingCmd)
	gatewayCmd.AddCommand(ingestCmd)
	gatewayCmd.AddCommand(analyzeCmd)

	// RAG subcommands
	gatewayRagCmd.AddCommand(ragSearchCmd)
	gatewayRagCmd.AddCommand(ragExplainCmd)
	gatewayRagCmd.AddCommand(ragSimilarCmd)
	gatewayRagCmd.AddCommand(ragIndexCmd)

	// Ingest subcommands
	ingestCmd.AddCommand(ingestRepoCmd)
	ingestCmd.AddCommand(ingestOrgCmd)
	ingestCmd.AddCommand(ingestStatusCmd)
	ingestCmd.AddCommand(ingestHistoryCmd)

	// Analyze subcommands
	analyzeCmd.AddCommand(analyzeRunCmd)
	analyzeCmd.AddCommand(analyzeStatusCmd)
	analyzeCmd.AddCommand(analyzeResultsCmd)
	analyzeCmd.AddCommand(analyzeStatsCmd)

	// Hybrid search flags
	hybridSearchCmd.Flags().StringVar(&searchMode, "mode", "hybrid", "Search mode: hybrid, vector, bm25")
	hybridSearchCmd.Flags().IntVar(&searchLimit, "limit", 10, "Maximum results to return")
	hybridSearchCmd.Flags().Float64Var(&vectorWeight, "vector-weight", 0.5, "Weight for vector search (0-1)")
	hybridSearchCmd.Flags().Float64Var(&bm25Weight, "bm25-weight", 0.5, "Weight for BM25 search (0-1)")
	hybridSearchCmd.Flags().BoolVar(&enableReranking, "rerank", false, "Enable result reranking")
	hybridSearchCmd.Flags().Float64Var(&similarityThreshold, "threshold", 0.3, "Minimum similarity threshold")
	hybridSearchCmd.Flags().StringVar(&embeddingProvider, "provider", "auto", "Embedding provider: auto, local, openai, voyage, ollama")

	// Code search flags
	codeSearchCmd.Flags().StringVar(&searchMode, "mode", "hybrid", "Search mode: hybrid, vector, bm25")
	codeSearchCmd.Flags().IntVar(&searchLimit, "limit", 10, "Maximum results to return")
	codeSearchCmd.Flags().StringVar(&searchLanguage, "language", "", "Filter by language (typescript, python, go, etc.)")
	codeSearchCmd.Flags().StringVar(&searchNodeType, "node-type", "", "Filter by AST node type (function, class, interface)")

	// RAG search flags
	ragSearchCmd.Flags().StringVar(&searchMode, "mode", "hybrid", "Search mode: semantic, keyword, hybrid")
	ragSearchCmd.Flags().IntVar(&searchLimit, "limit", 10, "Maximum results to return")

	// RAG explain flags
	ragExplainCmd.Flags().StringVar(&searchLanguage, "language", "", "Programming language hint")

	// RAG similar flags
	ragSimilarCmd.Flags().IntVar(&searchLimit, "limit", 5, "Maximum similar results")

	// Embedding flags
	embeddingCmd.Flags().StringVar(&embeddingProvider, "provider", "auto", "Embedding provider: auto, local, openai, voyage, ollama")

	// Ingest repo flags
	ingestRepoCmd.Flags().StringVar(&ingestOwner, "owner", "", "Repository owner (required)")
	ingestRepoCmd.Flags().StringVar(&ingestRepo, "repo", "", "Repository name (required)")
	ingestRepoCmd.Flags().BoolVar(&ingestIncludeCode, "include-code", false, "Include source code files")
	ingestRepoCmd.Flags().BoolVar(&ingestIncludeDocs, "include-docs", true, "Include documentation files (default: true)")
	ingestRepoCmd.Flags().BoolVar(&ingestIncludeTests, "include-tests", false, "Include test files")
	ingestRepoCmd.Flags().IntVar(&ingestMaxFileSizeKB, "max-file-size", 500, "Maximum file size in KB")

	// Ingest org flags
	ingestOrgCmd.Flags().StringVar(&ingestOwner, "owner", "", "Organization owner (required)")
	ingestOrgCmd.Flags().BoolVar(&ingestIncludeCode, "include-code", false, "Include source code files")
	ingestOrgCmd.Flags().BoolVar(&ingestIncludeDocs, "include-docs", true, "Include documentation files (default: true)")
	ingestOrgCmd.Flags().BoolVar(&ingestIncludeTests, "include-tests", false, "Include test files")
	ingestOrgCmd.Flags().IntVar(&ingestMaxFileSizeKB, "max-file-size", 500, "Maximum file size in KB")
	ingestOrgCmd.Flags().BoolVar(&ingestScheduleDaily, "schedule-daily", false, "Schedule daily re-ingestion at 2 AM")

	// Ingest history flags
	ingestHistoryCmd.Flags().StringVar(&ingestOwner, "owner", "", "Filter by owner")
	ingestHistoryCmd.Flags().StringVar(&ingestRepo, "repo", "", "Filter by repo")
	ingestHistoryCmd.Flags().IntVar(&searchLimit, "limit", 20, "Maximum results to return")

	// Analyze run flags
	analyzeRunCmd.Flags().StringVar(&ingestOwner, "owner", "", "Repository owner (required)")
	analyzeRunCmd.Flags().StringVar(&ingestRepo, "repo", "", "Repository name (required)")
	analyzeRunCmd.Flags().StringVar(&analyzeType, "type", "codebaseExplain", "Analysis type: codebaseExplain, patterns, issues, wiki, copilot")
	analyzeRunCmd.Flags().BoolVar(&analyzeForce, "force", false, "Force refresh (ignore cache)")

	// Analyze results flags
	analyzeResultsCmd.Flags().StringVar(&ingestOwner, "owner", "", "Repository owner (required)")
	analyzeResultsCmd.Flags().StringVar(&ingestRepo, "repo", "", "Repository name (required)")
}
