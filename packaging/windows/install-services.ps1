param(
    [string]$Prefix = (Join-Path $env:LOCALAPPDATA 'BuildOpt'),
    [string]$ServerConfig,
    [string]$EdgeConfig,
    [string]$DefinitionOutput,
    [switch]$Install
)
$ErrorActionPreference = 'Stop'
$Prefix = [System.IO.Path]::GetFullPath($Prefix)
$Definitions = @()
if ($ServerConfig) {
    $ServerConfig = [System.IO.Path]::GetFullPath($ServerConfig)
    if (-not (Test-Path -LiteralPath $ServerConfig -PathType Leaf)) { throw 'Server config is unavailable' }
    $Binary = Join-Path $Prefix 'bin\buildopt-service.exe'
    if (-not (Test-Path -LiteralPath $Binary -PathType Leaf)) { throw 'buildopt-service.exe is unavailable' }
    $Definitions += [ordered]@{ name = 'BuildOptServer'; displayName = 'BuildOpt Server'; binaryPath = ('"{0}" --service-name BuildOptServer --component server --config "{1}"' -f $Binary, $ServerConfig) }
}
if ($EdgeConfig) {
    $EdgeConfig = [System.IO.Path]::GetFullPath($EdgeConfig)
    if (-not (Test-Path -LiteralPath $EdgeConfig -PathType Leaf)) { throw 'Edge config is unavailable' }
    $EdgeBinary = Join-Path $Prefix 'bin\buildopt-edge.exe'
    $Binary = Join-Path $Prefix 'bin\buildopt-service.exe'
    if (-not (Test-Path -LiteralPath $EdgeBinary -PathType Leaf) -or -not (Test-Path -LiteralPath $Binary -PathType Leaf)) { throw 'BuildOpt Edge service binaries are unavailable' }
    & $EdgeBinary validate --config $EdgeConfig
    if ($LASTEXITCODE -ne 0) { throw 'Edge config is invalid' }
    $Definitions += [ordered]@{ name = 'BuildOptEdge'; displayName = 'BuildOpt Edge'; binaryPath = ('"{0}" --service-name BuildOptEdge --component edge --config "{1}"' -f $Binary, $EdgeConfig) }
}
if ($Definitions.Count -eq 0) { throw 'At least one service config is required' }
if ($DefinitionOutput) {
    $DefinitionOutput = [System.IO.Path]::GetFullPath($DefinitionOutput)
    $Definitions | ConvertTo-Json -Depth 4 | Set-Content -Encoding utf8NoBOM $DefinitionOutput
}
if ($Install) {
    foreach ($Definition in $Definitions) {
        $Existing = Get-Service -Name $Definition.name -ErrorAction SilentlyContinue
        if ($Existing) { throw "Service $($Definition.name) already exists; uninstall it before changing its definition" }
        New-Service -Name $Definition.name -DisplayName $Definition.displayName -BinaryPathName $Definition.binaryPath -StartupType Automatic | Out-Null
    }
}
$Definitions | ConvertTo-Json -Depth 4
