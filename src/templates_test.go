package main

import (
	"os/exec"
	"strings"
	"testing"
)

func TestBuildWakealarmScript(t *testing.T) {
	script, err := buildWakealarmScript("07:30")
	if err != nil {
		t.Fatalf("buildWakealarmScript returned an error: %v", err)
	}

	expected := []string{
		`WAKE_TIME="07:30"`,
		`RTC_PATH="/sys/class/rtc/rtc0/wakealarm"`,
		`LOG_PATH="/var/log/wakealarm-ensure.log"`,
		`now_epoch="$(/bin/date +%s)"`,
		`printf '%s\n' "$wake_epoch"`,
	}
	for _, value := range expected {
		if !strings.Contains(script, value) {
			t.Errorf("rendered script does not contain %q", value)
		}
	}

	cmd := exec.Command("bash", "-n")
	cmd.Stdin = strings.NewReader(script)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("rendered script has invalid Bash syntax: %v: %s", err, output)
	}
}

func TestBuildManagedCronBlock(t *testing.T) {
	block, err := buildManagedCronBlock("23:05")
	if err != nil {
		t.Fatalf("buildManagedCronBlock returned an error: %v", err)
	}

	expected := strings.Join([]string{
		"# BEGIN power-schedule",
		"5 23 * * * /sbin/shutdown -h now",
		"0 * * * * /usr/local/sbin/wakealarm-ensure.sh",
		"# END power-schedule",
	}, "\n")
	if block != expected {
		t.Fatalf("unexpected cron block:\n%s", block)
	}
}
