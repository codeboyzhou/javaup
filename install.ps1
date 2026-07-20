<#
.SYNOPSIS
  Installs jup on Windows with PowerShell 5.1 or later.

.EXAMPLE
  irm https://github.com/codeboyzhou/javaup/releases/latest/download/install.ps1 | iex

.EXAMPLE
  $env:JUP_VERSION = 'v0.1.0'
  irm https://github.com/codeboyzhou/javaup/releases/latest/download/install.ps1 | iex

.NOTES
  Optional environment variables:
    JUP_VERSION         Release version to install, such as v0.1.0 or 0.1.0
    JUP_INSTALL_DIR     Installation directory; defaults to %USERPROFILE%\.javaup
    JUP_NO_MODIFY_PATH  Skip the user PATH update when set to a non-empty value
#>

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'

# PowerShell 5.1 on older Windows may not negotiate TLS 1.2 by default.
[Net.ServicePointManager]::SecurityProtocol =
  [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

$Repository = 'codeboyzhou/javaup'
$ReleaseBase = "https://github.com/$Repository/releases"
$ApiBase = "https://api.github.com/repos/$Repository"
$JupVersion = $env:JUP_VERSION
$JupInstallDir = if ($env:JUP_INSTALL_DIR) {
  $env:JUP_INSTALL_DIR
} else {
  Join-Path $env:USERPROFILE '.javaup'
}
$JupNoModifyPath = $env:JUP_NO_MODIFY_PATH
$RequestHeaders = @{
  Accept = 'application/vnd.github+json'
  'User-Agent' = 'javaup-installer'
}

function Write-Step([string]$Message) {
  Write-Host "==> $Message" -ForegroundColor Cyan
}

function Stop-Install([string]$Message) {
  throw $Message
}

function Get-Architecture {
  $rawArchitecture = try {
    [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()
  } catch {
    if ($env:PROCESSOR_ARCHITEW6432) {
      $env:PROCESSOR_ARCHITEW6432
    } else {
      $env:PROCESSOR_ARCHITECTURE
    }
  }

  switch ($rawArchitecture) {
    'X64' { return 'amd64' }
    'AMD64' { return 'amd64' }
    'Arm64' { return 'arm64' }
    'ARM64' { return 'arm64' }
    default { Stop-Install "unsupported Windows architecture: $rawArchitecture" }
  }
}

function Resolve-Version {
  if ($JupVersion) {
    $tag = $JupVersion.Trim()
    if (-not $tag.StartsWith('v')) {
      $tag = "v$tag"
    }
    if ($tag -notmatch '^v[0-9]+\.[0-9]+\.[0-9]+$') {
      Stop-Install "invalid release version: $JupVersion"
    }
    Write-Step "Using requested version $tag"
    return $tag
  }

  Write-Step 'Resolving the latest GitHub release'
  $release = Invoke-RestMethod -Uri "$ApiBase/releases/latest" -Headers $RequestHeaders
  $tag = [string]$release.tag_name
  if ($tag -notmatch '^v[0-9]+\.[0-9]+\.[0-9]+$') {
    Stop-Install "latest release has an invalid tag: $tag"
  }
  Write-Step "Latest version: $tag"
  return $tag
}

function Get-ExpectedChecksum([string]$ChecksumFile, [string]$ArchiveName) {
  $escapedName = [regex]::Escape($ArchiveName)
  foreach ($line in Get-Content -LiteralPath $ChecksumFile) {
    if ($line -match "^([a-fA-F0-9]{64})\s+\*?$escapedName$") {
      return $Matches[1].ToLowerInvariant()
    }
  }
  Stop-Install "checksum for $ArchiveName was not found"
}

function Test-Checksum([string]$File, [string]$Expected) {
  $actual = (Get-FileHash -LiteralPath $File -Algorithm SHA256).Hash.ToLowerInvariant()
  if ($actual -ne $Expected) {
    Stop-Install "checksum mismatch: expected $Expected, got $actual"
  }
}

function Add-ToUserPath([string]$Directory) {
  if ($JupNoModifyPath) {
    Write-Step 'Skipping PATH update because JUP_NO_MODIFY_PATH is set'
    return
  }

  $normalizedDirectory = [System.IO.Path]::GetFullPath($Directory).TrimEnd('\')
  $current = [Environment]::GetEnvironmentVariable('Path', 'User')
  foreach ($entry in @($current -split ';')) {
    if (-not $entry) {
      continue
    }
    try {
      $normalizedEntry = [System.IO.Path]::GetFullPath($entry).TrimEnd('\')
    } catch {
      continue
    }
    if ([string]::Equals($normalizedEntry, $normalizedDirectory, [StringComparison]::OrdinalIgnoreCase)) {
      Write-Step "$normalizedDirectory is already in the user PATH"
      return
    }
  }

  $newPath = if ($current) { "$normalizedDirectory;$current" } else { $normalizedDirectory }
  [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
  $env:Path = "$normalizedDirectory;$env:Path"
  Write-Step "Added $normalizedDirectory to the user PATH"
}

function Install-Binary([string]$Source, [string]$Destination) {
  $directory = Split-Path -Parent $Destination
  New-Item -ItemType Directory -Path $directory -Force | Out-Null

  $identifier = [guid]::NewGuid().ToString('N')
  $staged = Join-Path $directory ".jup-$identifier.tmp.exe"
  $backup = Join-Path $directory ".jup-$identifier.bak.exe"
  Copy-Item -LiteralPath $Source -Destination $staged

  try {
    if (Test-Path -LiteralPath $Destination) {
      Move-Item -LiteralPath $Destination -Destination $backup
    }
    Move-Item -LiteralPath $staged -Destination $Destination
    if (Test-Path -LiteralPath $backup) {
      Remove-Item -LiteralPath $backup -Force
    }
  } catch {
    if (-not (Test-Path -LiteralPath $Destination) -and (Test-Path -LiteralPath $backup)) {
      Move-Item -LiteralPath $backup -Destination $Destination
    }
    Remove-Item -LiteralPath $staged -Force -ErrorAction SilentlyContinue
    throw
  }
}

try {
  if ($env:OS -ne 'Windows_NT') {
    Stop-Install 'this installer supports Windows only'
  }

  $architecture = Get-Architecture
  $tag = Resolve-Version
  $version = $tag.Substring(1)
  $archiveName = "javaup-$version-windows-$architecture.zip"
  $downloadBase = "$ReleaseBase/download/$tag"
  $temporary = Join-Path ([System.IO.Path]::GetTempPath()) ("javaup-install-" + [guid]::NewGuid().ToString('N'))
  New-Item -ItemType Directory -Path $temporary | Out-Null

  try {
    $archive = Join-Path $temporary $archiveName
    $checksums = Join-Path $temporary 'checksums.txt'

    Write-Step "Downloading $archiveName"
    Invoke-WebRequest -UseBasicParsing -Uri "$downloadBase/$archiveName" -Headers $RequestHeaders -OutFile $archive
    Invoke-WebRequest -UseBasicParsing -Uri "$downloadBase/checksums.txt" -Headers $RequestHeaders -OutFile $checksums

    Write-Step 'Verifying SHA-256 checksum'
    $expected = Get-ExpectedChecksum $checksums $archiveName
    Test-Checksum $archive $expected

    $expanded = Join-Path $temporary 'expanded'
    Expand-Archive -LiteralPath $archive -DestinationPath $expanded
    $binaries = @(Get-ChildItem -LiteralPath $expanded -Filter 'jup.exe' -File -Recurse)
    if ($binaries.Count -ne 1) {
      Stop-Install "release archive contains $($binaries.Count) jup.exe files, expected 1"
    }

    $binDirectory = Join-Path $JupInstallDir 'bin'
    $destination = Join-Path $binDirectory 'jup.exe'
    Install-Binary $binaries[0].FullName $destination
    Add-ToUserPath $binDirectory

    Write-Step "Installed jup $tag to $destination"
    Write-Step 'Run: jup version'
  } finally {
    Remove-Item -LiteralPath $temporary -Recurse -Force -ErrorAction SilentlyContinue
  }
} catch {
  [Console]::Error.WriteLine("error: $($_.Exception.Message)")
  exit 1
}
