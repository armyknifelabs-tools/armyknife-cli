package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	apiURL  string
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "armyknife",
	Short: "ArmyKnife CLI - Command line tool for the Software Engineering Intelligence Platform",
	Long: `ArmyKnife CLI is a comprehensive command-line tool for testing and interacting with
the ArmyKnifeLabs SEIP platform. It provides access to all API endpoints including:
- Authentication via OAuth device flow
- DORA metrics (Deployment Frequency, Lead Time, MTTR, Change Failure Rate)
- GitHub operations and repository management
- AI-powered code analysis and RAG queries
- Cache management and monitoring
- System health checks`,
}

// Execute adds all child commands to the root command and sets flags appropriately
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&apiURL, "api-url", "https://api.armyknifelabs.com/api/v1", "API base URL")
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.armyknife/config.json)")
}

func initConfig() {
	// Config initialization is handled in the config package
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("ArmyKnife CLI v0.3.0")
		fmt.Println()
		fmt.Println("Features:")
		fmt.Println("  - Multi-provider Git support (GitHub, GitLab, Bitbucket, Azure DevOps)")
		fmt.Println("  - RAG-powered semantic code search and analysis")
		fmt.Println("  - Developer workflow automation (GitFlow, pre-commit, PR creation)")
		fmt.Println("  - DORA metrics and developer velocity tracking")
		fmt.Println("  - AI-powered code analysis and suggestions")
		fmt.Println("  - HashiCorp Vault secrets management")
		fmt.Println("  - Cache management and monitoring")
		fmt.Println()
		fmt.Println("Commands:")
		fmt.Println("  auth       - OAuth device flow authentication")
		fmt.Println("  git        - Multi-provider Git operations")
		fmt.Println("  github     - GitHub-specific operations")
		fmt.Println("  rag        - RAG semantic search and embeddings")
		fmt.Println("  workflow   - Developer workflow automation")
		fmt.Println("  dora       - DORA metrics and analytics")
		fmt.Println("  ai         - AI-powered code analysis")
		fmt.Println("  vault      - Secrets management")
		fmt.Println("  cache      - Cache operations")
		fmt.Println("  health     - System health checks")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
