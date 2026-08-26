@echo off
setlocal
set "BUILDOPT_WRAPPER_FILE=%~f0"
powershell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -Command "$forward=@();$seen=$false;foreach($item in [Environment]::GetCommandLineArgs()){if($seen){$forward+=,$item}elseif($item -ceq '--buildopt-wrapper-arguments'){$seen=$true}};if(-not $seen){exit 70};$commandArgs=@($args);if($commandArgs.Count -gt 0 -and $commandArgs[0] -ceq '--buildopt-wrapper-arguments'){$commandArgs=@($commandArgs|Select-Object -Skip 1)};if($commandArgs.Count -gt 0 -and $commandArgs[0] -ceq '--buildopt'){$forward=$commandArgs};$raw=[IO.File]::ReadAllText($env:BUILDOPT_WRAPPER_FILE);$marker='# BUILDOPT_'+'POWERSHELL';$offset=$raw.IndexOf($marker);if($offset -lt 0){exit 70};$body=$raw.Substring($offset+$marker.Length);& ([ScriptBlock]::Create($body)) @forward" --buildopt-wrapper-arguments %*
exit /b %ERRORLEVEL%
# BUILDOPT_POWERSHELL
$ErrorActionPreference = 'Stop'

$WrapperArgs = @($args)
if ($WrapperArgs.Count -gt 0 -and $WrapperArgs[0] -ceq '--') {
    $WrapperArgs = @($WrapperArgs | Select-Object -Skip 1)
}
$ForceGradle = $false
if ($WrapperArgs.Count -gt 0 -and $WrapperArgs[0] -ceq '--gradle') {
    $ForceGradle = $true
    $WrapperArgs = @($WrapperArgs | Select-Object -Skip 1)
}
$ManagementInvocation = -not $ForceGradle -and $WrapperArgs.Count -gt 0 -and $WrapperArgs[0] -ceq '--buildopt'
$BootstrapFallbackEnabled = $false
$GradleWrapper = $null

function Clear-BuildOptChildEnvironment {
    $Names = @(
        'BUILDOPT_BYPASS', 'BUILDOPT_WRAPPER_CACHE_HOME',
        'BUILDOPT_PLUGIN_ATTEMPT_ID', 'BUILDOPT_PLUGIN_SOCKET', 'BUILDOPT_PLUGIN_TOKEN',
        'BUILDOPT_GATEWAY_URL', 'BUILDOPT_GATEWAY_USERNAME', 'BUILDOPT_GATEWAY_PASSWORD',
        'BUILDOPT_GATEWAY_GENERATION', 'BUILDOPT_MANAGED_GATEWAY_STATE_ROOT',
        'BUILDOPT_MANAGED_RUNNER_SLOT', 'BUILDOPT_MANAGED_GATEWAY_IDLE_TIMEOUT',
        'BUILDOPT_L1_STATE_ROOT', 'BUILDOPT_L1_TENANT', 'BUILDOPT_L1_REPOSITORY',
        'BUILDOPT_L1_TRUST_DOMAIN', 'BUILDOPT_L1_COMPATIBILITY', 'BUILDOPT_L1_GENERATION',
        'BUILDOPT_L1_L2_WRITER', 'BUILDOPT_L1_DIRECTORY', 'BUILDOPT_L1_MODE',
        'BUILDOPT_L1_OUTPUT_GENERATION', 'BUILDOPT_L1_RETENTION_GENERATIONS',
        'BUILDOPT_LOCAL_AUTHORITY_PATH', 'BUILDOPT_LOCAL_TRUST_ROOT_PATH',
        'BUILDOPT_LOCAL_CREDENTIAL_PATH', 'BUILDOPT_SHARED_CACHE_TOKEN_PATH',
        'BUILDOPT_SHARED_CACHE_CA_PATH', 'BUILDOPT_SHARED_CACHE_URL',
        'BUILDOPT_MANAGED_SHARED_MODE', 'BUILDOPT_MANAGED_AUTHORITY_DIGEST',
        'BUILDOPT_MANAGED_POLICY_DIGEST', 'BUILDOPT_MANAGED_CONFIGURATION_DIGEST',
        'BUILDOPT_MANAGED_AUTHORITY_CONTRACT', 'BUILDOPT_GRADLE_BOOTSTRAP_CONFIG_PATH',
        'BUILDOPT_SERVER_URL', 'BUILDOPT_SERVER_TOKEN', 'BUILDOPT_SESSION_CONTEXT',
        'BUILDOPT_GRADLE_SAFE_CACHE', 'BUILDOPT_GRADLE_STANDARD_JAR_CACHE',
        'BUILDOPT_POC_EDGE_CACHE_URL', 'BUILDOPT_RUNTIME_CHECKSTYLE_MAX_HEAP',
        'BUILDOPT_CACHE_STANDARD_COPY_TASKS'
    )
    foreach ($Name in $Names) {
        [Environment]::SetEnvironmentVariable($Name, $null, 'Process')
    }
}

function Invoke-GradleDirect {
    Clear-BuildOptChildEnvironment
    & $GradleWrapper @WrapperArgs
    exit $LASTEXITCODE
}

function ConvertTo-NativeArgument([string]$Argument) {
    if ($Argument.Length -gt 0 -and $Argument -cnotmatch '[\s"]') { return $Argument }
    $Quoted = [Text.StringBuilder]::new()
    [void]$Quoted.Append('"')
    $Backslashes = 0
    foreach ($Character in $Argument.ToCharArray()) {
        if ($Character -ceq '\') {
            $Backslashes++
        } elseif ($Character -ceq '"') {
            [void]$Quoted.Append(('\' * (($Backslashes * 2) + 1)))
            [void]$Quoted.Append('"')
            $Backslashes = 0
        } else {
            if ($Backslashes -gt 0) { [void]$Quoted.Append(('\' * $Backslashes)) }
            [void]$Quoted.Append($Character)
            $Backslashes = 0
        }
    }
    if ($Backslashes -gt 0) { [void]$Quoted.Append(('\' * ($Backslashes * 2))) }
    [void]$Quoted.Append('"')
    return $Quoted.ToString()
}

function Invoke-NativePassthrough([string]$FilePath, [string[]]$Arguments) {
    $StartInfo = [Diagnostics.ProcessStartInfo]::new()
    $StartInfo.FileName = $FilePath
    $StartInfo.UseShellExecute = $false
    $EscapedArguments = @($Arguments | ForEach-Object { ConvertTo-NativeArgument $_ })
    $StartInfo.Arguments = [string]::Join(' ', $EscapedArguments)
    $Process = [Diagnostics.Process]::Start($StartInfo)
    if ($null -eq $Process) { Stop-Wrapper 126 'verified BuildOpt binary could not start' }
    try {
        $Process.WaitForExit()
        return $Process.ExitCode
    } finally { $Process.Dispose() }
}

function Stop-Wrapper([int]$Code, [string]$Message) {
    if ($BootstrapFallbackEnabled) {
        [Console]::Error.WriteLine('buildoptw: verified bootstrap unavailable; running Gradle directly')
        Invoke-GradleDirect
    }
    [Console]::Error.WriteLine("buildoptw: $Message")
    exit $Code
}

$WrapperPath = [IO.Path]::GetFullPath($env:BUILDOPT_WRAPPER_FILE)
$RepositoryRoot = Split-Path -Parent $WrapperPath
$GradleWrapper = Join-Path $RepositoryRoot 'gradlew.bat'
if (-not $ManagementInvocation -or $env:BUILDOPT_BYPASS -ceq '1') {
    if (-not (Test-Path -LiteralPath $GradleWrapper)) {
        Stop-Wrapper 127 'repository Gradle Wrapper is missing'
    }
    if (-not (Test-Path -LiteralPath $GradleWrapper -PathType Leaf)) {
        Stop-Wrapper 126 'repository Gradle Wrapper is not executable'
    }
}
if ($env:BUILDOPT_BYPASS -ceq '1') {
    Invoke-GradleDirect
}
if (-not $ManagementInvocation) {
    $BootstrapFallbackEnabled = $true
}

$PropertiesPath = Join-Path $RepositoryRoot '.buildopt\wrapper.properties'
if (-not (Test-Path -LiteralPath $PropertiesPath -PathType Leaf) -or
    ((Get-Item -LiteralPath $PropertiesPath -Force).Attributes -band [IO.FileAttributes]::ReparsePoint)) {
    Stop-Wrapper 65 'wrapper properties are missing or unsafe'
}

$ExpectedKeys = @(
    'schemaVersion', 'distributionVersion',
    'distributionUrl.linux-amd64', 'distributionSha256.linux-amd64',
    'distributionUrl.macos-amd64', 'distributionSha256.macos-amd64',
    'distributionUrl.macos-arm64', 'distributionSha256.macos-arm64',
    'distributionUrl.windows-amd64', 'distributionSha256.windows-amd64',
    'network.connectTimeoutMs', 'network.readTimeoutMs',
    'network.redirectPolicy', 'network.proxyMode'
)
$Lines = [IO.File]::ReadAllLines($PropertiesPath)
if ($Lines.Count -ne $ExpectedKeys.Count) { Stop-Wrapper 65 'wrapper properties entry count is invalid' }
$Properties = @{}
for ($Index = 0; $Index -lt $ExpectedKeys.Count; $Index++) {
    $Separator = $Lines[$Index].IndexOf('=')
    if ($Separator -lt 1 -or $Lines[$Index].IndexOf('=', $Separator + 1) -ge 0) {
        Stop-Wrapper 65 "property $($ExpectedKeys[$Index]) is malformed"
    }
    $Key = $Lines[$Index].Substring(0, $Separator)
    if ($Key -cne $ExpectedKeys[$Index]) { Stop-Wrapper 65 "expected property $($ExpectedKeys[$Index])" }
    $Properties[$Key] = $Lines[$Index].Substring($Separator + 1)
}
if ($Properties.schemaVersion -cne 'buildopt.wrapper/v1' -or
    $Properties.distributionVersion -cnotmatch '^[0-9]+\.[0-9]+\.[0-9]+$' -or
    $Properties.'network.connectTimeoutMs' -cne '5000' -or
    $Properties.'network.readTimeoutMs' -cne '30000' -or
    $Properties.'network.redirectPolicy' -cne 'https-only-max-5' -or
    $Properties.'network.proxyMode' -cne 'environment') {
    Stop-Wrapper 65 'wrapper properties contain unsupported values'
}

$Architecture = [Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()
if ($Architecture -cne 'X64') { Stop-Wrapper 69 'this Windows architecture is unsupported' }
$Version = $Properties.distributionVersion
$Platform = 'windows-amd64'
$ArchiveRoot = "buildopt-$Version-windows-amd64"
$Url = $Properties.'distributionUrl.windows-amd64'
$ArchiveSha = $Properties.'distributionSha256.windows-amd64'
if ($ArchiveSha -cnotmatch '^[0-9a-f]{64}$') { Stop-Wrapper 65 'distribution checksum is invalid' }
try { $Uri = [Uri]::new($Url) } catch { Stop-Wrapper 65 'distribution URL is invalid' }
if ($Uri.Scheme -cne 'https' -or -not [string]::IsNullOrEmpty($Uri.UserInfo) -or
    -not [string]::IsNullOrEmpty($Uri.Query) -or -not [string]::IsNullOrEmpty($Uri.Fragment) -or
    $Uri.AbsolutePath -match '(?i)/latest/' -or
    ($Uri.AbsolutePath -notmatch "/v?$([regex]::Escape($Version))/")) {
    Stop-Wrapper 65 'distribution URL is not immutable HTTPS'
}

$MarkerLines = @('buildopt.wrapper-install/v1', $Version, $Platform, $ArchiveSha)
function Get-Sha256Hex([string]$Path) {
    $InputStream = [IO.File]::OpenRead($Path)
    try {
        $Hasher = [Security.Cryptography.SHA256]::Create()
        try {
            return ([BitConverter]::ToString($Hasher.ComputeHash($InputStream))).Replace('-', '').ToLowerInvariant()
        } finally { $Hasher.Dispose() }
    } finally { $InputStream.Dispose() }
}

function Test-Manifest([string]$Root) {
    if (-not (Test-Path -LiteralPath $Root -PathType Container)) { return $false }
    $RootItem = Get-Item -LiteralPath $Root -Force
    if ($RootItem.Attributes -band [IO.FileAttributes]::ReparsePoint) { return $false }
    $Binary = Join-Path $Root 'bin\buildopt.exe'
    $Manifest = Join-Path $Root 'SHA256SUMS'
    if (-not (Test-Path -LiteralPath $Binary -PathType Leaf) -or
        -not (Test-Path -LiteralPath $Manifest -PathType Leaf)) { return $false }
    foreach ($Item in Get-ChildItem -LiteralPath $Root -Force -Recurse) {
        if ($Item.Attributes -band [IO.FileAttributes]::ReparsePoint) { return $false }
    }
    $Count = 0
    foreach ($Line in [IO.File]::ReadAllLines($Manifest)) {
        if ($Line -cnotmatch '^([0-9a-f]{64})  ((bin|lib)/[A-Za-z0-9._/-]+)$' -or $Matches[2] -match '(^|/)\.\.(/|$)') {
            return $false
        }
        $Target = Join-Path $Root ($Matches[2].Replace('/', '\'))
        if (-not (Test-Path -LiteralPath $Target -PathType Leaf)) { return $false }
        if ((Get-Sha256Hex $Target) -cne $Matches[1]) { return $false }
        $Count++
    }
    return $Count -ge 4
}

function Test-Install([string]$Root) {
    $Marker = Join-Path $Root '.buildopt-wrapper-install'
    if (-not (Test-Path -LiteralPath $Marker -PathType Leaf)) { return $false }
    if ((Get-Item -LiteralPath $Marker -Force).Attributes -band [IO.FileAttributes]::ReparsePoint) { return $false }
    $Actual = @([IO.File]::ReadAllLines($Marker))
    if ($Actual.Count -ne $MarkerLines.Count) { return $false }
    for ($Index = 0; $Index -lt $MarkerLines.Count; $Index++) {
        if ($Actual[$Index] -cne $MarkerLines[$Index]) { return $false }
    }
    return Test-Manifest $Root
}

$CacheHome = $env:BUILDOPT_WRAPPER_CACHE_HOME
if ([string]::IsNullOrEmpty($CacheHome)) { $CacheHome = Join-Path $env:LOCALAPPDATA 'BuildOptCache' }
if ([string]::IsNullOrEmpty($CacheHome) -or -not [IO.Path]::IsPathRooted($CacheHome) -or $CacheHome.Contains("`n") -or $CacheHome.Contains("`r")) {
    Stop-Wrapper 70 'wrapper cache root must be an absolute safe path'
}
$Parent = Join-Path $CacheHome "buildopt\wrapper\distributions\$Version\$Platform"
$InstallRoot = Join-Path $Parent $ArchiveSha
$Lock = "$InstallRoot.lock"

if (Test-Path -LiteralPath $InstallRoot) {
    if (-not (Test-Install $InstallRoot)) { Stop-Wrapper 74 'cached distribution failed verification' }
} else {
    New-Item -ItemType Directory -Force -Path $Parent | Out-Null
    $HaveLock = $false
    for ($Attempt = 0; $Attempt -lt 300 -and -not $HaveLock; $Attempt++) {
        try {
            New-Item -ItemType Directory -Path $Lock -ErrorAction Stop | Out-Null
            $HaveLock = $true
        } catch {
            if ((Test-Path -LiteralPath $InstallRoot) -and (Test-Install $InstallRoot)) { break }
            Start-Sleep -Milliseconds 100
        }
    }
    if (-not $HaveLock -and -not (Test-Install $InstallRoot)) { Stop-Wrapper 70 'another bootstrap did not complete' }
    if ($HaveLock) {
        $Work = Join-Path $Parent ('.buildopt-wrapper.' + [guid]::NewGuid().ToString('N'))
        try {
            New-Item -ItemType Directory -Path $Work | Out-Null
            if (Test-Path -LiteralPath $InstallRoot) {
                if (-not (Test-Install $InstallRoot)) { Stop-Wrapper 74 'cached distribution failed verification' }
            } else {
                $Archive = Join-Path $Work 'distribution.zip'
                $Response = $null
                try {
                    $CurrentUri = $Uri
                    $RequestClock = [Diagnostics.Stopwatch]::StartNew()
                    for ($Redirects = 0; $Redirects -le 5; $Redirects++) {
                        $Request = [Net.HttpWebRequest]::Create($CurrentUri)
                        $Request.AllowAutoRedirect = $false
                        $Request.Timeout = 5000
                        $Request.ReadWriteTimeout = 30000
                        $Request.Method = 'GET'
                        $Response = $Request.GetResponse()
                        if ([int]$Response.StatusCode -lt 300 -or [int]$Response.StatusCode -ge 400) { break }
                        $Location = $Response.Headers['Location']
                        if ($Redirects -eq 5 -or [string]::IsNullOrEmpty($Location)) { Stop-Wrapper 69 'distribution redirect limit was exceeded' }
                        $NextUri = [Uri]::new($CurrentUri, $Location)
                        if ($NextUri.Scheme -cne 'https') { Stop-Wrapper 69 'distribution redirect must use HTTPS' }
                        $Response.Dispose()
                        $Response = $null
                        $CurrentUri = $NextUri
                    }
                    if ([int]$Response.StatusCode -lt 200 -or [int]$Response.StatusCode -ge 300) { Stop-Wrapper 69 'distribution download failed' }
                    $InputStream = $Response.GetResponseStream()
                    $OutputStream = [IO.File]::Create($Archive)
                    try {
                        $Buffer = [byte[]]::new(65536)
                        while ($true) {
                            $Remaining = [TimeSpan]::FromSeconds(30) - $RequestClock.Elapsed
                            if ($Remaining -le [TimeSpan]::Zero) { Stop-Wrapper 69 'distribution download timed out' }
                            $ReadCancellation = [Threading.CancellationTokenSource]::new($Remaining)
                            try {
                                $Read = $InputStream.ReadAsync($Buffer, 0, $Buffer.Length, $ReadCancellation.Token).GetAwaiter().GetResult()
                            } finally { $ReadCancellation.Dispose() }
                            if ($Read -eq 0) { break }
                            $OutputStream.Write($Buffer, 0, $Read)
                        }
                    } finally { $OutputStream.Dispose(); $InputStream.Dispose() }
                } catch {
                    if ($_.Exception -is [System.Management.Automation.ExitException]) { throw }
                    Stop-Wrapper 69 'distribution download failed'
                } finally {
                    if ($null -ne $Response) { $Response.Dispose() }
                }
                if ((Get-Sha256Hex $Archive) -cne $ArchiveSha) {
                    Stop-Wrapper 74 'distribution checksum mismatch'
                }
                Add-Type -AssemblyName System.IO.Compression.FileSystem
                $Zip = [IO.Compression.ZipFile]::OpenRead($Archive)
                try {
                    $Names = [Collections.Generic.HashSet[string]]::new([StringComparer]::Ordinal)
                    foreach ($Entry in $Zip.Entries) {
                        $Name = $Entry.FullName
                        if ([string]::IsNullOrEmpty($Name) -or $Name.Length -gt 1024 -or $Name.Contains('\') -or
                            -not $Name.StartsWith("$ArchiveRoot/", [StringComparison]::Ordinal) -or
                            -not $Names.Add($Name) -or ($Name.Split('/') -contains '..') -or
                            ((($Entry.ExternalAttributes -shr 16) -band 0xF000) -eq 0xA000)) {
                            Stop-Wrapper 74 'distribution archive contains an unsafe entry'
                        }
                    }
                    foreach ($Entry in $Zip.Entries) {
                        $Destination = Join-Path $Work ($Entry.FullName.Replace('/', '\'))
                        if ($Entry.FullName.EndsWith('/')) {
                            New-Item -ItemType Directory -Force -Path $Destination | Out-Null
                        } else {
                            New-Item -ItemType Directory -Force -Path (Split-Path -Parent $Destination) | Out-Null
                            if (Test-Path -LiteralPath $Destination) { Stop-Wrapper 74 'distribution archive contains duplicate output' }
                            $InputStream = $Entry.Open()
                            $OutputStream = [IO.File]::Create($Destination)
                            try { $InputStream.CopyTo($OutputStream) } finally { $OutputStream.Dispose(); $InputStream.Dispose() }
                        }
                    }
                } finally { $Zip.Dispose() }
                $Staged = Join-Path $Work $ArchiveRoot
                if (-not (Test-Manifest $Staged)) { Stop-Wrapper 74 'distribution contents failed verification' }
                [IO.File]::WriteAllLines((Join-Path $Staged '.buildopt-wrapper-install'), $MarkerLines, [Text.UTF8Encoding]::new($false))
                [IO.Directory]::Move($Staged, $InstallRoot)
                if (-not (Test-Install $InstallRoot)) { Stop-Wrapper 74 'published distribution failed verification' }
            }
        } finally {
            Remove-Item -LiteralPath $Work -Recurse -Force -ErrorAction SilentlyContinue
            Remove-Item -LiteralPath $Lock -Force -ErrorAction SilentlyContinue
        }
    }
}

if ($ManagementInvocation -and
    ($WrapperArgs.Count -eq 2 -or ($WrapperArgs.Count -eq 3 -and $WrapperArgs[2] -ceq '--json')) -and
    $WrapperArgs[0] -ceq '--buildopt' -and $WrapperArgs[1] -ceq 'version') {
    if ($WrapperArgs.Count -eq 3) {
        Write-Output "{`"schemaVersion`":`"buildopt.wrapper-status/v1`",`"distributionVersion`":`"$Version`",`"bootstrap`":`"VERIFIED`"}"
    } else {
        Write-Output "BuildOpt Wrapper $Version (verified)"
    }
    exit 0
}
if ($ManagementInvocation) {
    Stop-Wrapper 64 'unknown or invalid management command'
}

$BootstrapFallbackEnabled = $false
Clear-BuildOptChildEnvironment
$BuildOptBinary = Join-Path $InstallRoot 'bin\buildopt.exe'
# Preserve the public buildopt run -- contract while passing each argument separately.
$BuildOptArguments = @('run', '--', $GradleWrapper) + $WrapperArgs
$BuildOptExitCode = Invoke-NativePassthrough $BuildOptBinary $BuildOptArguments
exit $BuildOptExitCode
