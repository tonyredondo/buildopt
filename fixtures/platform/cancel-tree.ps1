$ErrorActionPreference = 'Stop'
if ([string]::IsNullOrWhiteSpace($env:BUILDOPT_CANCEL_PID_FILE)) { throw 'missing BUILDOPT_CANCEL_PID_FILE' }
$Child = Start-Process -FilePath 'powershell.exe' -ArgumentList @('-NoProfile', '-Command', 'Start-Sleep -Seconds 300') -PassThru
[System.IO.File]::WriteAllText($env:BUILDOPT_CANCEL_PID_FILE, "$($Child.Id)`n")
$Child.WaitForExit()
