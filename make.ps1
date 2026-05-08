param(
    [string]$Target = "build"
)

$projectPath = $PSScriptRoot
$mockgen = "$env:USERPROFILE\go\bin\mockgen.exe"

switch ($Target) {
    "build" {
        Push-Location $projectPath
        try {
            Write-Host "[1/2] Generate mocks..."
            & $mockgen "-source=internal/service/service.go" "-destination=internal/service/mocks/user_repo_mock.go" -package=mocks 2>$null
            Write-Host "[2/2] Build app..."
            go build -o ./cmd/gophermart/gophermart ./cmd/gophermart
        } finally {
            Pop-Location
        }
        Write-Host "[OK] Done!"
    }
    "generate-mocks" {
        Push-Location $projectPath
        try {
            Write-Host "[INFO] Generate mocks..."
            & $mockgen "-source=internal/service/service.go" "-destination=internal/service/mocks/user_repo_mock.go" -package=mocks
        } finally {
            Pop-Location
        }
        Write-Host "[OK] Done!"
    }
    "tests" {
        Push-Location $projectPath
        try {
            Write-Host "[INFO] Run unit tests..."
            go test -v ./internal/service/...
        } finally {
            Pop-Location
        }
    }
    "docker-up" {
        Write-Host "[INFO] Start docker compose..."
        Push-Location $projectPath
        try {
            docker compose up -d
        } finally {
            Pop-Location
        }
    }
    "docker-down" {
        Write-Host "[INFO] Stop docker compose..."
        Push-Location $projectPath
        try {
            docker compose down
        } finally {
            Pop-Location
        }
    }
    "docker-it" {
        Write-Host "[INFO] Run integration tests via docker compose..."
        if (-not (docker --version 2>$null)) {
            Write-Host "[ERROR] Docker не установлен" -ForegroundColor Red
            exit 1
        }
        Push-Location $projectPath
        docker compose run --rm integration_tests
        $exitCode = $LASTEXITCODE
        Pop-Location
        exit $exitCode
    }
    "clean" {
        Write-Host "[INFO] Clean..."
        Remove-Item -Path "$projectPath\internal\service\mocks" -Recurse -Force -ErrorAction SilentlyContinue
        Remove-Item -Path "$projectPath\cmd\gophermart\gophermart" -Force -ErrorAction SilentlyContinue
        Write-Host "[OK] Done!"
    }
    "all" {
        Push-Location $projectPath
        try {
            Write-Host "[INFO] Full build..."
            Remove-Item -Path "./internal/service/mocks" -Recurse -Force -ErrorAction SilentlyContinue
            Remove-Item -Path "./cmd/gophermart/gophermart" -Force -ErrorAction SilentlyContinue
            & $mockgen "-source=internal/service/service.go" "-destination=internal/service/mocks/user_repo_mock.go" -package=mocks 2>$null
            go build -o ./cmd/gophermart/gophermart ./cmd/gophermart
            go test -v ./internal/service/...
        } finally {
            Pop-Location
        }
        Write-Host "[OK] Done!"
    }
    default {
        Write-Host "[INFO] Usage: .\make.ps1 target"
        Write-Host "Available targets: build, generate-mocks, tests, docker-up, docker-down, docker-it, clean, all"
        Write-Host "Integration tests: .\make.ps1 docker-it"
    }
}
