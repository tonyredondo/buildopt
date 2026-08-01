param([string]$Prefix = (Join-Path $env:LOCALAPPDATA 'BuildOpt'))
$ErrorActionPreference = 'Stop'
$Prefix = [System.IO.Path]::GetFullPath($Prefix)
$ReceiptPath = Join-Path $Prefix 'receipt.json'
if (-not (Test-Path -LiteralPath $ReceiptPath -PathType Leaf)) { throw 'BuildOpt installation receipt is missing' }
$Receipt = Get-Content -Raw $ReceiptPath | ConvertFrom-Json
if ($Receipt.schemaVersion -ne 'buildopt.install/v1') { throw 'BuildOpt installation receipt is invalid' }
foreach ($Relative in $Receipt.files) {
    if ($Relative -notin @('bin/buildopt.exe', 'bin/buildopt-impact.exe', 'bin/buildopt-server.exe', 'bin/buildopt-edge.exe', 'bin/buildopt-service.exe')) { throw "Unsafe receipt entry: $Relative" }
    Remove-Item -Force -ErrorAction SilentlyContinue (Join-Path $Prefix ($Relative -replace '/', '\'))
}
if ($Receipt.pathUpdated) {
    $Bin = Join-Path $Prefix 'bin'
    $Current = [Environment]::GetEnvironmentVariable('Path', 'User')
    $Parts = @($Current -split ';' | Where-Object { $_ -and $_ -ne $Bin })
    [Environment]::SetEnvironmentVariable('Path', ($Parts -join ';'), 'User')
}
Remove-Item -Force $ReceiptPath
Write-Output "BuildOpt removed from $Prefix"
