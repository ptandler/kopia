---
title: "Scheduling Automated Backups"
linkTitle: "Scheduling Automated Backups"
weight: 20
---

Kopia does not include a built-in scheduler, but you can automate snapshots using your operating system's native scheduling tools.

## Prerequisites

Before scheduling automated backups, ensure:
1. You have a Kopia repository configured and connected
2. Your snapshot sources have schedule policies set (see [Scheduling Policy](../../../features/))
3. The kopia binary is in your PATH

## Linux

### Option 1: Cron

1. Open crontab editor:
```shell
crontab -e
```

2. Add an entry for each schedule. For example, to run at 2 AM daily:
```shell
0 2 * * * /usr/local/bin/kopia snapshot run-scheduled
```

Examples for different schedules:

| Schedule | Cron Entry |
|----------|-----------|
| Daily at 2 AM | `0 2 * * * /usr/local/bin/kopia snapshot run-scheduled` |
| Every 4 hours | `0 */4 * * * /usr/local/bin/kopia snapshot run-scheduled` |
| Daily at 8 AM and 8 PM | `0 8,20 * * * /usr/local/bin/kopia snapshot run-scheduled` |
| Every Sunday at 3 AM | `0 3 * * 0 /usr/local/bin/kopia snapshot run-scheduled` |

3. Save and exit.

### Option 2: Systemd Timers

1. Create the service file at `~/.config/systemd/user/kopia-snapshot.service`:
```ini
[Unit]
Description=Kopia Scheduled Snapshots

[Service]
Type=oneshot
ExecStart=/usr/local/bin/kopia snapshot run-scheduled
Environment=KOPIA_PASSWORD=yourpassword
```

2. Create the timer file at `~/.config/systemd/user/kopia-snapshot.timer`:
```ini
[Unit]
Description=Run Kopia snapshots daily

[Timer]
OnCalendar=*-*-* 02:00:00
Persistent=true

[Install]
WantedBy=timers.target
```

3. Enable and start the timer:
```shell
systemctl --user daemon-reload
systemctl --user enable --now kopia-snapshot.timer
```

Other schedule options for `OnCalendar`:

| Schedule | OnCalendar Value |
|----------|----------------|
| Daily at 2 AM | `*-*-* 02:00:00` |
| Every 4 hours | `*-*-*/4:00:00` |
| Weekly on Sunday at 3 AM | `Sun *-*-* 03:00:00` |
| First day of month at midnight | `*-*-01 00:00:00` |

### Option 3: fcron

For more advanced scheduling (e.g., "every 4 hours on weekdays"):
```shell
%every 4h 8-18 *1-5 /usr/local/bin/kopia snapshot run-scheduled
```

## macOS

### Option 1: launchd

1. Create the plist file at `~/Library/LaunchAgents/io.kopia.snapshot.plist`:
```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>io.kopia.snapshot</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/kopia</string>
        <string>snapshot</string>
        <string>run-scheduled</string>
    </array>
    <key>StartCalendarInterval</key>
    <array>
        <dict>
            <key>Hour</key>
            <integer>2</integer>
            <key>Minute</key>
            <integer>0</integer>
        </dict>
    </array>
</dict>
</plist>
```

2. Load the job:
```shell
launchctl load ~/Library/LaunchAgents/io.kopia.snapshot.plist
```

### Option 2: Cron

Same as Linux cron instructions above.

## Windows

### Option 1: Task Scheduler (GUI)

1. Open Task Scheduler (`taskschd.msc`)
2. Create Basic Task:
   - Name: "Kopia Scheduled Snapshots"
   - Trigger: Daily at 2:00 AM (or your preferred time)
3. Action: Start a program
   - Program: `C:\Program Files\Kopia\kopia.exe`
   - Arguments: `snapshot run-scheduled`
4. Configure: Select "Run whether user is logged on or not" if needed

### Option 2: Schtasks (CLI)

```batch
schtasks /create /tn "Kopia Snapshots" /tr "C:\Program Files\Kopia\kopia.exe snapshot run-scheduled" /sc daily /st 02:00
```

Other schedule options:

| Schedule | schtasks Command |
|----------|----------------|
| Every 4 hours | `/sc hourly /mo 4` |
| Every day at 2 PM | `/sc daily /st 14:00` |
| Weekly on Sunday | `/sc weekly /d SUN /st 02:00` |

### Option 3: PowerShell Scheduled Job

```powershell
$action = New-ScheduledTaskAction -Execute 'C:\Program Files\Kopia\kopia.exe' -Argument 'snapshot run-scheduled'
$trigger = New-ScheduledTaskTrigger -Daily -At 2am
Register-ScheduledTask -TaskName "Kopia Snapshots" -Action $action -Trigger $trigger
```

## Security Considerations

### Storing Passwords Securely

Avoid storing passwords in plain text. Options:

1. **Use keyring**: Configure Kopia to use system keyring
2. **Environment variables**: Use `KOPIA_PASSWORD` env var (less secure)
3. **Config file**: Store in config with `--persist-credentials`

### Restrict Permissions

```shell
# Linux: Restrict config file permissions
chmod 600 ~/.config/kopia/repository.config
```

### Use Service Account

For system-wide scheduling, consider a dedicated service account with minimal permissions.

## Troubleshooting

### Check Logs

If scheduled snapshots aren't running, check:
- System scheduler logs
- Kopia logs in `~/.cache/kopia/logs/`

### Dry-Run First

Test your command with `--dry-run`:
```shell
kopia snapshot run-scheduled --dry-run
```

### Manual Test

Run manually to verify it works:
```shell
kopia snapshot run-scheduled
```

### Cron Environment

Cron runs with minimal environment. Use full paths:
```shell
0 2 * * * /usr/bin/path/kopia snapshot run-scheduled --log-dir=/tmp/kopia-logs
```

## Monitoring

Add notification to receive backup reports. See [Notification Configuration](../../../advanced/notifications/).