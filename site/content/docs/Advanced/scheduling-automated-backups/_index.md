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

### Running with Low Priority

Running backup tasks with low priority is important because:

- **Backup operations are I/O-intensive** - They read and write many files, which can slow down other processes
- **May run on servers or desktops** - You don't want backups to interfere with user activity
- **Long-running tasks** - A full backup can take hours; low priority ensures the system stays responsive

To run Kopia snapshots with lower CPU and I/O priority:

```shell
nice -n 15 ionice -c 3 /usr/local/bin/kopia snapshot create-scheduled
```

- `nice -n 15`: Runs with lower CPU priority (15 is fairly low, see `nice -n` range 1-19)
- `ionice -c 3`: Uses **idle** I/O class - only runs when system is truly idle (lowest priority)

**I/O priority classes:**
- `-c 1` (real-time): Highest - can starve other processes, **don't use**
- `-c 2` (best-effort): Normal priority, levels 0-7 (0=highest, 7=lowest)  
- `-c 3` (idle): Lowest - only runs when nothing else needs I/O

**Low priority option** (still runs but doesn't hog I/O):
```shell
nice -n 19 ionice -c 2 -n 7 /usr/local/bin/kopia snapshot create-scheduled
```
- `-c 2 -n 7`: Best-effort with lowest priority

### Option 1: Cron

1. Open crontab editor:
```shell
crontab -e
```

2. Add an entry for each schedule. For example, to run at 2 AM daily:
```shell
0 2 * * * nice -n 15 ionice -c 3 /usr/local/bin/kopia snapshot create-scheduled
```

Examples for different schedules:

| Schedule | Cron Entry |
|----------|-----------|
| Every hour | `0 * * * * nice -n 15 ionice -c 3 /usr/local/bin/kopia snapshot create-scheduled` |
| Every 4 hours | `0 */4 * * * nice -n 15 ionice -c 3 /usr/local/bin/kopia snapshot create-scheduled` |
| Daily at 2 AM | `0 2 * * * nice -n 15 ionice -c 3 /usr/local/bin/kopia snapshot create-scheduled` |
| Daily at 8 AM and 8 PM | `0 8,20 * * * nice -n 15 ionice -c 3 /usr/local/bin/kopia snapshot create-scheduled` |
| Every Sunday at 3 AM | `0 3 * * 0 nice -n 15 ionice -c 3 /usr/local/bin/kopia snapshot create-scheduled` |

3. Save and exit.

### Option 2: Systemd Timers (user-level, recommended)

1. Create the service file at `~/.config/systemd/user/kopia-snapshot.service`:
```ini
[Unit]
Description=Kopia Scheduled Snapshots

[Service]
Type=oneshot
ExecStart=/usr/bin/nice -n 15 /usr/bin/ionice -c 3 /usr/local/bin/kopia snapshot create-scheduled
Environment=KOPIA_PASSWORD=yourpassword
```

2. Create the timer file at `~/.config/systemd/user/kopia-snapshot.timer`:
```ini
[Unit]
Description=Run Kopia snapshots hourly

[Timer]
OnCalendar=*-*-* *:00:00
Persistent=true

[Install]
WantedBy=timers.target
```

3. Enable and start the timer:
```shell
systemctl --user daemon-reload
systemctl --user enable --now kopia-snapshot.timer
```

### Option 3: Systemd (system-wide service)

For running as a system service (all users), create files in `/etc/systemd/system/`:

1. Create `/etc/systemd/system/kopia-snapshot.service`:
```ini
[Unit]
Description=Kopia Scheduled Snapshots
After=network-online.target

[Service]
Type=oneshot
User=yourusername
ExecStart=/usr/bin/nice -n 15 /usr/bin/ionice -c 3 /usr/local/bin/kopia snapshot create-scheduled
Environment=KOPIA_PASSWORD=yourpassword
```

2. Create `/etc/systemd/system/kopia-snapshot.timer`:
```ini
[Unit]
Description=Run Kopia snapshots hourly

[Timer]
OnCalendar=*-*-* *:00:00
Persistent=true

[Install]
WantedBy=timers.target
```

3. Enable and start:
```shell
sudo systemctl daemon-reload
sudo systemctl enable --now kopia-snapshot.timer
```

**Important: Use the same user** that created and owns the Kopia repository:
- Set `User=yourusername` in the service file
- The repository config is typically in `~/.config/kopia/`
- If a different user runs the backup, it won't find the repository

Other schedule options for `OnCalendar`:

| Schedule | OnCalendar Value |
|----------|----------------|
| Every hour | `*-*-* *:00:00` |
| Every 4 hours | `*-*-*/4:00:00` |
| Daily at 2 AM | `*-*-* 02:00:00` |
| Weekly on Sunday at 3 AM | `Sun *-*-* 03:00:00` |
| First day of month at midnight | `*-*-01 00:00:00` |

### Option 4: fcron

For more advanced scheduling (e.g., "every 4 hours on weekdays"):
```shell
%every 4h 8-18 *1-5 /usr/local/bin/kopia snapshot create-scheduled
```

## macOS

### Running with Low Priority

Running backup tasks with low priority is important because backup operations are I/O-intensive and can slow down other processes on your Mac. Use `nice` to lower CPU priority:

```shell
nice -n 15 /usr/local/bin/kopia snapshot create-scheduled
```

- macOS does not have `ionice` - I/O scheduling is handled by the OS automatically
- `nice -n 15` lowers CPU priority

For more aggressive throttling on macOS, consider third-party tools like `iopriority` from [DarwinPorts](https://www.macports.org/).

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
        <string>/usr/bin/nice</string>
        <string>-n</string>
        <string>15</string>
        <string>/usr/local/bin/kopia</string>
        <string>snapshot</string>
        <string>create-scheduled</string>
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

### Running with Low Priority

Running backup tasks with low priority is important because backup operations are I/O-intensive and can slow down other processes. Windows handles I/O automatically; you mainly control CPU priority:

1. **Using Task Scheduler** (recommended):
   - Windows tasks run at normal priority by default, which is usually fine
   - Only use "Run with highest privileges" if truly needed

2. **Using PowerShell with low priority**:
```powershell
$process = Start-Process -FilePath 'C:\Program Files\Kopia\kopia.exe' -ArgumentList 'snapshot create-scheduled' -PassThru -WindowStyle Hidden
$process.PriorityClass = 'BelowNormal'
```

3. **Using CMD with start /low**:
```batch
start /low /b "" "C:\Program Files\Kopia\kopia.exe" snapshot create-scheduled
```

- Windows I/O scheduling is automatic - the OS handles background task priority
- Only CPU priority needs explicit lowering if desired

### Option 1: Task Scheduler (GUI)

1. Open Task Scheduler (`taskschd.msc`)
2. Create Basic Task:
   - Name: "Kopia Scheduled Snapshots"
   - Trigger: Daily at 2:00 AM (or your preferred time)
3. Action: Start a program
   - Program: `C:\Program Files\Kopia\kopia.exe`
   - Arguments: `snapshot create-scheduled`
4. Configure: Select "Run whether user is logged on or not" if needed

### Option 2: Schtasks (CLI)

```batch
schtasks /create /tn "Kopia Snapshots" /tr "C:\Program Files\Kopia\kopia.exe snapshot create-scheduled" /sc daily /st 02:00
```

Other schedule options:

| Schedule | schtasks Command |
|----------|----------------|
| Every 4 hours | `/sc hourly /mo 4` |
| Every day at 2 PM | `/sc daily /st 14:00` |
| Weekly on Sunday | `/sc weekly /d SUN /st 02:00` |

### Option 3: PowerShell Scheduled Job

```powershell
$action = New-ScheduledTaskAction -Execute 'C:\Program Files\Kopia\kopia.exe' -Argument 'snapshot create-scheduled'
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
kopia snapshot create-scheduled --dry-run
```

### Manual Test

Run manually to verify it works:
```shell
kopia snapshot create-scheduled
```

### Cron Environment

Cron runs with minimal environment. Use full paths:
```shell
0 2 * * * nice -n 15 ionice -c 3 /usr/bin/path/kopia snapshot create-scheduled --log-dir=/tmp/kopia-logs
```

## Monitoring

Add notification to receive backup reports. See [Notification Configuration](../../../advanced/notifications/).
