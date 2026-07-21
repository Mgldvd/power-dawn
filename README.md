# Power-Dawn

A Linux Mint focused Go CLI/TUI for scheduling a daily shutdown and RTC-based automatic wake using `/sys/class/rtc/rtc0/wakealarm`. It automates the required setup so it does not have to be completed manually.

```bash
curl -fsSL https://github.com/Mgldvd/power-dawn/releases/latest/download/power-dawn -o power-dawn && chmod +x power-dawn && sudo ./power-dawn
```

<div align="center">
  <img src="./.img/logo.png" alt="moma logo" height="300px">
</div>

## Features

- Single Go binary
- Reproducible Go toolchain and project tasks managed by `mise`
- Bubble Tea TUI with a simple install workflow
- Dynamic `/usr/local/sbin/wakealarm-ensure.sh` generation
- Safe managed root `crontab` block updates
- Status and log inspection screens
- RTC wakealarm verification after installation

## Development Prerequisites

1. Install `mise` by following the [official installation guide](https://mise.jdx.dev/installing-mise.html).
2. Verify that `mise` is available in the current shell.

```bash
mise --version
```

3. Install Go 1.22.12 from the repository configuration.

```bash
mise install
```

`mise` is a required development and build dependency. The compiled `power-dawn` binary does not require `mise` or Go at runtime.

## Build

Build with the project task:

```bash
mise run build
```

Alternatively, use the wrapper script. It checks for `mise`, installs the configured tools, and runs the same build task.

```bash
./build.sh
```

Both commands run `go mod tidy` and create `dist/power-dawn` with the Go version declared in `mise.toml`.

## Check

Run all Go tests:

```bash
mise run test
```

Run Go static analysis:

```bash
mise run vet
```

Run tests and static analysis together:

```bash
mise run check
```

## Run

Build and start the application through `mise`:

```bash
mise run run
```

Alternatively, run an existing binary directly:

```bash
sudo ./dist/power-dawn
```

The application exits immediately when it is not run as `root`.

---

## Notes

- Target platform: Linux Mint
- Development tool manager: `mise` 2024.11.1 or newer
- Go version: 1.22.12, installed and selected by `mise`
- RTC path used: `/sys/class/rtc/rtc0/wakealarm`
- Source files live in `src/`
- Build output goes to `dist/`
- The application updates only the managed cron block between:
  - `# BEGIN power-schedule`
  - `# END power-schedule`

## What the Application Does

`Power-Dawn` provides a terminal interface for configuring two daily power events:

1. Shut down Linux automatically at a selected local time.
2. Ask the hardware RTC to power the computer back on at a selected local time on the following day.

The shutdown is scheduled through the root user's `crontab`. The automatic power-on is scheduled through the Linux RTC wake-alarm interface. The project does not install or manage a `systemd` service.

## Linux System Changes

Running the application and selecting `Install` affects the following system locations:

| Location | Change |
| --- | --- |
| `/var/log/power-dawn.log` | Created or opened as soon as the application starts. Application activity is appended to this file. |
| `/usr/local/sbin/wakealarm-ensure.sh` | Created or replaced during installation with executable mode `0755`. |
| `/var/log/wakealarm-ensure.log` | Created during installation with mode `0644`. RTC scheduling results are appended to this file. |
| Root user's `crontab` | Updated during installation with a managed shutdown and wake-alarm block. |
| `/sys/class/rtc/rtc0/wakealarm` | Read and written when checking, installing, or testing RTC wake functionality. |

Entries outside the following managed markers are preserved when the root `crontab` is updated:

```text
# BEGIN power-schedule
# END power-schedule
```

Starting the application creates or appends to `/var/log/power-dawn.log` even if no schedule is installed.

## Installed Schedule

For a shutdown time of `23:00`, installation adds a block similar to this one to the root user's `crontab`:

```cron
# BEGIN power-schedule
0 23 * * * /sbin/shutdown -h now
0 * * * * /usr/local/sbin/wakealarm-ensure.sh
# END power-schedule
```

The first cron entry shuts down the computer every day at the selected shutdown time. It runs without an additional confirmation prompt.

The second cron entry runs the helper script at minute `00` of every hour. The script keeps the RTC alarm pointed at the selected wake time on the following day.

Installing the schedule also runs the helper script immediately instead of waiting for the next hourly cron execution.

## RTC Wake Process

The generated helper script performs these steps:

1. Calculate a Unix timestamp for the selected wake time on the following day.
2. Read the current alarm from `/sys/class/rtc/rtc0/wakealarm`.
3. Keep the existing value when it already matches the desired timestamp.
4. Run `rtcwake -m no -t <timestamp>` to program the alarm without suspending the computer.
5. Clear and rewrite `/sys/class/rtc/rtc0/wakealarm` directly if `rtcwake` fails or is unavailable.
6. Append the result to `/var/log/wakealarm-ensure.log`.

The later cron shutdown powers off Linux. The motherboard firmware must then detect the RTC alarm and turn the computer back on.

Automatic power-on requires all of the following:

- Physical hardware with RTC/ACPI wake-alarm support.
- A BIOS or UEFI configuration that permits RTC-based power-on.
- Continuous power to the computer while it is shut down.
- A Linux RTC device available at `/sys/class/rtc/rtc0/wakealarm`.

VirtualBox, VMware, and similar virtual machines normally cannot power themselves on through this mechanism.

## Menu Actions

### Install

1. Select the daily shutdown time.
2. Select the daily wake time.
3. Press `Enter` to install the schedule.
4. Allow the application to create the helper script and log file.
5. Allow the application to replace its managed root cron block.
6. Check the displayed RTC value, cron preview, and recent helper log.

Running `Install` again replaces the previous managed block and generated helper script with the newly selected times.

### Status

Select `Status` to inspect:

- The generated helper script.
- The helper-script log file.
- The managed root cron block.
- The configured shutdown and wake times.
- The current RTC alarm value.
- RTC availability and write access.

`Status` is not read-only. It writes `0` to `/sys/class/rtc/rtc0/wakealarm` to test write access. This clears the current RTC alarm and does not restore it.

### View Logs

Select `View Logs` to display up to 50 recent lines from:

- `/var/log/wakealarm-ensure.log`
- `/var/log/power-dawn.log`

### Test Wake

1. Save and close all work.
2. Select `Test Wake`.
3. Read the shutdown warning.
4. Confirm the test only when the computer is ready to shut down.

The test programs an RTC alarm for 60 seconds in the future. It first uses `rtcwake` and then tries the direct sysfs method if necessary. After setting the alarm, it immediately runs:

```bash
shutdown -h now "Power-Dawn: RTC test — will wake in 60 s"
```

The application does not shut down the computer when both RTC programming methods fail.

## Runtime Requirements

- Run the binary as `root`.
- Keep `sudo` installed because internal installation and status commands invoke it even when the application already runs as `root`.
- Keep the cron service installed and running.
- Provide Bash, GNU `date`, `cat`, and standard Linux command-line tools.
- Install `rtcwake` from `util-linux` when possible. Direct sysfs access is used as a fallback.

## Privacy and Network Activity

- The running application does not make network requests.
- The running application does not upload files, logs, or system information.
- `mise install` may access the network to download the configured Go toolchain when it is not already installed.
- The build task may access the network to download Go modules that are not already cached.
- The build task runs `go mod tidy`, which may update `go.mod` and `go.sum` before compiling `dist/power-dawn`.

## Important Warnings and Limitations

- Save all work before the configured shutdown time. The cron shutdown runs automatically without asking for confirmation.
- Save all work before using `Test Wake`. A successful RTC setup is followed by an immediate shutdown.
- Opening `Status` clears the currently programmed RTC alarm. Wait for the next hourly helper execution or run `Install` again before relying on the next automatic wake-up.
- The generated helper script exits successfully even when both RTC programming methods fail. The interface may therefore report a completed installation although the machine will not wake.
- Verify `/sys/class/rtc/rtc0/wakealarm` and `/var/log/wakealarm-ensure.log` before relying on the schedule.
- The helper always schedules the selected wake time for the following day, even when the selected time has not yet occurred on the current day.
- A firmware setting, unsupported motherboard, disconnected power supply, or virtual machine can prevent wake-up even when Linux accepts the RTC alarm.
- The application has no uninstall option. Removing the schedule requires deleting the managed root cron block and the generated files manually.

## Manual Setup Without the Power-Dawn Application

Use these steps to configure the same daily shutdown and RTC wake behavior without building or running `Power-Dawn`. This example shuts down at `23:00` and schedules wake-up for `07:30` on the following day. Replace both times as needed.

This setup uses `rtcwake` directly from the root user's `crontab`. It does not create or use `/usr/local/sbin/wakealarm-ensure.sh`.

### 1. Install the required Linux packages

```bash
sudo apt update
sudo apt install cron util-linux
```

### 2. Enable the cron service

```bash
sudo systemctl enable --now cron
sudo systemctl status cron
```

### 3. Check the RTC device and commands

```bash
sudo test -e /sys/class/rtc/rtc0/wakealarm
command -v rtcwake
command -v shutdown
```

Continue only when the RTC path exists and both commands are available. The standard Linux Mint paths used below are `/usr/sbin/rtcwake` and `/sbin/shutdown`.

### 4. Create the manual RTC log

```bash
sudo touch /var/log/wakealarm-manual.log
sudo chmod 0644 /var/log/wakealarm-manual.log
```

### 5. Open the root user's crontab

```bash
sudo crontab -e
```

### 6. Add the shutdown and RTC entries

Add this block at the bottom of the root `crontab`:

```cron
# BEGIN manual-power-schedule
0 23 * * * /sbin/shutdown -h now
0 * * * * /usr/sbin/rtcwake -m no -t "$(/bin/date -d '07:30 tomorrow' +\%s)" >> /var/log/wakealarm-manual.log 2>&1
# END manual-power-schedule
```

Use cron field order `minute hour day-of-month month day-of-week`. For example, change `0 23` to `30 22` for a shutdown at `22:30`.

Keep the backslash in `+\%s` inside the crontab. Cron treats an unescaped `%` as a line break before passing the command to the shell.

Save and close the editor.

### 7. Verify the installed cron entries

```bash
sudo crontab -l
```

Confirm that only one `manual-power-schedule` block exists.

### 8. Program the first RTC alarm immediately

Do not wait for the next hourly cron execution. Program the example `07:30` wake time now:

```bash
sudo /usr/sbin/rtcwake -m no -t "$(/bin/date -d '07:30 tomorrow' +%s)"
```

The `no` mode programs the RTC without suspending or shutting down Linux.

### 9. Verify the RTC alarm

```bash
sudo cat /sys/class/rtc/rtc0/wakealarm
date -d "@$(sudo cat /sys/class/rtc/rtc0/wakealarm)"
```

Confirm that the converted value matches the intended date and local wake time.

### 10. Check the hourly cron log

Check the log after the next full hour:

```bash
sudo tail -n 50 /var/log/wakealarm-manual.log
```

### 11. Test a complete shutdown and wake cycle

Save and close all work before continuing. These commands schedule an alarm 60 seconds in the future and shut down the computer immediately:

```bash
sudo /usr/sbin/rtcwake -m no -s 60
sudo /sbin/shutdown -h now
```

The computer should power on after approximately 60 seconds. This test requires physical hardware and compatible BIOS/UEFI RTC wake settings.

### 12. Restore the daily alarm after the test

After the computer powers on, restore the example daily wake time:

```bash
sudo /usr/sbin/rtcwake -m no -t "$(/bin/date -d '07:30 tomorrow' +%s)"
```

### Direct sysfs fallback

Use this fallback only when `rtcwake` fails and the RTC device supports direct writes:

```bash
wake_epoch="$(/bin/date -d '07:30 tomorrow' +%s)"
printf '0\n' | sudo tee /sys/class/rtc/rtc0/wakealarm
printf '%s\n' "$wake_epoch" | sudo tee /sys/class/rtc/rtc0/wakealarm
sudo cat /sys/class/rtc/rtc0/wakealarm
```

The hourly cron entry does not include this fallback. Use the `Power-Dawn` application when automatic fallback and separate helper-script logging are required.
