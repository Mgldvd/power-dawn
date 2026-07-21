package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var osReadFile = os.ReadFile

func runCommand(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(output))
	if err != nil {
		if text == "" {
			return text, fmt.Errorf("%s %s failed: %w", name, strings.Join(args, " "), err)
		}
		return text, fmt.Errorf("%s %s failed: %w: %s", name, strings.Join(args, " "), err, text)
	}
	return text, nil
}

func runCommandWithInput(ctx context.Context, input string, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = bytes.NewBufferString(input)
	output, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(output))
	if err != nil {
		if text == "" {
			return text, fmt.Errorf("%s %s failed: %w", name, strings.Join(args, " "), err)
		}
		return text, fmt.Errorf("%s %s failed: %w: %s", name, strings.Join(args, " "), err, text)
	}
	return text, nil
}

func isSudoPasswordError(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "a password is required") ||
		strings.Contains(lower, "sudo") && strings.Contains(lower, "password")
}

func isNoCrontabOutput(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "no crontab for")
}
