package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

type ScheduleConfig struct {
	ShutdownTime string
	WakeTime     string
}

type InstallResult struct {
	ShutdownTime     string
	WakeTime         string
	ScriptCreated    bool
	LogPrepared      bool
	CronInstalled    bool
	CurrentWakealarm string
	RecentLog        string
	CronPreview      string
	Success          bool
	FatalError       string
	Warnings         []string
}

func PerformInstall(cfg ScheduleConfig) InstallResult {
	log := logOrDiscard()
	log.Info("install started", "shutdown", cfg.ShutdownTime, "wake", cfg.WakeTime)

	result := InstallResult{
		ShutdownTime: cfg.ShutdownTime,
		WakeTime:     cfg.WakeTime,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	tempScript, err := writeTempScript(cfg.WakeTime)
	if err != nil {
		log.Error("failed to prepare helper script", "err", err)
		result.FatalError = fmt.Sprintf("unable to prepare helper script: %v", err)
		return result
	}
	defer os.Remove(tempScript)

	if _, err := runCommand(ctx, "sudo", "install", "-m", "0755", tempScript, scriptPath); err != nil {
		log.Error("failed to install script", "path", scriptPath, "err", err)
		result.FatalError = fmt.Sprintf("failed to install %s: %v", scriptPath, err)
		return result
	}
	if _, err := runCommand(ctx, "sudo", "chmod", "0755", scriptPath); err != nil {
		log.Error("failed to chmod script", "path", scriptPath, "err", err)
		result.FatalError = fmt.Sprintf("failed to set permissions on %s: %v", scriptPath, err)
		return result
	}
	log.Info("script installed", "path", scriptPath)
	result.ScriptCreated = true

	if _, err := runCommand(ctx, "sudo", "touch", logPath); err != nil {
		log.Error("failed to create log file", "path", logPath, "err", err)
		result.FatalError = fmt.Sprintf("failed to create %s: %v", logPath, err)
		return result
	}
	if _, err := runCommand(ctx, "sudo", "chmod", "0644", logPath); err != nil {
		log.Error("failed to chmod log file", "path", logPath, "err", err)
		result.FatalError = fmt.Sprintf("failed to set permissions on %s: %v", logPath, err)
		return result
	}
	log.Info("log file prepared", "path", logPath)
	result.LogPrepared = true

	cronPreview, err := installManagedCron(ctx, cfg.ShutdownTime)
	if err != nil {
		log.Error("failed to install cron block", "err", err)
		result.FatalError = fmt.Sprintf("failed to install root cron block: %v", err)
		return result
	}
	log.Info("cron block installed", "shutdown", cfg.ShutdownTime)
	result.CronInstalled = true
	result.CronPreview = cronPreview

	if _, err := runCommand(ctx, "sudo", scriptPath); err != nil {
		log.Warn("initial wakealarm run failed", "err", err)
		result.Warnings = append(result.Warnings, fmt.Sprintf("initial wakealarm run failed: %v", err))
	}

	if value, err := readWakealarmValue(); err != nil {
		log.Warn("unable to read wakealarm value", "path", rtcPath, "err", err)
		result.Warnings = append(result.Warnings, fmt.Sprintf("unable to read %s: %v", rtcPath, err))
	} else {
		log.Debug("current wakealarm", "value", value)
		result.CurrentWakealarm = value
	}

	if logOutput, err := runCommand(ctx, "sudo", "tail", "-n", "50", logPath); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("unable to read recent logs: %v", err))
	} else {
		result.RecentLog = strings.TrimSpace(logOutput)
	}

	if cronOutput, err := runCommand(ctx, "sudo", "crontab", "-l"); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("unable to verify root crontab: %v", err))
	} else {
		result.CronPreview = strings.TrimSpace(extractManagedCronBlock(cronOutput))
	}

	result.Success = result.ScriptCreated && result.LogPrepared && result.CronInstalled && result.FatalError == ""
	if result.Success {
		log.Info("install complete", "shutdown", cfg.ShutdownTime, "wake", cfg.WakeTime)
	} else {
		log.Error("install finished with errors")
	}
	return result
}

func writeTempScript(wakeTime string) (string, error) {
	normalized, err := normalizeTime(wakeTime)
	if err != nil {
		return "", err
	}
	script, err := buildWakealarmScript(normalized)
	if err != nil {
		return "", err
	}

	file, err := os.CreateTemp("", "wakealarm-ensure-*.sh")
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := file.WriteString(script); err != nil {
		return "", err
	}
	if err := file.Chmod(0o755); err != nil {
		return "", err
	}
	return file.Name(), nil
}

func installManagedCron(ctx context.Context, shutdownTime string) (string, error) {
	existing, err := runCommand(ctx, "sudo", "crontab", "-l")
	if err != nil && !isNoCrontabOutput(existing) {
		return "", err
	}
	if isNoCrontabOutput(existing) {
		existing = ""
	}

	cleaned := removeManagedCronBlock(existing)
	block, err := buildManagedCronBlock(shutdownTime)
	if err != nil {
		return "", err
	}

	finalCrontab := strings.TrimSpace(cleaned)
	if finalCrontab != "" {
		finalCrontab += "\n\n"
	}
	finalCrontab += block + "\n"

	if _, err := runCommandWithInput(ctx, finalCrontab, "sudo", "crontab", "-"); err != nil {
		return "", err
	}

	verified, err := runCommand(ctx, "sudo", "crontab", "-l")
	if err != nil {
		return block, err
	}
	return strings.TrimSpace(extractManagedCronBlock(verified)), nil
}
