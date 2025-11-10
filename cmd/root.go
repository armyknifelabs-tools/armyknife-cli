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

	rootCmd.PersistentFlags().StringVar(&apiURL, "api-url", "https://test.armyknifelabs.com/api/v1", "API base URL")
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.armyknife/config.json)")
}

func initConfig() {
	// Config initialization is handled in the config package
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("ArmyKnife CLI v0.1.0")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
