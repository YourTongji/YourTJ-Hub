package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/leancodebox/GooseForum/app/models/forum/pageConfig"
	"github.com/leancodebox/GooseForum/app/service/filemigrateservice"
	"github.com/leancodebox/GooseForum/app/service/storageservice"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "migrate-files",
		Short: "Migrate SQLite BLOB files to the configured object storage",
		RunE:  runMigrateFiles,
	}
	cmd.Flags().String("provider", "", "Storage provider override: local|s3 (default: use storage settings)")
	cmd.Flags().String("endpoint", "", "S3 endpoint override")
	cmd.Flags().String("bucket", "", "S3 bucket override")
	cmd.Flags().String("region", "", "S3 region override")
	cmd.Flags().String("bucket-lookup", "auto", "S3 bucket lookup: auto|dns|path")
	cmd.Flags().Bool("secure", true, "Use HTTPS for S3 endpoint")
	cmd.Flags().String("access-key", "", "S3 access key")
	cmd.Flags().String("secret-key", "", "S3 secret key")
	cmd.Flags().Bool("clear-after-migrate", false, "Clear BLOB column after a successful upload")
	appendCommand(cmd)
}

func runMigrateFiles(cmd *cobra.Command, _ []string) error {
	cfg := storageservice.GetStorageSettings()

	if provider, _ := cmd.Flags().GetString("provider"); provider != "" {
		cfg.Provider = provider
	}
	applyFlagOverrides(cmd, &cfg)

	if cfg.Provider != storageservice.ProviderS3 {
		return fmt.Errorf("migrate-files requires provider=s3 (current: %s)", cfg.Provider)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := storageservice.TestConnection(ctx, cfg); err != nil {
		return fmt.Errorf("storage connection test failed: %w", err)
	}

	clearAfter, _ := cmd.Flags().GetBool("clear-after-migrate")
	start := time.Now()
	processed, failed, err := filemigrateservice.MigrateFiles(ctx, 0, clearAfter, func(lastID uint64, proc, fail int64) {
		fmt.Printf("migrated %d files (failed %d), last id %d\n", proc, fail, lastID)
	})
	fmt.Printf("migration finished: processed=%d failed=%d duration=%s\n",
		processed, failed, time.Since(start).Round(time.Millisecond))
	return err
}

func applyFlagOverrides(cmd *cobra.Command, cfg *pageConfig.StorageSettings) {
	if v, _ := cmd.Flags().GetString("endpoint"); v != "" {
		cfg.Endpoint = v
	}
	if v, _ := cmd.Flags().GetString("bucket"); v != "" {
		cfg.Bucket = v
	}
	if v, _ := cmd.Flags().GetString("region"); v != "" {
		cfg.Region = v
	}
	if v, _ := cmd.Flags().GetString("bucket-lookup"); v != "" {
		cfg.BucketLookup = v
	}
	if v, _ := cmd.Flags().GetString("access-key"); v != "" {
		cfg.AccessKey = v
	}
	if v, _ := cmd.Flags().GetString("secret-key"); v != "" {
		cfg.SecretKey = v
	}
	if cmd.Flags().Changed("secure") {
		cfg.Secure, _ = cmd.Flags().GetBool("secure")
	}
}
