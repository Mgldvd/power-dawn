package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	rtcPath    = "/sys/class/rtc/rtc0/wakealarm"
	scriptPath = "/usr/local/sbin/wakealarm-ensure.sh"
	logPath    = "/var/log/wakealarm-ensure.log"
	cronBegin  = "# BEGIN power-schedule"
	cronEnd    = "# END power-schedule"
)

var hhmmPattern = regexp.MustCompile(`^([01]\d|2[0-3]):([0-5]\d)$`)

func normalizeTime(value string) (string, error) {
	value = strings.TrimSpace(value)
	if !hhmmPattern.MatchString(value) {
		return "", fmt.Errorf("time must be in HH:MM 24-hour format")
	}
	t, err := time.Parse("15:04", value)
	if err != nil {
		return "", fmt.Errorf("invalid time %q", value)
	}
	return t.Format("15:04"), nil
}

func cronFieldsForTime(value string) (minute int, hour int, err error) {
	normalized, err := normalizeTime(value)
	if err != nil {
		return 0, 0, err
	}
	parts := strings.Split(normalized, ":")
	hour, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid hour: %w", err)
	}
	minute, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid minute: %w", err)
	}
	return minute, hour, nil
}

func removeManagedCronBlock(existing string) string {
	var kept []string
	insideBlock := false

	for _, line := range strings.Split(strings.ReplaceAll(existing, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		switch trimmed {
		case cronBegin:
			insideBlock = true
			continue
		case cronEnd:
			insideBlock = false
			continue
		}
		if insideBlock {
			continue
		}
		kept = append(kept, line)
	}

	return strings.Trim(strings.Join(kept, "\n"), "\n")
}

func extractManagedCronBlock(crontab string) string {
	lines := strings.Split(strings.ReplaceAll(crontab, "\r\n", "\n"), "\n")
	start := -1
	end := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == cronBegin {
			start = i
		}
		if trimmed == cronEnd && start >= 0 {
			end = i
			break
		}
	}
	if start >= 0 && end >= start {
		return strings.Join(lines[start:end+1], "\n")
	}
	return ""
}

func hasManagedCronBlock(crontab string) bool {
	return extractManagedCronBlock(crontab) != ""
}

func parseWakeTimeFromScript(content string) string {
	matcher := regexp.MustCompile(`(?m)^WAKE_TIME="?([0-2]\d:[0-5]\d)"?$`)
	matches := matcher.FindStringSubmatch(content)
	if len(matches) == 2 {
		return matches[1]
	}
	return ""
}

func parseShutdownTimeFromCron(crontab string) string {
	for _, line := range strings.Split(strings.ReplaceAll(crontab, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) < 6 {
			continue
		}
		if !strings.Contains(trimmed, "/sbin/shutdown -h now") {
			continue
		}
		minute, minErr := strconv.Atoi(fields[0])
		hour, hourErr := strconv.Atoi(fields[1])
		if minErr != nil || hourErr != nil {
			continue
		}
		return fmt.Sprintf("%02d:%02d", hour, minute)
	}
	return ""
}

func readWakealarmValue() (string, error) {
	data, err := osReadFile(rtcPath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func tailLines(content string, limit int) string {
	if limit <= 0 {
		return ""
	}
	lines := strings.Split(strings.ReplaceAll(strings.TrimRight(content, "\n"), "\r\n", "\n"), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return ""
	}
	if len(lines) <= limit {
		return strings.Join(lines, "\n")
	}
	return strings.Join(lines[len(lines)-limit:], "\n")
}

func yesNo(flag bool) string {
	if flag {
		return "yes"
	}
	return "no"
}
