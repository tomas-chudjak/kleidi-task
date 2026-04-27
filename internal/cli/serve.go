package cli

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/ahoylog/kvik-tasks/internal/api"
	"github.com/ahoylog/kvik-tasks/internal/core"
	"github.com/ahoylog/kvik-tasks/internal/db"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start HTTP server (REST API + UI)",
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetInt("port")
		addr := fmt.Sprintf("127.0.0.1:%d", port)

		manager, err := db.NewManager()
		if err != nil {
			return fmt.Errorf("initializing database: %w", err)
		}
		defer manager.Close()

		projectService := core.NewProjectService(manager)
		router := api.NewRouter(projectService)
		server := api.NewServer(addr, router)

		// Graceful shutdown on SIGINT/SIGTERM
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			slog.Info("starting server", "addr", addr, "url", fmt.Sprintf("http://%s", addr))
			if err := server.ListenAndServe(); err != nil {
				slog.Error("server error", "err", err)
			}
		}()

		<-stop
		slog.Info("shutting down")
		return server.Close()
	},
}

func init() {
	serveCmd.Flags().Int("port", 7842, "Port to listen on")
	rootCmd.AddCommand(serveCmd)
}
