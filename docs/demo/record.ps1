[CmdletBinding()]
param(
    [switch]$SkipBuild
)

$ErrorActionPreference = 'Stop'
$image = 'javaup-demo'
$repositoryRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path

docker version --format '{{.Server.Version}}' *> $null
if ($LASTEXITCODE -ne 0) {
    throw 'Docker is not available. Start Docker Desktop and try again.'
}

Push-Location $repositoryRoot
try {
    if (-not $SkipBuild) {
        Write-Host 'Building the javaup VHS recording image...'
        docker build --file docs/demo/Dockerfile --tag $image .
        if ($LASTEXITCODE -ne 0) {
            throw 'Failed to build the javaup VHS recording image.'
        }
    }

    Write-Host 'Recording docs/demo/demo.gif...'
    docker run --rm --volume "${repositoryRoot}:/jup" $image docs/demo/demo.tape
    if ($LASTEXITCODE -ne 0) {
        throw 'Failed to record docs/demo/demo.gif.'
    }

    Write-Host "Recorded $repositoryRoot\docs\demo.gif"
}
finally {
    Pop-Location
}
