param(
    [Parameter(Mandatory = $true)][string]$Version,
    [Parameter(Mandatory = $true)][string]$Output
)
$ErrorActionPreference = 'Stop'
if ($Version -notmatch '^[0-9]+\.[0-9]+\.[0-9]+([+-][0-9A-Za-z.-]+)?$') {
    throw "Invalid BuildOpt version: $Version"
}
$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
$Architecture = (& go env GOARCH).Trim()
if ($Architecture -notin @('amd64', 'arm64')) {
    throw "Unsupported Windows architecture: $Architecture"
}
New-Item -ItemType Directory -Force -Path $Output | Out-Null
$Work = Join-Path ([System.IO.Path]::GetTempPath()) ("buildopt-package-" + [guid]::NewGuid())
$Base = "buildopt-$Version-windows-$Architecture"
$Root = Join-Path $Work $Base
try {
    New-Item -ItemType Directory -Force -Path (Join-Path $Root 'bin') | Out-Null
    Push-Location $RepoRoot
    try {
        $env:CGO_ENABLED = '0'
        $env:GOOS = 'windows'
        $env:GOARCH = $Architecture
        foreach ($Name in @('buildopt', 'buildopt-impact', 'buildopt-server', 'buildopt-edge', 'buildopt-service')) {
            & go build -mod=readonly -buildvcs=false -trimpath -o (Join-Path $Root "bin\$Name.exe") "./cmd/$Name"
            if ($LASTEXITCODE -ne 0) { throw "$Name build failed" }
        }
    } finally {
        Pop-Location
    }
    Copy-Item (Join-Path $PSScriptRoot 'install.ps1'), (Join-Path $PSScriptRoot 'uninstall.ps1'), (Join-Path $PSScriptRoot 'install-services.ps1'), (Join-Path $PSScriptRoot 'uninstall-services.ps1') -Destination $Root
    $Checksums = @()
    foreach ($Name in @('buildopt.exe', 'buildopt-impact.exe', 'buildopt-server.exe', 'buildopt-edge.exe', 'buildopt-service.exe')) {
        $Hash = (Get-FileHash -Algorithm SHA256 (Join-Path $Root "bin\$Name")).Hash.ToLowerInvariant()
        $Checksums += "$Hash  bin/$Name"
    }
    [System.IO.File]::WriteAllLines((Join-Path $Root 'SHA256SUMS'), $Checksums, [System.Text.UTF8Encoding]::new($false))
    $Archive = Join-Path $Output "$Base.zip"
    if (Test-Path $Archive) { Remove-Item -Force $Archive }
    Compress-Archive -Path $Root -DestinationPath $Archive -CompressionLevel Optimal
    Write-Output $Archive
} finally {
    Remove-Item -Recurse -Force -ErrorAction SilentlyContinue $Work
}
