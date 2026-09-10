package stickywrapper

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestPowerShellDirectFallbackRemovesPrivateEnvironment(t *testing.T) {
	shell, err := exec.LookPath("pwsh")
	if err != nil {
		shell, err = exec.LookPath("powershell.exe")
	}
	if err != nil {
		t.Skip("PowerShell runtime unavailable; native platform gate remains separate")
	}
	root := t.TempDir()
	template := filepath.Join(root, "buildoptw.bat")
	if err := os.WriteFile(template, windowsWrapper, 0600); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(root, "check.ps1")
	source := `param([string]$Template)
$ErrorActionPreference = 'Stop'
$raw = [IO.File]::ReadAllText($Template)
$marker = '# BUILDOPT_POWERSHELL'
$body = $raw.Substring($raw.IndexOf($marker) + $marker.Length)
$tokens = $null
$parseErrors = $null
$ast = [System.Management.Automation.Language.Parser]::ParseInput($body, [ref]$tokens, [ref]$parseErrors)
if ($parseErrors.Count -ne 0) { throw 'Windows wrapper syntax rejected' }
$function = $ast.Find({ param($node) $node -is [System.Management.Automation.Language.FunctionDefinitionAst] -and $node.Name -eq 'Clear-BuildOptDirectGradleEnvironment' }, $true)
. ([ScriptBlock]::Create($function.Extent.Text))
$env:buildopt_lowercase = 'private-test'
$env:WCNCP_UPPERCASE = 'private-test'
$env:NATIVE_TEST_VALUE = 'retained'
Clear-BuildOptDirectGradleEnvironment
if ((Test-Path Env:buildopt_lowercase) -or (Test-Path Env:WCNCP_UPPERCASE)) { throw 'Private namespace remains in native environment' }
if ($env:NATIVE_TEST_VALUE -ne 'retained') { throw 'Native environment changed' }
`
	if err := os.WriteFile(script, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, shell, "-NoLogo", "-NoProfile", "-NonInteractive", "-File", script, template)
	if raw, err := command.CombinedOutput(); err != nil {
		t.Fatalf("PowerShell fallback contract: %v\n%s", err, raw)
	}
}
