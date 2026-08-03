param(
    [string]$Version = 'latest',
    [string]$Prefix = (Join-Path $env:LOCALAPPDATA 'BuildOpt'),
    [switch]$UpdatePath
)
$ErrorActionPreference = 'Stop'
if (-not [Environment]::Is64BitOperatingSystem) { throw 'BuildOpt requires 64-bit Windows' }
if ($Version -eq 'latest') {
    $Version = (Invoke-RestMethod 'https://github.com/tonyredondo/buildopt/releases/latest/download/buildopt-version.txt').Trim()
}
if ($Version -notmatch '^[0-9]+\.[0-9]+\.[0-9]+([+-][0-9A-Za-z.-]+)?$') { throw "Invalid BuildOpt version: $Version" }
$Base = "buildopt-$Version-windows-amd64"
$Release = "https://github.com/tonyredondo/buildopt/releases/download/v$Version"
$Work = Join-Path ([System.IO.Path]::GetTempPath()) ("buildopt-install-" + [guid]::NewGuid())
try {
    New-Item -ItemType Directory -Path $Work | Out-Null
    $Archive = Join-Path $Work "$Base.zip"
    $Checksum = Join-Path $Work "$Base.zip.sha256"
    Invoke-WebRequest "$Release/$Base.zip" -OutFile $Archive
    Invoke-WebRequest "$Release/$Base.zip.sha256" -OutFile $Checksum
    $ExpectedLine = (Get-Content -Raw $Checksum).Trim()
    if ($ExpectedLine -notmatch "^([0-9a-f]{64})  $([regex]::Escape($Base)).zip$") { throw 'Release checksum file is invalid' }
    if ((Get-FileHash -Algorithm SHA256 $Archive).Hash.ToLowerInvariant() -ne $Matches[1]) { throw 'Release archive checksum mismatch' }
    Expand-Archive $Archive $Work
    & (Join-Path $Work "$Base\install.ps1") -Prefix $Prefix -UpdatePath:$UpdatePath
} finally {
    Remove-Item -Recurse -Force -ErrorAction SilentlyContinue $Work
}
