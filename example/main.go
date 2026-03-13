// Command example demonstrates how to integrate cobragenskill into a Cobra CLI.
//
// Run:
//
//	go run ./example gen-skill --dry-run --no-ai
//	go run ./example gen-skill --project --no-ai
//	go run ./example gen-skill --global           # uses AI if available
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	cobragenskill "github.com/theburrowhub/go-cobra-gen-skill"
)

func main() {
	root := &cobra.Command{
		Use:   "acme",
		Short: "Acme deployment CLI",
		Long: `acme is a CLI tool for managing deployments, secrets, and infrastructure
for the Acme platform. It supports multiple environments and integrates
with the Acme API.`,
	}

	// deploy command
	deployCmd := &cobra.Command{
		Use:   "deploy [environment]",
		Short: "Deploy the application to an environment",
		Long: `Deploy the application to the specified environment.
Environments: staging, production, preview.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Deploying to %s...\n", args[0])
			return nil
		},
	}
	deployCmd.Flags().StringP("image", "i", "", "Docker image tag to deploy")
	deployCmd.Flags().BoolP("dry-run", "n", false, "Simulate deployment without applying changes")

	// secrets command group
	secretsCmd := &cobra.Command{
		Use:   "secrets",
		Short: "Manage environment secrets",
	}
	secretsCmd.AddCommand(
		&cobra.Command{
			Use:   "list [environment]",
			Short: "List all secrets for an environment",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Printf("Secrets for %s: (none)\n", args[0])
				return nil
			},
		},
		&cobra.Command{
			Use:   "set [environment] [key] [value]",
			Short: "Set a secret value",
			Args:  cobra.ExactArgs(3),
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Printf("Set %s in %s\n", args[1], args[0])
				return nil
			},
		},
		&cobra.Command{
			Use:   "delete [environment] [key]",
			Short: "Delete a secret",
			Args:  cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Printf("Deleted %s from %s\n", args[1], args[0])
				return nil
			},
		},
	)

	// status command
	statusCmd := &cobra.Command{
		Use:   "status [environment]",
		Short: "Show the current deployment status",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			env := "production"
			if len(args) > 0 {
				env = args[0]
			}
			fmt.Printf("Status for %s: healthy\n", env)
			return nil
		},
	}

	root.AddCommand(deployCmd, secretsCmd, statusCmd)

	// Register the gen-skill command with project metadata.
	cobragenskill.RegisterCommand(root,
		cobragenskill.WithVersion("1.0.0"),
		cobragenskill.WithLicense("MIT"),
		cobragenskill.WithMetadata("author", "acme-org"),
	)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
