package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var workflowCmd = &cobra.Command{
	Use:   "workflow",
	Short: "Developer workflow automation commands",
	Long: `Commands for automating development workflows including:
- Feature branch creation with proper naming
- Pre-commit checks and validation
- PR creation with templates
- Environment promotion (guest → main)
- Task tracking and status updates`,
}

// Feature branch creation
var featureBranchCmd = &cobra.Command{
	Use:   "feature [task-id] [description]",
	Short: "Create a new feature branch following GitFlow conventions",
	Long: `Creates a properly named feature branch from the latest develop/guest branch.

Examples:
  seip workflow feature SEIP-123 add-user-profile
  seip workflow feature SEIP-456 fix-oauth-redirect --type bugfix
  seip workflow feature SEIP-789 critical-security-patch --type hotfix`,
	Args: cobra.MinimumNArgs(2),
	Run:  runFeatureBranch,
}

var (
	branchType   string
	baseBranch   string
	skipPull     bool
	announceWork bool
)

func init() {
	rootCmd.AddCommand(workflowCmd)

	// Feature branch flags
	featureBranchCmd.Flags().StringVarP(&branchType, "type", "t", "feature", "Branch type: feature, bugfix, hotfix")
	featureBranchCmd.Flags().StringVarP(&baseBranch, "base", "b", "", "Base branch (default: develop or guest)")
	featureBranchCmd.Flags().BoolVar(&skipPull, "skip-pull", false, "Skip pulling latest changes")
	featureBranchCmd.Flags().BoolVar(&announceWork, "announce", true, "Announce work in task tracker")

	// Pre-commit check flags
	preCommitCmd.Flags().BoolVar(&runTests, "tests", true, "Run tests")
	preCommitCmd.Flags().BoolVar(&runLint, "lint", true, "Run linter")
	preCommitCmd.Flags().BoolVar(&runBuild, "build", false, "Run build check")
	preCommitCmd.Flags().BoolVar(&runTypeCheck, "types", true, "Run TypeScript type checking")

	// PR creation flags
	createPRCmd.Flags().StringVar(&prBase, "base", "", "Base branch for PR (default: develop)")
	createPRCmd.Flags().StringVar(&prTitle, "title", "", "PR title (auto-generated if not provided)")
	createPRCmd.Flags().BoolVar(&draftPR, "draft", false, "Create as draft PR")
	createPRCmd.Flags().BoolVar(&autoMerge, "auto-merge", false, "Enable auto-merge when checks pass")

	// Promote flags
	promoteCmd.Flags().BoolVar(&dryRunPromote, "dry-run", false, "Show what would be promoted without doing it")
	promoteCmd.Flags().BoolVar(&skipChecklist, "skip-checklist", false, "Skip pre-promotion checklist")

	// Status flags
	workflowStatusCmd.Flags().BoolVar(&showAllTasks, "all", false, "Show all tasks including completed")
	workflowStatusCmd.Flags().StringVar(&filterByUser, "user", "", "Filter tasks by user")

	workflowCmd.AddCommand(featureBranchCmd)
	workflowCmd.AddCommand(preCommitCmd)
	workflowCmd.AddCommand(createPRCmd)
	workflowCmd.AddCommand(promoteCmd)
	workflowCmd.AddCommand(workflowStatusCmd)
	workflowCmd.AddCommand(checklistCmd)
	workflowCmd.AddCommand(workflowSyncCmd)
}

func runFeatureBranch(cmd *cobra.Command, args []string) {
	taskID := args[0]
	description := strings.Join(args[1:], "-")

	// Sanitize description for branch name
	description = strings.ToLower(description)
	description = strings.ReplaceAll(description, " ", "-")

	// Determine base branch
	if baseBranch == "" {
		baseBranch = detectBaseBranch()
	}

	// Validate branch type
	validTypes := map[string]bool{"feature": true, "bugfix": true, "hotfix": true}
	if !validTypes[branchType] {
		fmt.Println("❌ Invalid branch type. Use: feature, bugfix, or hotfix")
		os.Exit(1)
	}

	branchName := fmt.Sprintf("%s/%s-%s", branchType, taskID, description)
	fmt.Printf("🌿 Creating branch: %s\n", branchName)
	fmt.Printf("   Base: %s\n", baseBranch)

	if !skipPull {
		fmt.Println("📥 Pulling latest changes...")
		runGitCommand("checkout", baseBranch)
		runGitCommand("pull", "origin", baseBranch)
	}

	fmt.Println("🔀 Creating and switching to new branch...")
	runGitCommand("checkout", "-b", branchName)

	fmt.Println("📤 Pushing branch to origin...")
	runGitCommand("push", "-u", "origin", branchName)

	fmt.Println()
	fmt.Println("✅ Branch created successfully!")
	fmt.Println()
	fmt.Printf("📋 Next steps:\n")
	fmt.Printf("   1. Make your changes\n")
	fmt.Printf("   2. Run: seip workflow pre-commit\n")
	fmt.Printf("   3. Commit with: git commit -m \"%s: description\"\n", getCommitType(branchType))
	fmt.Printf("   4. Create PR: seip workflow pr\n")
}

func detectBaseBranch() string {
	// Check if 'guest' branch exists (SEIP workflow)
	out, err := exec.Command("git", "branch", "-r").Output()
	if err == nil && strings.Contains(string(out), "origin/guest") {
		return "guest"
	}
	// Check for 'develop' branch (GitFlow)
	if strings.Contains(string(out), "origin/develop") {
		return "develop"
	}
	// Default to main
	return "main"
}

func getCommitType(branchType string) string {
	switch branchType {
	case "feature":
		return "feat"
	case "bugfix":
		return "fix"
	case "hotfix":
		return "fix"
	default:
		return "feat"
	}
}

func runGitCommand(args ...string) {
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("❌ Git command failed: git %s\n", strings.Join(args, " "))
		os.Exit(1)
	}
}

// Pre-commit checks
var preCommitCmd = &cobra.Command{
	Use:   "pre-commit",
	Short: "Run pre-commit checks (tests, lint, type-check)",
	Long: `Runs a comprehensive pre-commit check suite including:
- Unit tests
- Linting (ESLint)
- TypeScript type checking
- Build verification (optional)

This ensures code quality before committing.`,
	Run: runPreCommit,
}

var (
	runTests     bool
	runLint      bool
	runBuild     bool
	runTypeCheck bool
)

func runPreCommit(cmd *cobra.Command, args []string) {
	fmt.Println("🔍 Running pre-commit checks...")
	fmt.Println()

	allPassed := true

	// Type checking
	if runTypeCheck {
		fmt.Println("📝 TypeScript type checking...")
		if !runNpmScript("type-check", "tsc --noEmit") {
			allPassed = false
		}
	}

	// Linting
	if runLint {
		fmt.Println("🧹 Running linter...")
		if !runNpmScript("lint", "eslint . --ext .ts,.tsx") {
			allPassed = false
		}
	}

	// Tests
	if runTests {
		fmt.Println("🧪 Running tests...")
		if !runNpmScript("test", "jest") {
			allPassed = false
		}
	}

	// Build
	if runBuild {
		fmt.Println("🏗️  Verifying build...")
		if !runNpmScript("build", "npm run build") {
			allPassed = false
		}
	}

	fmt.Println()
	if allPassed {
		fmt.Println("✅ All pre-commit checks passed!")
		fmt.Println("   You can now commit your changes.")
	} else {
		fmt.Println("❌ Some checks failed. Please fix the issues before committing.")
		os.Exit(1)
	}
}

func runNpmScript(name, fallback string) bool {
	// Try pnpm first, then npm
	var cmd *exec.Cmd
	if _, err := exec.LookPath("pnpm"); err == nil {
		cmd = exec.Command("pnpm", name)
	} else {
		cmd = exec.Command("npm", "run", name)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("   ❌ %s failed\n", name)
		return false
	}
	fmt.Printf("   ✅ %s passed\n", name)
	return true
}

// PR creation
var createPRCmd = &cobra.Command{
	Use:   "pr",
	Short: "Create a pull request with proper template",
	Long: `Creates a pull request with:
- Auto-generated title from branch name
- Pre-filled description template
- Proper base branch selection
- Optional draft mode
- Optional auto-merge enablement`,
	Run: runCreatePR,
}

var (
	prBase    string
	prTitle   string
	draftPR   bool
	autoMerge bool
)

func runCreatePR(cmd *cobra.Command, args []string) {
	// Get current branch
	branchBytes, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		fmt.Println("❌ Failed to get current branch")
		os.Exit(1)
	}
	currentBranch := strings.TrimSpace(string(branchBytes))

	// Determine base branch
	if prBase == "" {
		prBase = detectBaseBranch()
	}

	// Generate title from branch if not provided
	if prTitle == "" {
		prTitle = generatePRTitle(currentBranch)
	}

	fmt.Printf("📝 Creating PR: %s\n", prTitle)
	fmt.Printf("   From: %s → %s\n", currentBranch, prBase)

	// Build gh command
	ghArgs := []string{"pr", "create",
		"--base", prBase,
		"--title", prTitle,
		"--body", generatePRBody(currentBranch),
	}

	if draftPR {
		ghArgs = append(ghArgs, "--draft")
	}

	ghCmd := exec.Command("gh", ghArgs...)
	ghCmd.Stdout = os.Stdout
	ghCmd.Stderr = os.Stderr

	if err := ghCmd.Run(); err != nil {
		fmt.Println("❌ Failed to create PR")
		fmt.Println("   Make sure you have gh CLI installed and authenticated")
		os.Exit(1)
	}

	if autoMerge {
		fmt.Println("🔄 Enabling auto-merge...")
		amCmd := exec.Command("gh", "pr", "merge", "--auto", "--merge")
		amCmd.Run()
	}

	fmt.Println()
	fmt.Println("✅ PR created successfully!")
}

func generatePRTitle(branch string) string {
	// Parse branch name: type/TASK-ID-description
	parts := strings.SplitN(branch, "/", 2)
	if len(parts) != 2 {
		return branch
	}

	branchType := parts[0]
	rest := parts[1]

	// Find task ID
	restParts := strings.SplitN(rest, "-", 3)
	if len(restParts) >= 3 {
		taskID := restParts[0] + "-" + restParts[1]
		description := strings.ReplaceAll(restParts[2], "-", " ")
		return fmt.Sprintf("[%s] %s: %s", taskID, branchType, description)
	}

	return fmt.Sprintf("%s: %s", branchType, strings.ReplaceAll(rest, "-", " "))
}

func generatePRBody(branch string) string {
	return `## Summary
Brief description of changes

## Changes
- Change 1
- Change 2

## Testing
- [ ] Unit tests added/updated
- [ ] Integration tests pass
- [ ] Manual testing completed

## Pre-merge Checklist
- [ ] Code follows project style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] No merge conflicts
- [ ] CI/CD checks pass
- [ ] No secrets or credentials in code
`
}

// Promote to production
var promoteCmd = &cobra.Command{
	Use:   "promote",
	Short: "Promote changes from guest/develop to main",
	Long: `Promotes tested changes from staging (guest/develop) to production (main).

This command:
1. Verifies pre-promotion checklist
2. Creates a release branch
3. Creates PR to main with promotion details
4. Optionally triggers deployment after merge`,
	Run: runPromote,
}

var (
	dryRunPromote bool
	skipChecklist bool
)

func runPromote(cmd *cobra.Command, args []string) {
	fmt.Println("🚀 Preparing production promotion...")
	fmt.Println()

	sourceBranch := detectBaseBranch()
	if sourceBranch == "main" {
		fmt.Println("❌ Already on main branch. Nothing to promote.")
		os.Exit(1)
	}

	if !skipChecklist {
		fmt.Println("📋 Pre-promotion checklist:")
		fmt.Println()
		checklistItems := []string{
			"All tests passing (backend, frontend, integration)",
			"No TypeScript errors (build succeeds)",
			"Code reviewed and approved",
			"Staging environment tested",
			"Database migrations verified",
			"Performance benchmarks met",
			"Security scan completed",
			"Documentation updated",
		}

		for i, item := range checklistItems {
			fmt.Printf("   %d. [ ] %s\n", i+1, item)
		}
		fmt.Println()
		fmt.Println("⚠️  Ensure all items are checked before proceeding!")
		fmt.Println("   Use --skip-checklist to bypass (not recommended)")
		fmt.Println()
	}

	releaseBranch := fmt.Sprintf("release/promote-%s", time.Now().Format("20060102"))
	fmt.Printf("📦 Release branch: %s\n", releaseBranch)

	if dryRunPromote {
		fmt.Println()
		fmt.Println("🔍 Dry run - would execute:")
		fmt.Printf("   1. git checkout %s && git pull\n", sourceBranch)
		fmt.Printf("   2. git checkout -b %s\n", releaseBranch)
		fmt.Printf("   3. git push -u origin %s\n", releaseBranch)
		fmt.Println("   4. gh pr create --base main")
		return
	}

	// Execute promotion
	fmt.Println()
	fmt.Printf("📥 Switching to %s and pulling latest...\n", sourceBranch)
	runGitCommand("checkout", sourceBranch)
	runGitCommand("pull", "origin", sourceBranch)

	fmt.Printf("🔀 Creating release branch %s...\n", releaseBranch)
	runGitCommand("checkout", "-b", releaseBranch)
	runGitCommand("push", "-u", "origin", releaseBranch)

	fmt.Println("📝 Creating promotion PR...")
	prBody := generatePromotionPRBody(sourceBranch)

	ghCmd := exec.Command("gh", "pr", "create",
		"--base", "main",
		"--title", fmt.Sprintf("chore: promote %s to production - %s", sourceBranch, time.Now().Format("2006-01-02")),
		"--body", prBody,
	)
	ghCmd.Stdout = os.Stdout
	ghCmd.Stderr = os.Stderr
	ghCmd.Run()

	fmt.Println()
	fmt.Println("✅ Promotion PR created!")
	fmt.Println("   Next: Request review, merge when approved, then realign environments")
}

func generatePromotionPRBody(source string) string {
	// Get commits being promoted
	commitsBytes, _ := exec.Command("git", "log", fmt.Sprintf("main..%s", source), "--oneline", "--no-decorate").Output()
	commits := string(commitsBytes)
	if len(commits) > 2000 {
		commits = commits[:2000] + "\n... (truncated)"
	}

	return fmt.Sprintf(`## Production Promotion

Promotes tested changes from %s environment to production.

### Pre-Deployment Checklist
- [ ] Backend unit tests passed
- [ ] Frontend unit tests passed
- [ ] Integration tests passed
- [ ] Staging deployment verified
- [ ] RAG system healthy
- [ ] Manual UI testing passed
- [ ] Performance benchmarks met
- [ ] Database migrations verified

### Changes Included
%s

### Deployment Plan
1. Merge this PR to main
2. CI/CD deploys to production automatically
3. Verify production health endpoints
4. Run smoke tests
5. Tag release
6. Realign staging with main

🚀 Ready for production deployment
`, source, commits)
}

// Status command
var workflowStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current workflow status and active tasks",
	Long:  `Displays the current git status, active branches, and task tracking information.`,
	Run:   runWorkflowStatus,
}

var (
	showAllTasks bool
	filterByUser string
)

func runWorkflowStatus(cmd *cobra.Command, args []string) {
	fmt.Println("📊 Workflow Status")
	fmt.Println("==================")
	fmt.Println()

	// Current branch
	branchBytes, _ := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	currentBranch := strings.TrimSpace(string(branchBytes))
	fmt.Printf("🌿 Current branch: %s\n", currentBranch)

	// Git status
	statusBytes, _ := exec.Command("git", "status", "--short").Output()
	status := strings.TrimSpace(string(statusBytes))
	if status == "" {
		fmt.Println("📁 Working directory: Clean")
	} else {
		lines := strings.Split(status, "\n")
		fmt.Printf("📁 Working directory: %d files changed\n", len(lines))
	}

	// Unpushed commits
	unpushedBytes, _ := exec.Command("git", "log", "@{u}..", "--oneline").Output()
	unpushed := strings.TrimSpace(string(unpushedBytes))
	if unpushed == "" {
		fmt.Println("📤 Unpushed commits: None")
	} else {
		lines := strings.Split(unpushed, "\n")
		fmt.Printf("📤 Unpushed commits: %d\n", len(lines))
	}

	fmt.Println()

	// Active branches
	fmt.Println("🔀 Active feature branches:")
	branchesBytes, _ := exec.Command("git", "branch", "-r", "--sort=-committerdate").Output()
	branches := strings.Split(string(branchesBytes), "\n")
	count := 0
	for _, b := range branches {
		b = strings.TrimSpace(b)
		if strings.Contains(b, "feature/") || strings.Contains(b, "bugfix/") || strings.Contains(b, "hotfix/") {
			fmt.Printf("   %s\n", b)
			count++
			if count >= 10 {
				fmt.Println("   ... (showing first 10)")
				break
			}
		}
	}

	if count == 0 {
		fmt.Println("   (none)")
	}

	fmt.Println()

	// Check if connected to SEIP API for task tracking
	client := getWorkflowClient()
	if client != nil {
		fmt.Println("📋 Task Tracking:")
		// Would query API for active tasks
		fmt.Println("   (Connect to SEIP API for task tracking)")
	}
}

// Checklist command
var checklistCmd = &cobra.Command{
	Use:   "checklist [type]",
	Short: "Show workflow checklists",
	Long: `Display workflow checklists for different scenarios:
- pre-commit: Before committing code
- pre-pr: Before creating a pull request
- pre-merge: Before merging to main/production
- deployment: Post-deployment verification`,
	Args: cobra.MaximumNArgs(1),
	Run:  runChecklist,
}

func runChecklist(cmd *cobra.Command, args []string) {
	checklistType := "pre-commit"
	if len(args) > 0 {
		checklistType = args[0]
	}

	fmt.Printf("📋 %s Checklist\n", strings.Title(strings.ReplaceAll(checklistType, "-", " ")))
	fmt.Println("=" + strings.Repeat("=", len(checklistType)+10))
	fmt.Println()

	switch checklistType {
	case "pre-commit":
		fmt.Println("Code Quality:")
		fmt.Println("  [ ] Code follows project style guidelines")
		fmt.Println("  [ ] Self-review completed")
		fmt.Println("  [ ] No console.log or debug statements")
		fmt.Println("  [ ] No hardcoded secrets or credentials")
		fmt.Println()
		fmt.Println("Testing:")
		fmt.Println("  [ ] Unit tests added for new code")
		fmt.Println("  [ ] All existing tests pass")
		fmt.Println("  [ ] TypeScript compiles without errors")
		fmt.Println("  [ ] Linter passes with no warnings")

	case "pre-pr":
		fmt.Println("Before Creating PR:")
		fmt.Println("  [ ] Branch is up-to-date with base branch")
		fmt.Println("  [ ] All commits use conventional format")
		fmt.Println("  [ ] Tests pass: pnpm test")
		fmt.Println("  [ ] Build succeeds: pnpm build")
		fmt.Println("  [ ] Lint passes: pnpm lint")
		fmt.Println("  [ ] Feature has been manually tested")
		fmt.Println("  [ ] Documentation updated if needed")
		fmt.Println("  [ ] PR description is clear and complete")

	case "pre-merge":
		fmt.Println("Pre-Merge to Production:")
		fmt.Println()
		fmt.Println("Code Quality:")
		fmt.Println("  [ ] All CI/CD checks passing")
		fmt.Println("  [ ] Code reviewed by at least one team member")
		fmt.Println("  [ ] No merge conflicts")
		fmt.Println()
		fmt.Println("Testing:")
		fmt.Println("  [ ] Backend coverage > 90%")
		fmt.Println("  [ ] Frontend coverage > 85%")
		fmt.Println("  [ ] Integration tests passing")
		fmt.Println("  [ ] Staging environment tested")
		fmt.Println("  [ ] Performance benchmarks met (Lighthouse > 90)")
		fmt.Println()
		fmt.Println("Infrastructure:")
		fmt.Println("  [ ] Database migrations tested and reversible")
		fmt.Println("  [ ] Environment variables documented")
		fmt.Println("  [ ] No breaking API changes")
		fmt.Println()
		fmt.Println("RAG System (if applicable):")
		fmt.Println("  [ ] Embeddings sync working")
		fmt.Println("  [ ] Vector search returning results")
		fmt.Println("  [ ] Queue processing correctly")

	case "deployment":
		fmt.Println("Post-Deployment Verification:")
		fmt.Println()
		fmt.Println("Health Checks:")
		fmt.Println("  [ ] /health endpoint returns 200")
		fmt.Println("  [ ] /api/v1/rag/health returns healthy")
		fmt.Println("  [ ] Database connections stable")
		fmt.Println("  [ ] Redis cache accessible")
		fmt.Println()
		fmt.Println("Smoke Tests:")
		fmt.Println("  [ ] Login/authentication works")
		fmt.Println("  [ ] Main dashboard loads")
		fmt.Println("  [ ] Key API endpoints respond")
		fmt.Println("  [ ] No errors in CloudWatch/logs")
		fmt.Println()
		fmt.Println("Monitoring:")
		fmt.Println("  [ ] CloudWatch alarms not firing")
		fmt.Println("  [ ] Error rate < 1%")
		fmt.Println("  [ ] Response times normal")

	default:
		fmt.Printf("Unknown checklist type: %s\n", checklistType)
		fmt.Println("Available: pre-commit, pre-pr, pre-merge, deployment")
	}
}

// Sync command - sync with remote and resolve issues
var workflowSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync current branch with remote and base branch",
	Long: `Syncs your current branch by:
1. Stashing any uncommitted changes
2. Fetching latest from origin
3. Merging base branch into current branch
4. Restoring stashed changes

This helps avoid merge conflicts and keeps branches up to date.`,
	Run: runWorkflowSync,
}

func runWorkflowSync(cmd *cobra.Command, args []string) {
	// Get current branch
	branchBytes, _ := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	currentBranch := strings.TrimSpace(string(branchBytes))

	base := detectBaseBranch()

	fmt.Printf("🔄 Syncing %s with %s\n", currentBranch, base)
	fmt.Println()

	// Check for uncommitted changes
	statusBytes, _ := exec.Command("git", "status", "--porcelain").Output()
	hasChanges := len(strings.TrimSpace(string(statusBytes))) > 0

	if hasChanges {
		fmt.Println("📦 Stashing uncommitted changes...")
		runGitCommand("stash", "push", "-m", fmt.Sprintf("Auto-stash before sync %s", time.Now().Format("20060102-150405")))
	}

	fmt.Println("📥 Fetching latest from origin...")
	runGitCommand("fetch", "origin")

	fmt.Printf("📥 Pulling latest %s...\n", base)
	runGitCommand("checkout", base)
	runGitCommand("pull", "origin", base)

	fmt.Printf("🔀 Switching back to %s...\n", currentBranch)
	runGitCommand("checkout", currentBranch)

	fmt.Printf("🔀 Merging %s into %s...\n", base, currentBranch)

	mergeCmd := exec.Command("git", "merge", base)
	mergeCmd.Stdout = os.Stdout
	mergeCmd.Stderr = os.Stderr
	if err := mergeCmd.Run(); err != nil {
		fmt.Println()
		fmt.Println("⚠️  Merge conflicts detected!")
		fmt.Println("   Please resolve conflicts, then run:")
		fmt.Println("   git add . && git commit")
		if hasChanges {
			fmt.Println()
			fmt.Println("   Don't forget to restore your stashed changes:")
			fmt.Println("   git stash pop")
		}
		os.Exit(1)
	}

	if hasChanges {
		fmt.Println("📦 Restoring stashed changes...")
		exec.Command("git", "stash", "pop").Run()
	}

	fmt.Println()
	fmt.Println("✅ Branch synced successfully!")
}

// WorkflowConfig for API calls
type WorkflowConfig struct {
	TaskID      string `json:"task_id"`
	Branch      string `json:"branch"`
	Status      string `json:"status"`
	Description string `json:"description"`
	StartedAt   string `json:"started_at"`
}

// APIClient represents a connection to the SEIP API
type APIClient struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

// getWorkflowClient returns an API client if authenticated, nil otherwise
func getWorkflowClient() *APIClient {
	// Check for token in environment or config file
	token := os.Getenv("SEIP_API_TOKEN")
	if token == "" {
		// Try to read from config file
		configPath := os.ExpandEnv("$HOME/.armyknife/config.json")
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil
		}
		var cfg struct {
			AccessToken string `json:"access_token"`
			APIURL      string `json:"api_url"`
		}
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil
		}
		if cfg.AccessToken == "" {
			return nil
		}
		token = cfg.AccessToken
	}

	baseURL := os.Getenv("SEIP_API_URL")
	if baseURL == "" {
		baseURL = apiURL
	}

	return &APIClient{
		BaseURL: baseURL,
		Token:   token,
		HTTP:    &http.Client{Timeout: 30 * time.Second},
	}
}

func announceTask(config WorkflowConfig) error {
	client := getWorkflowClient()
	if client == nil {
		return fmt.Errorf("not authenticated")
	}

	jsonData, _ := json.Marshal(config)
	req, _ := http.NewRequest("POST", client.BaseURL+"/api/v1/workflow/tasks", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+client.Token)

	resp, err := client.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	return nil
}
