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

	// RAG subcommands
	gatewayRagCmd.AddCommand(ragSearchCmd)
	gatewayRagCmd.AddCommand(ragExplainCmd)
	gatewayRagCmd.AddCommand(ragSimilarCmd)
	gatewayRagCmd.AddCommand(ragIndexCmd)

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
}
