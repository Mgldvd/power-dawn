package main

import (
	"context"
	"fmt"
	"time"
)

// TestWakeResult holds the outcome of a wake-alarm hardware test.
type TestWakeResult struct {
	WakeEpoch int64
	WakeTime  string
	RTCSet    bool
	Error     string
	Warnings  []string
}

// PerformTestWake sets the RTC wake alarm 60 seconds from now, then
// immediately shuts down the machine. If the alarm cannot be set it returns
// without shutting down so the error can be shown in the UI.
// The binary must be running as root (enforced by main).
func PerformTestWake() TestWakeResult {
	log := logOrDiscard()
	result := TestWakeResult{}

	wakeEpoch := time.Now().Add(60 * time.Second).Unix()
	result.WakeEpoch = wakeEpoch
	result.WakeTime = time.Unix(wakeEpoch, 0).Format("15:04:05")
	log.Info("test wake initiated", "wake_epoch", wakeEpoch, "wake_at", result.WakeTime)

	// Try rtcwake with a tight per-command timeout.
	rtcCtx, rtcCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer rtcCancel()
	rtcOut, rtcErr := runCommand(rtcCtx, "rtcwake", "-m", "no", "-t",
		fmt.Sprintf("%d", wakeEpoch))
	if rtcErr == nil {
		log.Info("RTC alarm set via rtcwake", "epoch", wakeEpoch)
		result.RTCSet = true
	} else {
		log.Warn("rtcwake failed, trying sysfs fallback", "err", rtcErr, "output", rtcOut)
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("rtcwake failed (%v): %s", rtcErr, rtcOut))

		clearCtx, clearCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer clearCancel()
		_, clearErr := runCommand(clearCtx, "sh", "-c",
			fmt.Sprintf("printf '0\\n' > %s", rtcPath))
		if clearErr != nil {
			log.Warn("RTC disarm failed", "err", clearErr)
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("RTC disarm failed: %v", clearErr))
		}

		writeCtx, writeCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer writeCancel()
		_, writeErr := runCommand(writeCtx, "sh", "-c",
			fmt.Sprintf("printf '%d\\n' > %s", wakeEpoch, rtcPath))
		if writeErr != nil {
			log.Error("RTC alarm could not be set", "err", writeErr)
			result.Error = fmt.Sprintf(
				"RTC alarm could not be set — this hardware or VM does not support "+
					"wake-on-alarm.\n  Details: %v", writeErr)
			return result
		}
		log.Info("RTC alarm set via sysfs", "epoch", wakeEpoch)
		result.RTCSet = true
	}

	// RTC alarm is armed — shut down immediately.
	log.Info("initiating shutdown for wake test")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutCancel()
	_, shutErr := runCommand(shutCtx, "shutdown", "-h", "now",
		"Power-Dawn: RTC test — will wake in 60 s")
	if shutErr != nil {
		log.Error("shutdown command failed", "err", shutErr)
		result.Error = fmt.Sprintf("shutdown command failed: %v\n"+
			"  RTC alarm was set but the machine did not shut down.", shutErr)
	}
	return result
}
