Start-Transcript -Path "C:\output.txt" -Append
Set-NetFirewallProfile -Profile Domain,Public,Private -Enabled False -Confirm:$false
$gitInstallerUrl = "https://github.com/git-for-windows/git/releases/download/v2.47.1.windows.2/Git-2.47.1.2-64-bit.exe"
$gitInstallerPath = "C:\git-installer.exe"
Invoke-WebRequest -Uri $gitInstallerUrl -OutFile $gitInstallerPath
Start-Process -FilePath $gitInstallerPath -ArgumentList "/SILENT" -Wait
Remove-Item -Path $gitInstallerPath
Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass
Invoke-WebRequest -Uri "https://raw.githubusercontent.com/daytonaio/daytona/refs/heads/main/hack/install.ps1" -OutFile "install.ps1"
Invoke-Expression -Command ".\install.ps1"
$taskName = "RunDaytonaAgent"
$logFile = "C:\Users\daytona\.daytona-agent.log"
$command = "$Env:APPDATA\bin\daytona\daytona.exe agent *>> `"$logFile`" 2>&1"
$action = New-ScheduledTaskAction -Execute "powershell.exe" -Argument "-NoProfile -ExecutionPolicy Bypass -Command `"Start-Process -FilePath 'cmd.exe' -ArgumentList '/c $command' -NoNewWindow -PassThru`""
$trigger = New-ScheduledTaskTrigger -AtStartup
$principal = New-ScheduledTaskPrincipal -UserId "daytona" -LogonType Interactive -RunLevel Limited
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable
Register-ScheduledTask -TaskName $taskName -Action $action -Trigger $trigger -Principal $principal -Settings $settings -Force
Write-Output "Scheduled task '$taskName' created successfully. Logs at $logFile."
Get-WindowsCapability -Online -Name OpenSSH* | Add-WindowsCapability -Online
Set-Service -Name sshd -StartupType Automatic
Start-Service sshd
Start-ScheduledTask -TaskName $taskName
Stop-Transcript
