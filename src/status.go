package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

type StatusReport struct {
	ScriptExists     bool
	LogExists        bool
	CronInstalled    bool
	CronReadable     bool
	RTCAvailable     bool
	RTCPathExists    bool
	RTCWritable      bool
	RTCStatus        string
	RTCHardwareNote  string
	WakealarmValue   string
	ShutdownTime     string
	WakeTime         string
	Installed        bool
	ScriptPath       string
	LogPath          string
	CronPreview      string
	Errors           []string
	PermissionNotice string
}

type LogView struct {
	// Cron script log (/var/log/wakealarm-ensure.log)
	Exists   bool
	Path     string
	Content  string
	Message  string
	Warnings []string
	// App log (/var/log/power-dawn.log)
	AppLogExists bool
	AppLogPath   string
	AppLog       string
	AppLogMsg    string
}

func LoadStatus() StatusReport {
	report := StatusReport{
		ScriptPath: scriptPath,
		LogPath:    logPath,
	}

	if info, err := os.Stat(scriptPath); err == nil && !info.IsDir() {
		report.ScriptExists = true
		content, readErr := os.ReadFile(scriptPath)
		if readErr != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("unable to read %s: %v", scriptPath, readErr))
		} else {
			report.WakeTime = parseWakeTimeFromScript(string(content))
		}
	}

	if info, err := os.Stat(logPath); err == nil && !info.IsDir() {
		report.LogExists = true
	}

	if info, err := os.Stat(rtcPath); err == nil && !info.IsDir() {
		report.RTCPathExists = true
		report.RTCAvailable = true
		report.RTCStatus = "available"

		if value, readErr := readWakealarmValue(); readErr != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("unable to read %s: %v", rtcPath, readErr))
		} else {
			report.WakealarmValue = value
		}

		// Test whether the kernel actually accepts alarm writes.
		// We write 0 (disarm) which is harmless; failure means the
		// hardware/hypervisor doesn't support RTC alarms (e.g. VirtualBox).
		writeCtx, writeCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer writeCancel()
		_, writeErr := runCommand(writeCtx, "sudo", "-n", "sh", "-c",
			fmt.Sprintf("printf '0\\n' > %s", rtcPath))
		if writeErr == nil {
			report.RTCWritable = true
			report.RTCStatus = "available and writable"
		} else {
			report.RTCStatus = "available but not writable"
			report.RTCHardwareNote = "RTC alarm writes fail — this machine (or VM) does not support hardware wake-on-alarm. " +
				"Automatic power-on requires bare-metal hardware with ACPI/RTC alarm support."
		}
	} else if os.IsNotExist(err) {
		report.RTCStatus = "not detected"
		report.RTCHardwareNote = "No RTC device found at " + rtcPath
	} else if err != nil {
		report.RTCStatus = "unavailable"
		report.Errors = append(report.Errors, fmt.Sprintf("unable to inspect %s: %v", rtcPath, err))
	} else {
		report.RTCStatus = "unavailable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cronOutput, err := runCommand(ctx, "sudo", "-n", "crontab", "-l")
	if err != nil {
		if isSudoPasswordError(cronOutput) {
			report.PermissionNotice = "root crontab could not be read without sudo authentication"
		} else if isNoCrontabOutput(cronOutput) {
			report.CronReadable = true
		} else {
			report.Errors = append(report.Errors, fmt.Sprintf("unable to inspect root crontab: %v", err))
		}
	} else {
		report.CronReadable = true
		report.CronInstalled = hasManagedCronBlock(cronOutput)
		report.CronPreview = strings.TrimSpace(extractManagedCronBlock(cronOutput))
		report.ShutdownTime = parseShutdownTimeFromCron(cronOutput)
	}

	report.Installed = report.ScriptExists && report.LogExists && report.CronInstalled && report.WakeTime != "" && report.ShutdownTime != "" && report.RTCAvailable && report.RTCWritable
	return report
}

func LoadLogs(limit int) LogView {
	view := LogView{
		Path:       logPath,
		AppLogPath: appLogPath,
	}

	// --- cron script log ---
	data, err := os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			view.Message = "No cron script log found yet. Install the schedule first to create it."
		} else {
			view.Warnings = append(view.Warnings, fmt.Sprintf("unable to read %s: %v", logPath, err))
			view.Message = "The cron script log file exists but could not be read."
		}
	} else {
		view.Exists = true
		trimmed := strings.TrimSpace(string(data))
		if trimmed == "" {
			view.Message = "The cron script log exists but is currently empty."
		} else {
			view.Content = tailLines(trimmed, limit)
		}
	}

	// --- app log ---
	appData, appErr := os.ReadFile(appLogPath)
	if appErr != nil {
		if os.IsNotExist(appErr) {
			view.AppLogMsg = "No app log entries yet."
		} else {
			view.AppLogMsg = fmt.Sprintf("unable to read %s: %v", appLogPath, appErr)
		}
	} else {
		view.AppLogExists = true
		trimmed := strings.TrimSpace(string(appData))
		if trimmed == "" {
			view.AppLogMsg = "App log exists but is currently empty."
		} else {
			view.AppLog = tailLines(trimmed, limit)
		}
	}

	return view
}
