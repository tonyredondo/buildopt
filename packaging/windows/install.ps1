param(
    [string]$Prefix = (Join-Path $env:LOCALAPPDATA 'BuildOpt'),
    [switch]$UpdatePath
)
$ErrorActionPreference = 'Stop'
$PackageRoot = $PSScriptRoot
$Prefix = [System.IO.Path]::GetFullPath($Prefix)
$Bin = Join-Path $Prefix 'bin'
$Share = Join-Path $Prefix 'share\buildopt'
$Files = @(
    'bin/buildopt.exe', 'bin/buildopt-impact.exe', 'bin/buildopt-server.exe',
    'bin/buildopt-edge.exe', 'bin/buildopt-service.exe',
    'share/buildopt/buildopt.init.gradle',
    'share/buildopt/buildopt-gradle-plugin.jar',
    'share/buildopt/buildopt-jvm-agent.jar'
)
$ReceiptPath = Join-Path $Prefix 'receipt.json'
$PathUpdated = [bool]$UpdatePath
if (Test-Path -LiteralPath $ReceiptPath -PathType Leaf) {
    $PreviousReceipt = Get-Content -Raw $ReceiptPath | ConvertFrom-Json
    if ($PreviousReceipt.schemaVersion -ne 'buildopt.install/v1') { throw 'Existing BuildOpt installation receipt is invalid' }
    foreach ($Relative in $PreviousReceipt.files) {
        if ($Files -notcontains $Relative) { throw "Unsafe existing receipt entry: $Relative" }
    }
    $PathUpdated = $PathUpdated -or [bool]$PreviousReceipt.pathUpdated
}
New-Item -ItemType Directory -Force -Path $Bin, $Share | Out-Null
foreach ($Name in @('buildopt.exe', 'buildopt-impact.exe', 'buildopt-server.exe', 'buildopt-edge.exe', 'buildopt-service.exe')) {
    $ExpectedLine = Select-String -Path (Join-Path $PackageRoot 'SHA256SUMS') -Pattern "  bin/$Name$" | Select-Object -First 1
    if ($null -eq $ExpectedLine) { throw "Missing checksum for $Name" }
    $Expected = ($ExpectedLine.Line -split '  ')[0]
    $Source = Join-Path $PackageRoot "bin\$Name"
    $Actual = (Get-FileHash -Algorithm SHA256 $Source).Hash.ToLowerInvariant()
    if ($Actual -ne $Expected) { throw "Checksum mismatch for $Name" }
    $Temporary = Join-Path $Bin ".$Name.new.$PID"
    Copy-Item -Force $Source $Temporary
    Move-Item -Force $Temporary (Join-Path $Bin $Name)
}
$Plugin = Get-ChildItem (Join-Path $PackageRoot 'lib\buildopt-gradle-plugin-*.jar') | Select-Object -First 1
$Agent = Get-ChildItem (Join-Path $PackageRoot 'lib\buildopt-jvm-agent-*.jar') | Select-Object -First 1
$Assets = @{
    'buildopt.init.gradle' = 'buildopt.init.gradle'
    $Plugin.Name = 'buildopt-gradle-plugin.jar'
    $Agent.Name = 'buildopt-jvm-agent.jar'
}
foreach ($SourceName in $Assets.Keys) {
    $Expected = ((Select-String -Path (Join-Path $PackageRoot 'SHA256SUMS') -Pattern "  lib/$SourceName$").Line -split '  ')[0]
    $Source = Join-Path $PackageRoot "lib\$SourceName"
    if ((Get-FileHash -Algorithm SHA256 $Source).Hash.ToLowerInvariant() -ne $Expected) { throw "Checksum mismatch for $SourceName" }
    Copy-Item -Force $Source (Join-Path $Share $Assets[$SourceName])
}
$Receipt = [ordered]@{ schemaVersion = 'buildopt.install/v1'; files = $Files; pathUpdated = $PathUpdated }
$Receipt | ConvertTo-Json | Set-Content -Encoding utf8NoBOM $ReceiptPath
if ($PathUpdated) {
    $Current = [Environment]::GetEnvironmentVariable('Path', 'User')
    $Parts = @($Current -split ';' | Where-Object { $_ -and $_ -ne $Bin })
    [Environment]::SetEnvironmentVariable('Path', (($Parts + $Bin) -join ';'), 'User')
}
Write-Output "BuildOpt installed in $Bin"
