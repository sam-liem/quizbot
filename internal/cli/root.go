package cli

import (
	"os"

	"github.com/sam-liem/quizbot/internal/model"
	"github.com/spf13/cobra"
)

// NewRootCommand creates the root cobra command and registers all subcommands.
func NewRootCommand(app *App) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "quizbot",
		Short: "Adaptive quiz preparation tool",
	}

	rootCmd.AddCommand(newQuizCommand(app))
	rootCmd.AddCommand(newStatsCommand(app))
	rootCmd.AddCommand(newImportCommand(app))
	rootCmd.AddCommand(newPacksCommand(app))
	rootCmd.AddCommand(newConfigCommand(app))

	return rootCmd
}

func newQuizCommand(app *App) *cobra.Command {
	quizCmd := &cobra.Command{
		Use:   "quiz",
		Short: "Quiz management commands",
	}

	// quiz start
	var packID, topicID string
	var count int
	var mock bool

	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start a new quiz session",
		RunE: func(cmd *cobra.Command, args []string) error {
			mode := model.SessionModePractice
			if mock {
				mode = model.SessionModeMock
			}
			return app.RunQuizStart(packID, mode, count, topicID, os.Stdin, os.Stdout)
		},
	}
	startCmd.Flags().StringVar(&packID, "pack", "", "Quiz pack ID")
	startCmd.Flags().StringVar(&topicID, "topic", "", "Topic ID to filter by")
	startCmd.Flags().IntVar(&count, "count", 0, "Number of questions")
	startCmd.Flags().BoolVar(&mock, "mock", false, "Use mock exam mode")
	_ = startCmd.MarkFlagRequired("pack")

	// quiz resume
	resumeCmd := &cobra.Command{
		Use:   "resume",
		Short: "Resume an in-progress quiz session",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.RunQuizResume(os.Stdin, os.Stdout)
		},
	}

	quizCmd.AddCommand(startCmd, resumeCmd)
	return quizCmd
}

func newStatsCommand(app *App) *cobra.Command {
	var packID, topicID string
	var detailed bool

	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show quiz statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.RunStats(packID, topicID, detailed, os.Stdout)
		},
	}
	cmd.Flags().StringVar(&packID, "pack", "", "Quiz pack ID")
	cmd.Flags().StringVar(&topicID, "topic", "", "Topic ID to filter by")
	cmd.Flags().BoolVar(&detailed, "detailed", false, "Show detailed stats")
	_ = cmd.MarkFlagRequired("pack")

	return cmd
}

func newImportCommand(app *App) *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "import [file]",
		Short: "Import a quiz pack file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.RunImport(args[0], format, os.Stdout)
		},
	}
	cmd.Flags().StringVar(&format, "format", "", "File format (yaml, json, markdown). Auto-detected if omitted.")

	return cmd
}

func newPacksCommand(app *App) *cobra.Command {
	packsCmd := &cobra.Command{
		Use:   "packs",
		Short: "Pack management commands",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all quiz packs",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.RunPacksList(os.Stdout)
		},
	}

	activateCmd := &cobra.Command{
		Use:   "activate [pack-id]",
		Short: "Activate a quiz pack",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.RunPacksActivate(args[0], os.Stdout)
		},
	}

	deactivateCmd := &cobra.Command{
		Use:   "deactivate [pack-id]",
		Short: "Deactivate a quiz pack",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.RunPacksDeactivate(args[0], os.Stdout)
		},
	}

	packsCmd.AddCommand(listCmd, activateCmd, deactivateCmd)
	return packsCmd
}

func newConfigCommand(app *App) *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration management commands",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all configuration values",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.RunConfigList(os.Stdout)
		},
	}

	getCmd := &cobra.Command{
		Use:   "get [key]",
		Short: "Get a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.RunConfigGet(args[0], os.Stdout)
		},
	}

	setCmd := &cobra.Command{
		Use:   "set [key] [value]",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.RunConfigSet(args[0], args[1], os.Stdout)
		},
	}

	configCmd.AddCommand(listCmd, getCmd, setCmd)
	return configCmd
}
