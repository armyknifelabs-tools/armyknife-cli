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
	reviewFile       string
	reviewPRNumber   int
	reviewOutputFile string
	reviewFormat     string
	reviewStandard   string
	reviewLocal      bool
	reviewModel      string
)

// reviewCmd represents the review parent command
var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "AI-powered code review and analysis (The Cockpit)",
	Long: `Comprehensive AI-powered code intelligence commands.

The CLI is the COCKPIT - the command center that drives all code intelligence operations.
The Platform is the MEMORY - stores embeddings, patterns, metrics, and analysis history.

Operations:
  review code     - Full AI code review (quality, style, bugs)
  review pr       - Review a Pull Request
  review security - OWASP security scan
  review patterns - Detect code patterns and anti-patterns
  review standards - Check against code standards
  review architecture - Analyze code architecture/design
  review flow     - Generate code flow diagram (entry/exit points)
  review generate-pr - AI-assisted PR creation

Modes:
  --local   Use local Ollama/node-llm for private analysis
  --cloud   Use API Gateway (default) for powerful models

Examples:
  armyknife review code src/services/auth.ts
  armyknife review pr 123 --owner myorg --repo myrepo
  armyknife review security src/ --standard owasp-top-10
  armyknife review patterns src/services/ --output patterns.md
  armyknife review flow src/main.go --output flow-diagram.md
  armyknife review generate-pr --title "Add authentication" --branch feature/auth`,
}

// reviewCodeCmd performs AI code review
var reviewCodeCmd = &cobra.Command{
	Use:   "code <file-or-directory>",
	Short: "AI-powered code review",
	Long: `Perform comprehensive AI code review including:
  - Code quality analysis
  - Bug detection
  - Style and readability
  - Performance suggestions
  - Best practices compliance

Can run locally (Ollama/node-llm) or via API Gateway (Claude/GPT-4).

Examples:
  armyknife review code src/auth.ts
  armyknife review code src/services/ --local
  armyknife review code . --model gpt-4
  armyknife review code src/ --output review.md`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		target := args[0]

		fmt.Printf("🔍 AI Code Review\n")
		fmt.Printf("   Target: %s\n", target)
		if reviewLocal {
			fmt.Printf("   Mode: Local (Ollama/node-llm)\n")
		} else {
			fmt.Printf("   Mode: Cloud Gateway\n")
		}
		if reviewModel != "" {
			fmt.Printf("   Model: %s\n", reviewModel)
		}
		fmt.Println()

		// Read file content
		content, err := readFileOrDir(target)
		if err != nil {
			fmt.Printf("❌ Error reading target: %v\n", err)
			os.Exit(1)
		}

		reqBody := map[string]interface{}{
			"code":       content,
			"reviewType": "comprehensive",
			"target":     target,
			"options": map[string]interface{}{
				"checkBugs":        true,
				"checkStyle":       true,
				"checkPerformance": true,
				"checkSecurity":    true,
				"suggestRefactors": true,
			},
		}

		if reviewLocal {
			reqBody["provider"] = "local"
		}
		if reviewModel != "" {
			reqBody["model"] = reviewModel
		}

		result := callReviewAPI("/ai/review/code", reqBody)
		displayReviewResult(result, "Code Review")
	},
}

// reviewPRCmd reviews a Pull Request
var reviewPRCmd = &cobra.Command{
	Use:   "pr <pr-number>",
	Short: "Review a Pull Request",
	Long: `AI-powered Pull Request review including:
  - Code changes analysis
  - Impact assessment
  - Bug/issue detection
  - Style compliance
  - Security implications
  - Test coverage check
  - Merge recommendation

Examples:
  armyknife review pr 123 --owner myorg --repo myrepo
  armyknife review pr 456 --owner myorg --repo myrepo --local
  armyknife review pr 789 --output pr-review.md`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		prNumber := args[0]

		if ingestOwner == "" || ingestRepo == "" {
			fmt.Println("❌ Error: --owner and --repo are required")
			os.Exit(1)
		}

		fmt.Printf("🔍 PR Review\n")
		fmt.Printf("   Repository: %s/%s\n", ingestOwner, ingestRepo)
		fmt.Printf("   PR: #%s\n", prNumber)
		fmt.Println()

		reqBody := map[string]interface{}{
			"owner":    ingestOwner,
			"repo":     ingestRepo,
			"prNumber": prNumber,
			"options": map[string]interface{}{
				"checkCode":     true,
				"checkTests":    true,
				"checkSecurity": true,
				"checkDocs":     true,
			},
		}

		if reviewLocal {
			reqBody["provider"] = "local"
		}
		if reviewModel != "" {
			reqBody["model"] = reviewModel
		}

		result := callReviewAPI("/ai/review/pr", reqBody)
		displayPRReviewResult(result)
	},
}

// reviewSecurityCmd performs security scan
var reviewSecurityCmd = &cobra.Command{
	Use:   "security <file-or-directory>",
	Short: "Security vulnerability scan (OWASP)",
	Long: `AI-powered security analysis including:
  - OWASP Top 10 vulnerabilities
  - SQL injection detection
  - XSS detection
  - Authentication issues
  - Secrets/credentials in code
  - Dependency vulnerabilities
  - Security best practices

Standards:
  - owasp-top-10 (default)
  - cwe-top-25
  - sans-top-25
  - pci-dss
  - hipaa

Examples:
  armyknife review security src/
  armyknife review security src/api/ --standard owasp-top-10
  armyknife review security . --output security-report.md`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		target := args[0]

		fmt.Printf("🛡️ Security Scan\n")
		fmt.Printf("   Target: %s\n", target)
		fmt.Printf("   Standard: %s\n", reviewStandard)
		fmt.Println()

		content, err := readFileOrDir(target)
		if err != nil {
			fmt.Printf("❌ Error reading target: %v\n", err)
			os.Exit(1)
		}

		reqBody := map[string]interface{}{
			"code":     content,
			"target":   target,
			"standard": reviewStandard,
			"checks": []string{
				"injection",
				"xss",
				"authentication",
				"authorization",
				"secrets",
				"cryptography",
				"dependencies",
			},
		}

		if reviewLocal {
			reqBody["provider"] = "local"
		}

		result := callReviewAPI("/ai/review/security", reqBody)
		displaySecurityResult(result)
	},
}

// reviewPatternsCmd detects code patterns
var reviewPatternsCmd = &cobra.Command{
	Use:   "patterns <file-or-directory>",
	Short: "Detect code patterns and anti-patterns",
	Long: `Detect coding patterns in your codebase:
  - Design patterns (Singleton, Factory, Observer, etc.)
  - Anti-patterns (God class, Spaghetti code, etc.)
  - Framework patterns (MVC, Repository, Service layer)
  - Custom patterns from your standards

Output can be used to:
  - Document existing patterns
  - Enforce consistency
  - Train new developers

Examples:
  armyknife review patterns src/services/
  armyknife review patterns src/ --output patterns.md
  armyknife review patterns . --format json`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		target := args[0]

		fmt.Printf("🔬 Pattern Detection\n")
		fmt.Printf("   Target: %s\n", target)
		fmt.Println()

		content, err := readFileOrDir(target)
		if err != nil {
			fmt.Printf("❌ Error reading target: %v\n", err)
			os.Exit(1)
		}

		reqBody := map[string]interface{}{
			"code":   content,
			"target": target,
			"detect": []string{
				"design_patterns",
				"anti_patterns",
				"framework_patterns",
				"custom_patterns",
			},
		}

		if reviewLocal {
			reqBody["provider"] = "local"
		}

		result := callReviewAPI("/ai/review/patterns", reqBody)
		displayPatternsResult(result)
	},
}

// reviewStandardsCmd checks code standards
var reviewStandardsCmd = &cobra.Command{
	Use:   "standards <file-or-directory>",
	Short: "Check against code standards",
	Long: `Check code against defined standards:
  - Naming conventions
  - File organization
  - Documentation requirements
  - Error handling patterns
  - Logging standards
  - Testing requirements

Can use organization-defined standards stored in the Platform (memory).

Examples:
  armyknife review standards src/
  armyknife review standards src/services/ --standard typescript-strict
  armyknife review standards . --output standards-report.md`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		target := args[0]

		fmt.Printf("📏 Code Standards Check\n")
		fmt.Printf("   Target: %s\n", target)
		if reviewStandard != "" {
			fmt.Printf("   Standard: %s\n", reviewStandard)
		}
		fmt.Println()

		content, err := readFileOrDir(target)
		if err != nil {
			fmt.Printf("❌ Error reading target: %v\n", err)
			os.Exit(1)
		}

		reqBody := map[string]interface{}{
			"code":   content,
			"target": target,
			"checks": []string{
				"naming",
				"organization",
				"documentation",
				"error_handling",
				"logging",
				"testing",
			},
		}

		if reviewStandard != "" {
			reqBody["standardSet"] = reviewStandard
		}
		if reviewLocal {
			reqBody["provider"] = "local"
		}

		result := callReviewAPI("/ai/review/standards", reqBody)
		displayStandardsResult(result)
	},
}

// reviewArchitectureCmd analyzes code architecture
var reviewArchitectureCmd = &cobra.Command{
	Use:   "architecture <file-or-directory>",
	Short: "Analyze code architecture and design",
	Long: `Analyze codebase architecture including:
  - Layer separation (controllers, services, repositories)
  - Dependency flow and coupling
  - Module boundaries
  - API design quality
  - Database schema patterns
  - Microservice boundaries

Generates:
  - Architecture diagram (ASCII/Mermaid)
  - Dependency graph
  - Improvement suggestions

Examples:
  armyknife review architecture src/
  armyknife review architecture . --output architecture.md
  armyknife review architecture src/services/ --format mermaid`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		target := args[0]

		fmt.Printf("🏗️ Architecture Analysis\n")
		fmt.Printf("   Target: %s\n", target)
		fmt.Printf("   Format: %s\n", reviewFormat)
		fmt.Println()

		content, err := readFileOrDir(target)
		if err != nil {
			fmt.Printf("❌ Error reading target: %v\n", err)
			os.Exit(1)
		}

		reqBody := map[string]interface{}{
			"code":         content,
			"target":       target,
			"outputFormat": reviewFormat,
			"analyze": []string{
				"layers",
				"dependencies",
				"modules",
				"api_design",
				"data_flow",
			},
		}

		if reviewLocal {
			reqBody["provider"] = "local"
		}

		result := callReviewAPI("/ai/review/architecture", reqBody)
		displayArchitectureResult(result)
	},
}

// reviewFlowCmd generates code flow diagram
var reviewFlowCmd = &cobra.Command{
	Use:   "flow <file>",
	Short: "Generate code flow diagram (entry/exit points)",
	Long: `Generate code flow visualization showing:
  - Entry points (main, handlers, exports)
  - Exit points (returns, throws, process.exit)
  - Control flow paths
  - Function call graph
  - Data flow tracking

Output formats:
  - mermaid (default) - Mermaid flowchart
  - ascii - ASCII art diagram
  - dot - GraphViz DOT format
  - json - Structured JSON

Examples:
  armyknife review flow src/main.go
  armyknife review flow src/server.ts --format mermaid
  armyknife review flow src/api/handler.py --output flow.md`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		target := args[0]

		fmt.Printf("📊 Code Flow Analysis\n")
		fmt.Printf("   Target: %s\n", target)
		fmt.Printf("   Format: %s\n", reviewFormat)
		fmt.Println()

		content, err := readFileOrDir(target)
		if err != nil {
			fmt.Printf("❌ Error reading target: %v\n", err)
			os.Exit(1)
		}

		reqBody := map[string]interface{}{
			"code":         content,
			"target":       target,
			"outputFormat": reviewFormat,
			"analyze": []string{
				"entry_points",
				"exit_points",
				"control_flow",
				"call_graph",
				"data_flow",
			},
		}

		if reviewLocal {
			reqBody["provider"] = "local"
		}

		result := callReviewAPI("/ai/review/flow", reqBody)
		displayFlowResult(result)
	},
}

// reviewGeneratePRCmd generates a PR with AI assistance
var reviewGeneratePRCmd = &cobra.Command{
	Use:   "generate-pr",
	Short: "AI-assisted PR generation",
	Long: `Generate a Pull Request with AI-assisted:
  - Title generation from changes
  - Description with context
  - Test plan suggestions
  - Reviewer recommendations
  - Related issues linking

Can analyze staged changes or a specific branch.

Examples:
  armyknife review generate-pr --title "Add auth feature"
  armyknife review generate-pr --branch feature/auth --base main
  armyknife review generate-pr --analyze-changes
  armyknife review generate-pr --draft`,
	Run: func(cmd *cobra.Command, args []string) {
		title, _ := cmd.Flags().GetString("title")
		branch, _ := cmd.Flags().GetString("branch")
		base, _ := cmd.Flags().GetString("base")
		analyzeChanges, _ := cmd.Flags().GetBool("analyze-changes")
		draft, _ := cmd.Flags().GetBool("draft")

		fmt.Printf("📝 AI-Assisted PR Generation\n")
		if title != "" {
			fmt.Printf("   Title: %s\n", title)
		}
		if branch != "" {
			fmt.Printf("   Branch: %s\n", branch)
		}
		if base != "" {
			fmt.Printf("   Base: %s\n", base)
		}
		fmt.Println()

		reqBody := map[string]interface{}{
			"title":          title,
			"branch":         branch,
			"base":           base,
			"analyzeChanges": analyzeChanges,
			"draft":          draft,
			"options": map[string]interface{}{
				"generateDescription": true,
				"generateTestPlan":    true,
				"suggestReviewers":    true,
				"linkIssues":          true,
			},
		}

		if reviewLocal {
			reqBody["provider"] = "local"
		}

		result := callReviewAPI("/ai/review/generate-pr", reqBody)
		displayGeneratePRResult(result)
	},
}

// checkPRCmd checks an existing PR for issues
var checkPRCmd = &cobra.Command{
	Use:   "check-pr <pr-number>",
	Short: "Check PR code for issues before merge",
	Long: `Comprehensive PR validation before merge:
  - Code quality check
  - Test coverage verification
  - Security scan
  - Breaking changes detection
  - Documentation completeness
  - CI/CD status check

Returns a merge readiness score and blockers.

Examples:
  armyknife review check-pr 123 --owner myorg --repo myrepo
  armyknife review check-pr 456 --require-tests --require-docs`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		prNumber := args[0]

		if ingestOwner == "" || ingestRepo == "" {
			fmt.Println("❌ Error: --owner and --repo are required")
			os.Exit(1)
		}

		fmt.Printf("✅ PR Validation Check\n")
		fmt.Printf("   Repository: %s/%s\n", ingestOwner, ingestRepo)
		fmt.Printf("   PR: #%s\n", prNumber)
		fmt.Println()

		reqBody := map[string]interface{}{
			"owner":    ingestOwner,
			"repo":     ingestRepo,
			"prNumber": prNumber,
			"checks": []string{
				"code_quality",
				"test_coverage",
				"security",
				"breaking_changes",
				"documentation",
				"ci_status",
			},
		}

		result := callReviewAPI("/ai/review/check-pr", reqBody)
		displayCheckPRResult(result)
	},
}

// Helper functions

func readFileOrDir(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	if info.IsDir() {
		// For directories, we'll send the path and let the API read it
		return fmt.Sprintf("[DIRECTORY:%s]", path), nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func callReviewAPI(endpoint string, reqBody map[string]interface{}) map[string]interface{} {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		os.Exit(1)
	}

	resp, err := http.Post(
		fmt.Sprintf("%s%s", apiURL, endpoint),
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

	return result
}

func displayReviewResult(result map[string]interface{}, title string) {
	if success, ok := result["success"].(bool); ok && success {
		data := result["data"].(map[string]interface{})

		fmt.Printf("✅ %s Complete\n", title)
		fmt.Println(strings.Repeat("─", 60))

		if summary, ok := data["summary"].(string); ok {
			fmt.Printf("\n📋 Summary:\n%s\n", summary)
		}

		if issues, ok := data["issues"].([]interface{}); ok && len(issues) > 0 {
			fmt.Printf("\n⚠️  Issues Found (%d):\n", len(issues))
			for i, issue := range issues {
				if issueMap, ok := issue.(map[string]interface{}); ok {
					severity := issueMap["severity"]
					icon := "⚪"
					switch severity {
					case "critical":
						icon = "🔴"
					case "high":
						icon = "🟠"
					case "medium":
						icon = "🟡"
					case "low":
						icon = "🟢"
					}
					fmt.Printf("   %d. %s %s\n", i+1, icon, issueMap["message"])
					if line, ok := issueMap["line"].(float64); ok {
						fmt.Printf("      Line %d\n", int(line))
					}
				}
			}
		}

		if suggestions, ok := data["suggestions"].([]interface{}); ok && len(suggestions) > 0 {
			fmt.Printf("\n💡 Suggestions:\n")
			for _, s := range suggestions {
				fmt.Printf("   • %s\n", s)
			}
		}

		if score, ok := data["score"].(float64); ok {
			fmt.Printf("\n📊 Quality Score: %.0f/100\n", score)
		}

		// Write to file if output specified
		if reviewOutputFile != "" {
			writeOutputFile(result, reviewOutputFile)
		}
	} else {
		displayError(result)
	}
}

func displayPRReviewResult(result map[string]interface{}) {
	if success, ok := result["success"].(bool); ok && success {
		data := result["data"].(map[string]interface{})

		fmt.Println("✅ PR Review Complete")
		fmt.Println(strings.Repeat("─", 60))

		if summary, ok := data["summary"].(string); ok {
			fmt.Printf("\n📋 Summary:\n%s\n", summary)
		}

		if changes, ok := data["changesAnalysis"].(map[string]interface{}); ok {
			fmt.Printf("\n📝 Changes Analysis:\n")
			if filesChanged, ok := changes["filesChanged"].(float64); ok {
				fmt.Printf("   Files changed: %d\n", int(filesChanged))
			}
			if additions, ok := changes["additions"].(float64); ok {
				fmt.Printf("   Additions: +%d\n", int(additions))
			}
			if deletions, ok := changes["deletions"].(float64); ok {
				fmt.Printf("   Deletions: -%d\n", int(deletions))
			}
		}

		if verdict, ok := data["verdict"].(string); ok {
			icon := "✅"
			if verdict == "request_changes" {
				icon = "🔄"
			} else if verdict == "reject" {
				icon = "❌"
			}
			fmt.Printf("\n%s Verdict: %s\n", icon, strings.ToUpper(verdict))
		}

		if reviewOutputFile != "" {
			writeOutputFile(result, reviewOutputFile)
		}
	} else {
		displayError(result)
	}
}

func displaySecurityResult(result map[string]interface{}) {
	if success, ok := result["success"].(bool); ok && success {
		data := result["data"].(map[string]interface{})

		fmt.Println("✅ Security Scan Complete")
		fmt.Println(strings.Repeat("─", 60))

		if vulns, ok := data["vulnerabilities"].([]interface{}); ok {
			if len(vulns) == 0 {
				fmt.Printf("\n✅ No vulnerabilities found!\n")
			} else {
				fmt.Printf("\n🚨 Vulnerabilities Found (%d):\n", len(vulns))
				for i, v := range vulns {
					if vuln, ok := v.(map[string]interface{}); ok {
						severity := vuln["severity"]
						icon := "⚪"
						switch severity {
						case "critical":
							icon = "🔴"
						case "high":
							icon = "🟠"
						case "medium":
							icon = "🟡"
						case "low":
							icon = "🟢"
						}
						fmt.Printf("\n   %d. %s %s (%s)\n", i+1, icon, vuln["type"], severity)
						if desc, ok := vuln["description"].(string); ok {
							fmt.Printf("      %s\n", desc)
						}
						if file, ok := vuln["file"].(string); ok {
							fmt.Printf("      File: %s", file)
							if line, ok := vuln["line"].(float64); ok {
								fmt.Printf(":%d", int(line))
							}
							fmt.Println()
						}
						if fix, ok := vuln["fix"].(string); ok {
							fmt.Printf("      Fix: %s\n", fix)
						}
					}
				}
			}
		}

		if score, ok := data["securityScore"].(float64); ok {
			fmt.Printf("\n🛡️ Security Score: %.0f/100\n", score)
		}

		if reviewOutputFile != "" {
			writeOutputFile(result, reviewOutputFile)
		}
	} else {
		displayError(result)
	}
}

func displayPatternsResult(result map[string]interface{}) {
	if success, ok := result["success"].(bool); ok && success {
		data := result["data"].(map[string]interface{})

		fmt.Println("✅ Pattern Detection Complete")
		fmt.Println(strings.Repeat("─", 60))

		if patterns, ok := data["designPatterns"].([]interface{}); ok && len(patterns) > 0 {
			fmt.Printf("\n🏗️ Design Patterns Found:\n")
			for _, p := range patterns {
				if pattern, ok := p.(map[string]interface{}); ok {
					fmt.Printf("   ✅ %s\n", pattern["name"])
					if location, ok := pattern["location"].(string); ok {
						fmt.Printf("      Location: %s\n", location)
					}
				}
			}
		}

		if antiPatterns, ok := data["antiPatterns"].([]interface{}); ok && len(antiPatterns) > 0 {
			fmt.Printf("\n⚠️  Anti-Patterns Detected:\n")
			for _, p := range antiPatterns {
				if pattern, ok := p.(map[string]interface{}); ok {
					fmt.Printf("   ❌ %s\n", pattern["name"])
					if suggestion, ok := pattern["suggestion"].(string); ok {
						fmt.Printf("      Suggestion: %s\n", suggestion)
					}
				}
			}
		}

		if reviewOutputFile != "" {
			writeOutputFile(result, reviewOutputFile)
		}
	} else {
		displayError(result)
	}
}

func displayStandardsResult(result map[string]interface{}) {
	if success, ok := result["success"].(bool); ok && success {
		data := result["data"].(map[string]interface{})

		fmt.Println("✅ Standards Check Complete")
		fmt.Println(strings.Repeat("─", 60))

		if violations, ok := data["violations"].([]interface{}); ok {
			if len(violations) == 0 {
				fmt.Printf("\n✅ All standards met!\n")
			} else {
				fmt.Printf("\n📏 Violations Found (%d):\n", len(violations))
				for i, v := range violations {
					if violation, ok := v.(map[string]interface{}); ok {
						fmt.Printf("   %d. %s\n", i+1, violation["rule"])
						if file, ok := violation["file"].(string); ok {
							fmt.Printf("      File: %s\n", file)
						}
						if suggestion, ok := violation["suggestion"].(string); ok {
							fmt.Printf("      Fix: %s\n", suggestion)
						}
					}
				}
			}
		}

		if compliance, ok := data["complianceScore"].(float64); ok {
			fmt.Printf("\n📊 Compliance Score: %.0f%%\n", compliance)
		}

		if reviewOutputFile != "" {
			writeOutputFile(result, reviewOutputFile)
		}
	} else {
		displayError(result)
	}
}

func displayArchitectureResult(result map[string]interface{}) {
	if success, ok := result["success"].(bool); ok && success {
		data := result["data"].(map[string]interface{})

		fmt.Println("✅ Architecture Analysis Complete")
		fmt.Println(strings.Repeat("─", 60))

		if summary, ok := data["summary"].(string); ok {
			fmt.Printf("\n📋 Architecture Overview:\n%s\n", summary)
		}

		if diagram, ok := data["diagram"].(string); ok {
			fmt.Printf("\n📊 Architecture Diagram:\n")
			fmt.Println("```")
			fmt.Println(diagram)
			fmt.Println("```")
		}

		if layers, ok := data["layers"].([]interface{}); ok && len(layers) > 0 {
			fmt.Printf("\n🏗️ Layers Detected:\n")
			for _, l := range layers {
				if layer, ok := l.(map[string]interface{}); ok {
					fmt.Printf("   • %s\n", layer["name"])
				}
			}
		}

		if suggestions, ok := data["suggestions"].([]interface{}); ok && len(suggestions) > 0 {
			fmt.Printf("\n💡 Improvement Suggestions:\n")
			for _, s := range suggestions {
				fmt.Printf("   • %s\n", s)
			}
		}

		if reviewOutputFile != "" {
			writeOutputFile(result, reviewOutputFile)
		}
	} else {
		displayError(result)
	}
}

func displayFlowResult(result map[string]interface{}) {
	if success, ok := result["success"].(bool); ok && success {
		data := result["data"].(map[string]interface{})

		fmt.Println("✅ Code Flow Analysis Complete")
		fmt.Println(strings.Repeat("─", 60))

		if entryPoints, ok := data["entryPoints"].([]interface{}); ok && len(entryPoints) > 0 {
			fmt.Printf("\n🚪 Entry Points:\n")
			for _, e := range entryPoints {
				if entry, ok := e.(map[string]interface{}); ok {
					fmt.Printf("   → %s (%s)\n", entry["name"], entry["type"])
				}
			}
		}

		if exitPoints, ok := data["exitPoints"].([]interface{}); ok && len(exitPoints) > 0 {
			fmt.Printf("\n🚶 Exit Points:\n")
			for _, e := range exitPoints {
				if exit, ok := e.(map[string]interface{}); ok {
					fmt.Printf("   ← %s (%s)\n", exit["name"], exit["type"])
				}
			}
		}

		if diagram, ok := data["flowDiagram"].(string); ok {
			fmt.Printf("\n📊 Flow Diagram (%s):\n", reviewFormat)
			fmt.Println("```" + reviewFormat)
			fmt.Println(diagram)
			fmt.Println("```")
		}

		if reviewOutputFile != "" {
			writeOutputFile(result, reviewOutputFile)
		}
	} else {
		displayError(result)
	}
}

func displayGeneratePRResult(result map[string]interface{}) {
	if success, ok := result["success"].(bool); ok && success {
		data := result["data"].(map[string]interface{})

		fmt.Println("✅ PR Generated")
		fmt.Println(strings.Repeat("─", 60))

		if title, ok := data["title"].(string); ok {
			fmt.Printf("\n📝 Title: %s\n", title)
		}

		if description, ok := data["description"].(string); ok {
			fmt.Printf("\n📋 Description:\n%s\n", description)
		}

		if testPlan, ok := data["testPlan"].(string); ok {
			fmt.Printf("\n🧪 Test Plan:\n%s\n", testPlan)
		}

		if reviewers, ok := data["suggestedReviewers"].([]interface{}); ok && len(reviewers) > 0 {
			fmt.Printf("\n👥 Suggested Reviewers:\n")
			for _, r := range reviewers {
				fmt.Printf("   • %s\n", r)
			}
		}

		if prUrl, ok := data["prUrl"].(string); ok {
			fmt.Printf("\n🔗 PR URL: %s\n", prUrl)
		}
	} else {
		displayError(result)
	}
}

func displayCheckPRResult(result map[string]interface{}) {
	if success, ok := result["success"].(bool); ok && success {
		data := result["data"].(map[string]interface{})

		fmt.Println("✅ PR Validation Complete")
		fmt.Println(strings.Repeat("─", 60))

		if ready, ok := data["mergeReady"].(bool); ok {
			if ready {
				fmt.Printf("\n✅ PR is ready to merge!\n")
			} else {
				fmt.Printf("\n❌ PR has blocking issues\n")
			}
		}

		if blockers, ok := data["blockers"].([]interface{}); ok && len(blockers) > 0 {
			fmt.Printf("\n🚫 Blockers:\n")
			for _, b := range blockers {
				fmt.Printf("   • %s\n", b)
			}
		}

		if warnings, ok := data["warnings"].([]interface{}); ok && len(warnings) > 0 {
			fmt.Printf("\n⚠️  Warnings:\n")
			for _, w := range warnings {
				fmt.Printf("   • %s\n", w)
			}
		}

		if score, ok := data["readinessScore"].(float64); ok {
			fmt.Printf("\n📊 Merge Readiness: %.0f%%\n", score)
		}
	} else {
		displayError(result)
	}
}

func displayError(result map[string]interface{}) {
	fmt.Printf("❌ Operation Failed\n")
	if errData, ok := result["error"].(map[string]interface{}); ok {
		fmt.Printf("   Error: %v\n", errData["message"])
		if details, ok := errData["details"]; ok {
			fmt.Printf("   Details: %v\n", details)
		}
	}
	os.Exit(1)
}

func writeOutputFile(result map[string]interface{}, filename string) {
	var output []byte
	var err error

	if strings.HasSuffix(filename, ".json") {
		output, err = json.MarshalIndent(result, "", "  ")
	} else {
		// Write as markdown
		var sb strings.Builder
		if data, ok := result["data"].(map[string]interface{}); ok {
			for key, value := range data {
				sb.WriteString(fmt.Sprintf("## %s\n\n", key))
				sb.WriteString(fmt.Sprintf("%v\n\n", value))
			}
		}
		output = []byte(sb.String())
	}

	if err != nil {
		fmt.Printf("⚠️  Error formatting output: %v\n", err)
		return
	}

	if err := os.WriteFile(filename, output, 0644); err != nil {
		fmt.Printf("⚠️  Error writing output file: %v\n", err)
		return
	}

	fmt.Printf("\n📄 Output written to: %s\n", filename)
}

func init() {
	rootCmd.AddCommand(reviewCmd)

	// Add subcommands
	reviewCmd.AddCommand(reviewCodeCmd)
	reviewCmd.AddCommand(reviewPRCmd)
	reviewCmd.AddCommand(reviewSecurityCmd)
	reviewCmd.AddCommand(reviewPatternsCmd)
	reviewCmd.AddCommand(reviewStandardsCmd)
	reviewCmd.AddCommand(reviewArchitectureCmd)
	reviewCmd.AddCommand(reviewFlowCmd)
	reviewCmd.AddCommand(reviewGeneratePRCmd)
	reviewCmd.AddCommand(checkPRCmd)

	// Global review flags
	reviewCmd.PersistentFlags().BoolVar(&reviewLocal, "local", false, "Use local AI (Ollama/node-llm)")
	reviewCmd.PersistentFlags().StringVar(&reviewModel, "model", "", "Specify model to use")
	reviewCmd.PersistentFlags().StringVarP(&reviewOutputFile, "output", "o", "", "Output file for results")
	reviewCmd.PersistentFlags().StringVar(&reviewFormat, "format", "mermaid", "Output format: mermaid, ascii, dot, json")

	// Code review flags
	reviewCodeCmd.Flags().StringVar(&reviewFile, "file", "", "Specific file to review")

	// PR review flags
	reviewPRCmd.Flags().StringVar(&ingestOwner, "owner", "", "Repository owner")
	reviewPRCmd.Flags().StringVar(&ingestRepo, "repo", "", "Repository name")

	// Security flags
	reviewSecurityCmd.Flags().StringVar(&reviewStandard, "standard", "owasp-top-10", "Security standard: owasp-top-10, cwe-top-25, pci-dss")

	// Standards flags
	reviewStandardsCmd.Flags().StringVar(&reviewStandard, "standard", "", "Standards set to check against")

	// Generate PR flags
	reviewGeneratePRCmd.Flags().String("title", "", "PR title")
	reviewGeneratePRCmd.Flags().String("branch", "", "Source branch")
	reviewGeneratePRCmd.Flags().String("base", "main", "Base branch")
	reviewGeneratePRCmd.Flags().Bool("analyze-changes", true, "Analyze staged/committed changes")
	reviewGeneratePRCmd.Flags().Bool("draft", false, "Create as draft PR")

	// Check PR flags
	checkPRCmd.Flags().StringVar(&ingestOwner, "owner", "", "Repository owner")
	checkPRCmd.Flags().StringVar(&ingestRepo, "repo", "", "Repository name")
	checkPRCmd.Flags().Bool("require-tests", false, "Require test coverage")
	checkPRCmd.Flags().Bool("require-docs", false, "Require documentation")
}
