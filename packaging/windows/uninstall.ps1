param([string]$Prefix = (Join-Path $env:LOCALAPPDATA 'BuildOpt'))
$ErrorActionPreference = 'Stop'
$Prefix = [System.IO.Path]::GetFullPath($Prefix)
$ReceiptPath = Join-Path $Prefix 'receipt.json'
if (-not (Test-Path -LiteralPath $ReceiptPath -PathType Leaf)) { throw 'BuildOpt installation receipt is missing' }
$Receipt = Get-Content -Raw $ReceiptPath | ConvertFrom-Json
if ($Receipt.schemaVersion -ne 'buildopt.install/v1') { throw 'BuildOpt installation receipt is invalid' }
$Allowed = @('bin/buildopt.exe', 'bin/buildopt-impact.exe', 'bin/buildopt-server.exe', 'bin/buildopt-edge.exe', 'bin/buildopt-service.exe', 'share/buildopt/buildopt.init.gradle', 'share/buildopt/buildopt-gradle-plugin.jar', 'share/buildopt/buildopt-jvm-agent.jar')
function Remove-BuildOptFile {
    param([Parameter(Mandatory = $true)][string]$Path)
    for ($Attempt = 1; $Attempt -le 10; $Attempt++) {
        Remove-Item -LiteralPath $Path -Force -ErrorAction SilentlyContinue
        if (-not (Test-Path -LiteralPath $Path)) { return }
        Start-Sleep -Milliseconds 250
    }
    throw "BuildOpt uninstall could not remove $Path after bounded retries"
}
foreach ($Relative in $Receipt.files) {
    if ($Allowed -notcontains $Relative) { throw "Unsafe receipt entry: $Relative" }
    Remove-BuildOptFile (Join-Path $Prefix ($Relative -replace '/', '\'))
}
if ($Receipt.pathUpdated) {
    $Bin = Join-Path $Prefix 'bin'
    $Current = [Environment]::GetEnvironmentVariable('Path', 'User')
    $Parts = @($Current -split ';' | Where-Object { $_ -and $_ -ne $Bin })
    [Environment]::SetEnvironmentVariable('Path', ($Parts -join ';'), 'User')
}
Remove-BuildOptFile $ReceiptPath
Write-Output "BuildOpt removed from $Prefix"
