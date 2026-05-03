package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tomas-chudjak/kleidi-task/internal/api"
	"github.com/tomas-chudjak/kleidi-task/internal/config"
	"github.com/tomas-chudjak/kleidi-task/internal/core"
	"github.com/tomas-chudjak/kleidi-task/internal/db"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start HTTP server (REST API + UI)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.LoadGlobal()
		port := cfg.Port
		if cmd.Flags().Changed("port") {
			port, _ = cmd.Flags().GetInt("port")
		}
		host := "127.0.0.1"
		if cmd.Flags().Changed("host") {
			host, _ = cmd.Flags().GetString("host")
		}
		addr := fmt.Sprintf("%s:%d", host, port)

		manager, err := db.NewManager()
		if err != nil {
			return fmt.Errorf("initializing database: %w", err)
		}
		defer manager.Close()

		projectService := core.NewProjectService(manager)
		router := api.NewRouter(projectService)
		server := api.NewServer(addr, router)

		// Start auto-archive background routine
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go runAutoArchive(ctx, projectService)

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
		cancel()
		return server.Close()
	},
}

// runAutoArchive periodically archives completed tasks older than N days per project config.
func runAutoArchive(ctx context.Context, projectService *core.ProjectService) {
	// Run once at startup, then every 24h
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	autoArchiveAll(ctx, projectService)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			autoArchiveAll(ctx, projectService)
		}
	}
}

func autoArchiveAll(ctx context.Context, projectService *core.ProjectService) {
	projects, err := projectService.List()
	if err != nil {
		return
	}
	for _, p := range projects {
		configService, err := projectService.ConfigServiceFor(p.Path)
		if err != nil {
			continue
		}
		cfg, err := configService.Get(ctx)
		if err != nil || cfg.AutoArchiveDays <= 0 {
			continue
		}
		taskService, err := projectService.TaskServiceFor(p.Path)
		if err != nil {
			continue
		}
		cutoff := time.Now().AddDate(0, 0, -int(cfg.AutoArchiveDays))
		count, err := taskService.ArchiveCompletedBefore(ctx, cutoff)
		if err == nil && count > 0 {
			slog.Info("auto-archived tasks", "project", p.Slug, "count", count, "older_than_days", cfg.AutoArchiveDays)
		}
	}
}

func init() {
	serveCmd.Flags().String("host", "127.0.0.1", "Host to bind to (use 0.0.0.0 for Docker)")
	serveCmd.Flags().Int("port", 7842, "Port to listen on")
	rootCmd.AddCommand(serveCmd)
}
