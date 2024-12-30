package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/percona/percona-xtradb-cluster-operator/cmd/pitr/collector"
	"github.com/percona/percona-xtradb-cluster-operator/cmd/pitr/recoverer"

	"github.com/caarlos0/env"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	command := "collect"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()

	srv := &http.Server{Addr: ":8080"}
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.HandleFunc("/health", healthHandler)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("ERROR: HTTP server error: %v", err)
		}
	}()

	go func() {
		<-ctx.Done()
		if err := srv.Shutdown(context.Background()); err != nil {
			log.Printf("ERROR: HTTP server shutdown: %v", err)
		}
	}()

	switch command {
	case "collect":
		runCollector(ctx)
	case "recover":
		runRecoverer(ctx)
	default:
		fmt.Fprintf(os.Stderr, "ERROR: unknown command \"%s\".\nCommands:\n  collect - collect binlogs\n  recover - recover from binlogs\n", command)
		os.Exit(1)
	}
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("ok")); err != nil {
		log.Println("ERROR: writing health response:", err)
	}
}

func runCollector(ctx context.Context) {
	config, err := getCollectorConfig()
	if err != nil {
		log.Fatalln("ERROR: get config:", err)
	}
	c, err := collector.New(ctx, config)
	if err != nil {
		log.Fatalln("ERROR: new controller:", err)
	}

	cleanupCtx, cleanupCancel := context.WithCancel(ctx)
	defer cleanupCancel()
	go runCleanup(cleanupCtx, c, config)

	log.Println("run binlog collector")
	for {
		timeout, cancel := context.WithTimeout(ctx, time.Duration(config.CollectSpanSec)*time.Second)
		defer cancel()

		err := c.Run(timeout)
		if err != nil {
			log.Fatalln("ERROR:", err)
		}

		t := time.NewTimer(time.Duration(config.CollectSpanSec) * time.Second)
		select {
		case <-ctx.Done():
			log.Fatalln("ERROR:", ctx.Err().Error())
		case <-t.C:
			break
		}
	}
}

func runCleanup(ctx context.Context, c *collector.Collector, config collector.Config) {
	var retentionDays, cleanupIntervalMin int
	switch config.StorageType {
	case "s3":
		retentionDays = config.BackupStorageS3.RetentionDays
		cleanupIntervalMin = config.BackupStorageS3.CleanupIntervalMin
	case "azure":
		retentionDays = config.BackupStorageAzure.RetentionDays
		cleanupIntervalMin = config.BackupStorageAzure.CleanupIntervalMin
	default:
		log.Printf("WARNING: unknown storage type for cleanup: %s", config.StorageType)
		return
	}

	ticker := time.NewTicker(time.Duration(cleanupIntervalMin) * time.Minute)
	defer ticker.Stop()

	// Run first cleanup immediately
	if err := c.CleanupOldBinlogs(ctx, retentionDays); err != nil {
		log.Printf("WARNING: Failed to cleanup old binlogs: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			log.Printf("Starting scheduled binlog cleanup (retention: %d days, interval: %d minutes)",
				retentionDays, cleanupIntervalMin)
			if err := c.CleanupOldBinlogs(ctx, retentionDays); err != nil {
				log.Printf("WARNING: Failed to cleanup old binlogs: %v", err)
			}
		}
	}
}

func runRecoverer(ctx context.Context) {
	config, err := getRecovererConfig()
	if err != nil {
		log.Fatalln("ERROR: get recoverer config:", err)
	}
	c, err := recoverer.New(ctx, config)
	if err != nil {
		log.Fatalln("ERROR: new recoverer controller:", err)
	}
	log.Println("run recover")
	err = c.Run(ctx)
	if err != nil {
		log.Fatalln("ERROR: recover:", err)
	}
}

func getCollectorConfig() (collector.Config, error) {
	cfg := collector.Config{}
	err := env.Parse(&cfg)
	switch cfg.StorageType {
	case "s3":
		if err := env.Parse(&cfg.BackupStorageS3); err != nil {
			return cfg, err
		}
	case "azure":
		if err := env.Parse(&cfg.BackupStorageAzure); err != nil {
			return cfg, err
		}
	default:
		return cfg, errors.New("unknown STORAGE_TYPE")
	}

	return cfg, err
}

func getRecovererConfig() (recoverer.Config, error) {
	cfg := recoverer.Config{}
	if err := env.Parse(&cfg); err != nil {
		return cfg, err
	}
	switch cfg.StorageType {
	case "s3":
		if err := env.Parse(&cfg.BackupStorageS3); err != nil {
			return cfg, err
		}
		if err := env.Parse(&cfg.BinlogStorageS3); err != nil {
			return cfg, err
		}
	case "azure":
		if err := env.Parse(&cfg.BackupStorageAzure); err != nil {
			return cfg, err
		}
		if err := env.Parse(&cfg.BinlogStorageAzure); err != nil {
			return cfg, err
		}
	default:
		return cfg, errors.New("unknown STORAGE_TYPE")
	}

	return cfg, nil
}
