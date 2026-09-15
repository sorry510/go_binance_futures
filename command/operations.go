package command

import (
	"context"
	"fmt"
	"io"
	"time"

	"go_binance_futures/appversion"
	"go_binance_futures/service/logcleanup"
	"go_binance_futures/service/systemhealth"
	"go_binance_futures/utils"
)

func Doctor(ctx context.Context, out io.Writer) (systemhealth.Report, error) {
	report, err := (systemhealth.Service{}).Report(ctx, systemhealth.Options{CheckBinanceREST: true})
	if err != nil {
		return report, err
	}
	fmt.Fprintf(out, "System doctor (%s)\n", time.UnixMilli(report.GeneratedAt).Format(time.RFC3339))
	printDoctorCheck(out, "Database", report.Database.Check)
	printDoctorCheck(out, "Binance REST", report.BinanceREST)
	printDoctorCheck(out, "Futures WS", report.FuturesWS)
	printDoctorCheck(out, "Announcement WS", report.AnnouncementWS)
	printDoctorCheck(out, "Market Intelligence", report.MarketIntelligence)
	printDoctorCheck(out, "MCP", report.MCP)
	printDoctorCheck(out, "LLM", report.LLM)
	printDoctorCheck(out, "Scheduler", report.Scheduler)
	printDoctorCheck(out, "Agent Runtime", report.Agent.Check)
	printDoctorCheck(out, "Trade Safety", report.Trade.Check)
	fmt.Fprintf(out, "Overall: %s\n", report.Overall)
	return report, nil
}

func DoctorExitCode(report systemhealth.Report) int {
	if report.Overall == systemhealth.StatusError {
		return 1
	}
	return 0
}

func printDoctorCheck(out io.Writer, name string, check systemhealth.Check) {
	marker := "OK"
	switch check.Status {
	case systemhealth.StatusError:
		marker = "ERROR"
	case systemhealth.StatusWarning, systemhealth.StatusUnknown:
		marker = "WARN"
	case systemhealth.StatusDisabled:
		marker = "OFF"
	}
	message := check.Message
	if check.LastError != "" {
		if message != "" {
			message += ": "
		}
		message += check.LastError
	}
	fmt.Fprintf(out, "[%s] %-20s %s\n", marker, name, message)
}

func CleanupLogs(ctx context.Context, beforeDays int, now time.Time, out io.Writer) error {
	if beforeDays <= 0 {
		return fmt.Errorf("--before-days must be greater than 0")
	}
	cfg, err := utils.GetSystemConfig()
	if err != nil {
		return fmt.Errorf("load database schema version: %w", err)
	}
	if cfg.Version < appversion.DatabaseSchemaVersion {
		return fmt.Errorf("database version %d is older than required version %d; run `go_binance_futures sync db` first", cfg.Version, appversion.DatabaseSchemaVersion)
	}
	if now.IsZero() {
		now = time.Now()
	}
	cutoff := now.Add(-time.Duration(beforeDays) * 24 * time.Hour).UnixMilli()
	results, err := logcleanup.Cleanup(ctx, cutoff)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "cleanup logs before: %s (%d days)\n", time.UnixMilli(cutoff).Format(time.RFC3339), beforeDays)
	var total int64
	for _, result := range results {
		fmt.Fprintf(out, "%-28s deleted %d\n", result.Table, result.Deleted)
		total += result.Deleted
	}
	fmt.Fprintf(out, "total deleted: %d\n", total)
	return nil
}
