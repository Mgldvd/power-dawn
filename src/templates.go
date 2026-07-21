package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"text/template"
)

//go:embed files/wakealarm-ensure.sh.tmpl
var wakealarmScriptTemplate string

//go:embed files/power-schedule.cron.tmpl
var powerScheduleTemplate string

type wakealarmScriptData struct {
	WakeTime string
	RTCPath  string
	LogPath  string
}

type powerScheduleData struct {
	Minute     int
	Hour       int
	ScriptPath string
	CronBegin  string
	CronEnd    string
}

func buildWakealarmScript(wakeTime string) (string, error) {
	return renderEmbeddedTemplate("wakealarm-ensure.sh", wakealarmScriptTemplate, wakealarmScriptData{
		WakeTime: wakeTime,
		RTCPath:  rtcPath,
		LogPath:  logPath,
	})
}

func buildManagedCronBlock(shutdownTime string) (string, error) {
	minute, hour, err := cronFieldsForTime(shutdownTime)
	if err != nil {
		return "", err
	}

	rendered, err := renderEmbeddedTemplate("power-schedule.cron", powerScheduleTemplate, powerScheduleData{
		Minute:     minute,
		Hour:       hour,
		ScriptPath: scriptPath,
		CronBegin:  cronBegin,
		CronEnd:    cronEnd,
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(rendered), nil
}

func renderEmbeddedTemplate(name, source string, data any) (string, error) {
	tmpl, err := template.New(name).Option("missingkey=error").Parse(source)
	if err != nil {
		return "", fmt.Errorf("parse embedded template %s: %w", name, err)
	}

	var output bytes.Buffer
	if err := tmpl.Execute(&output, data); err != nil {
		return "", fmt.Errorf("render embedded template %s: %w", name, err)
	}
	return output.String(), nil
}
