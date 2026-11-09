@echo off
setlocal
set "INVESTONE_BIN=%~dp0InvestOne.exe"
powershell.exe -NoProfile -Command "$ErrorActionPreference='Stop'; $target=[IO.Path]::GetFullPath($env:INVESTONE_BIN); $running=@(Get-CimInstance Win32_Process -Filter \"Name = 'InvestOne.exe'\" | Where-Object { $_.ExecutablePath -and ([IO.Path]::GetFullPath($_.ExecutablePath) -ieq $target) }); if ($running.Count -eq 0) { Write-Output 'No matching InvestOne process.' } else { foreach($p in $running) { Stop-Process -Id $p.ProcessId; Write-Output ('Stopped InvestOne process '+$p.ProcessId) } }"
if errorlevel 1 echo Could not stop InvestOne automatically.
pause
