[CmdletBinding()]
param(
    [switch]$ResolveOnly,
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$Packages
)

$ErrorActionPreference = 'Stop'

$candidates = foreach ($directory in [Environment]::GetEnvironmentVariable('PATH', 'Process').Split([IO.Path]::PathSeparator)) {
    $normalizedDirectory = $directory.Trim().Trim('"')
    if ([string]::IsNullOrWhiteSpace($normalizedDirectory)) {
        continue
    }
    $compiler = Join-Path $normalizedDirectory 'gcc.exe'
    if (-not (Test-Path -LiteralPath $compiler -PathType Leaf)) {
        continue
    }
    $versionText = (& $compiler -dumpfullversion 2>$null | Select-Object -First 1).Trim()
    $version = $null
    if (-not [Version]::TryParse($versionText, [ref]$version)) {
        continue
    }
    [pscustomobject]@{
        Path      = (Resolve-Path -LiteralPath $compiler).Path
        Directory = (Resolve-Path -LiteralPath $normalizedDirectory).Path
        Version   = $version
    }
}

$selected = $candidates |
    Sort-Object -Property @{ Expression = 'Version'; Descending = $true }, @{ Expression = 'Path'; Descending = $false } |
    Select-Object -First 1
if ($null -eq $selected) {
    Write-Error 'race-check: no usable gcc.exe was found on PATH'
    exit 2
}

if ($ResolveOnly) {
    Write-Output $selected.Path
    exit 0
}

if ($null -eq $Packages -or $Packages.Count -eq 0) {
    $Packages = @('./...')
}

$remainingPath = [Environment]::GetEnvironmentVariable('PATH', 'Process').Split([IO.Path]::PathSeparator) |
    Where-Object { -not [string]::Equals($_.Trim().Trim('"'), $selected.Directory, [StringComparison]::OrdinalIgnoreCase) }
$env:PATH = (@($selected.Directory) + @($remainingPath)) -join [IO.Path]::PathSeparator
$env:CC = 'gcc'
$env:CXX = 'g++'

[Console]::Error.WriteLine("race-check: using {0} ({1})", $selected.Path, $selected.Version)
& go test -race @Packages
exit $LASTEXITCODE
