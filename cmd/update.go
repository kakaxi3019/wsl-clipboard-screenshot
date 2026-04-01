package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	ver "github.com/kakaxi3019/wsl-clipboard-screenshot/internal/version"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Self-update to the latest version",
	RunE: func(cmd *cobra.Command, args []string) error {
		currentVersion := rootCmd.Version
		if currentVersion == "dev" || currentVersion == "" {
			fmt.Println("Development build - cannot self-update. Install a release version first.")
			return nil
		}

		latest, err := ver.CheckForUpdate(currentVersion)
		if err != nil {
			return fmt.Errorf("failed to check for updates: %w", err)
		}

		if latest == "" {
			fmt.Printf("Already at version %s\n", currentVersion)
			return nil
		}

		fmt.Printf("Updating from %s to %s...\n", currentVersion, latest)

		// Download and run install script
		installURL := fmt.Sprintf("https://raw.githubusercontent.com/kakaxi3019/wsl-clipboard-screenshot/main/scripts/install.sh")
		downloadCmd := fmt.Sprintf(`curl -fsSL "%s" | bash -s -- --version %s`, installURL, latest)

		runCmd := exec.Command("sh", "-c", downloadCmd)
		runCmd.Stdout = os.Stdout
		runCmd.Stderr = os.Stderr
		runCmd.Stdin = os.Stdin

		return runCmd.Run()
	},
}
