[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'

$docsDirectory = Split-Path -Parent $PSScriptRoot
$themeVersion = (Get-Content -Raw (Join-Path $docsDirectory 'THEME_VERSION')).Trim()
$themeSha256 = (Get-Content -Raw (Join-Path $docsDirectory 'THEME_SHA256')).Trim()
$themesDirectory = Join-Path $docsDirectory 'themes'
$themeDirectory = Join-Path $themesDirectory 'hugo-geekdoc'
$versionFile = Join-Path $themeDirectory '.javaup-theme-version'

if ((Test-Path -LiteralPath $versionFile) -and
    ((Get-Content -Raw $versionFile).Trim() -eq $themeVersion)) {
    Write-Output "hugo-geekdoc $themeVersion is already installed."
    exit 0
}

if ($themeVersion -notmatch '^v[0-9]') {
    throw "Invalid theme version: $themeVersion"
}

$temporaryDirectory = Join-Path ([IO.Path]::GetTempPath()) (
    'javaup-geekdoc-' + [guid]::NewGuid().ToString('N')
)
$archive = Join-Path $temporaryDirectory 'hugo-geekdoc.tar.gz'
$stagedTheme = Join-Path $temporaryDirectory 'hugo-geekdoc'
$downloadUrl = "https://github.com/thegeeklab/hugo-geekdoc/releases/download/$themeVersion/hugo-geekdoc.tar.gz"

try {
    New-Item -ItemType Directory -Path $stagedTheme -Force | Out-Null
    New-Item -ItemType Directory -Path $themesDirectory -Force | Out-Null
    Invoke-WebRequest -Uri $downloadUrl -OutFile $archive
    $actualSha256 = (Get-FileHash -Algorithm SHA256 $archive).Hash.ToLowerInvariant()
    if ($actualSha256 -ne $themeSha256) {
        throw "Theme checksum mismatch: expected $themeSha256, got $actualSha256"
    }
    tar -xzf $archive -C $stagedTheme

    $resolvedThemesDirectory = [IO.Path]::GetFullPath($themesDirectory)
    $resolvedThemeDirectory = [IO.Path]::GetFullPath($themeDirectory)
    if ((Split-Path -Parent $resolvedThemeDirectory) -ne $resolvedThemesDirectory) {
        throw "Refusing to replace unexpected theme path: $resolvedThemeDirectory"
    }

    if (Test-Path -LiteralPath $resolvedThemeDirectory) {
        Remove-Item -LiteralPath $resolvedThemeDirectory -Recurse -Force
    }
    Move-Item -LiteralPath $stagedTheme -Destination $resolvedThemeDirectory
    Set-Content -LiteralPath $versionFile -Value $themeVersion
    Write-Output "Installed hugo-geekdoc $themeVersion."
}
finally {
    $resolvedTemporaryDirectory = [IO.Path]::GetFullPath($temporaryDirectory)
    $systemTemporaryDirectory = [IO.Path]::GetFullPath([IO.Path]::GetTempPath())
    if ($resolvedTemporaryDirectory.StartsWith($systemTemporaryDirectory) -and
        (Test-Path -LiteralPath $resolvedTemporaryDirectory)) {
        Remove-Item -LiteralPath $resolvedTemporaryDirectory -Recurse -Force
    }
}
