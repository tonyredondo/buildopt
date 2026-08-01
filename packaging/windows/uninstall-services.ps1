param([switch]$Remove)
$ErrorActionPreference = 'Stop'
foreach ($Name in @('BuildOptServer', 'BuildOptEdge')) {
    $Service = Get-Service -Name $Name -ErrorAction SilentlyContinue
    if ($Service -and $Remove) {
        if ($Service.Status -ne 'Stopped') { Stop-Service -Name $Name -Force }
        & sc.exe delete $Name | Out-Null
        if ($LASTEXITCODE -ne 0) { throw "Failed to delete service $Name" }
    }
}
