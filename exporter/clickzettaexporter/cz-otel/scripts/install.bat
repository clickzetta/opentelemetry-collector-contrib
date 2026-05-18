@echo off
REM cz-otel remote installer script for Windows
REM Usage (PowerShell):
REM   irm https://raw.githubusercontent.com/clickzetta/opentelemetry-collector-contrib/main/exporter/clickzettaexporter/cz-otel/scripts/install.ps1 | iex
REM
REM Or run this .bat directly after downloading.
REM
REM Environment variables:
REM   CZ_OTEL_VERSION  - Version to install (default: latest)
REM   INSTALL_DIR      - Installation directory (default: %APPDATA%\cz-otel)

setlocal enabledelayedexpansion

REM --- Configuration ---
set "REPO=clickzetta/opentelemetry-collector-contrib"
set "BINARY_NAME=cz-otel"
set "COLLECTOR_NAME=otelcol-clickzetta"

if defined INSTALL_DIR (
    set "DEST=%INSTALL_DIR%"
) else (
    set "DEST=%APPDATA%\cz-otel"
)

REM --- Detect architecture ---
set "ARCH=amd64"
if "%PROCESSOR_ARCHITECTURE%"=="ARM64" set "ARCH=arm64"

REM --- Determine version ---
if defined CZ_OTEL_VERSION (
    set "VERSION=%CZ_OTEL_VERSION%"
    echo ==> Using specified version: %VERSION%
) else (
    echo ==> Fetching latest version...
    for /f "tokens=*" %%i in ('powershell -NoProfile -Command "(Invoke-RestMethod -Uri 'https://api.github.com/repos/%REPO%/releases/latest').tag_name -replace '^v',''"') do set "VERSION=%%i"
    if "!VERSION!"=="" (
        echo Error: Failed to determine latest version. Set CZ_OTEL_VERSION manually.
        exit /b 1
    )
    echo ==> Latest version: !VERSION!
)

REM --- Build download URL ---
set "PACKAGE_NAME=%BINARY_NAME%-!VERSION!-windows-%ARCH%.tar.gz"
set "DOWNLOAD_URL=https://github.com/%REPO%/releases/download/v!VERSION!/%PACKAGE_NAME%"

REM --- Create temp directory ---
set "TMP_DIR=%TEMP%\cz-otel-install-%RANDOM%"
mkdir "%TMP_DIR%" 2>nul

echo ==> Downloading %DOWNLOAD_URL%...
powershell -NoProfile -Command "Invoke-WebRequest -Uri '%DOWNLOAD_URL%' -OutFile '%TMP_DIR%\%PACKAGE_NAME%'"
if errorlevel 1 (
    echo Error: Download failed.
    rmdir /s /q "%TMP_DIR%" 2>nul
    exit /b 1
)

REM --- Extract ---
echo ==> Extracting...
powershell -NoProfile -Command "tar -xzf '%TMP_DIR%\%PACKAGE_NAME%' -C '%TMP_DIR%'"
if errorlevel 1 (
    echo Error: Extraction failed.
    rmdir /s /q "%TMP_DIR%" 2>nul
    exit /b 1
)

REM --- Install ---
echo ==> Installing to %DEST%...
if not exist "%DEST%\bin\" mkdir "%DEST%\bin"
if not exist "%DEST%\logs\" mkdir "%DEST%\logs"

REM Find and copy binaries
set "FOUND=0"
for /d %%d in ("%TMP_DIR%\%BINARY_NAME%-*") do (
    if exist "%%d\bin\%BINARY_NAME%.exe" (
        copy /Y "%%d\bin\%BINARY_NAME%.exe" "%DEST%\bin\%BINARY_NAME%.exe" >nul
        set "FOUND=1"
    )
    if exist "%%d\bin\%COLLECTOR_NAME%.exe" (
        copy /Y "%%d\bin\%COLLECTOR_NAME%.exe" "%DEST%\bin\%COLLECTOR_NAME%.exe" >nul
    )
)

if "%FOUND%"=="0" (
    REM Try top-level
    if exist "%TMP_DIR%\bin\%BINARY_NAME%.exe" (
        copy /Y "%TMP_DIR%\bin\%BINARY_NAME%.exe" "%DEST%\bin\%BINARY_NAME%.exe" >nul
        set "FOUND=1"
    )
    if exist "%TMP_DIR%\bin\%COLLECTOR_NAME%.exe" (
        copy /Y "%TMP_DIR%\bin\%COLLECTOR_NAME%.exe" "%DEST%\bin\%COLLECTOR_NAME%.exe" >nul
    )
)

REM --- Cleanup temp ---
rmdir /s /q "%TMP_DIR%" 2>nul

REM --- Verify ---
if not exist "%DEST%\bin\%BINARY_NAME%.exe" (
    echo Error: Installation failed - %BINARY_NAME%.exe not found.
    exit /b 1
)

echo.
echo ==> Installation complete!
echo.
echo To use cz-otel, add the following directory to your PATH:
echo.
echo   %DEST%\bin
echo.
echo You can do this by running (in PowerShell as admin):
echo   [Environment]::SetEnvironmentVariable("Path", $env:Path + ";%DEST%\bin", "User")
echo.
echo Or manually via:
echo   Settings ^> System ^> About ^> Advanced system settings ^> Environment Variables
echo.
echo Next steps:
echo   1. Run 'cz-otel config init' to configure your ClickZetta connection
echo   2. Run 'cz-otel start' to launch the collector
echo.

endlocal
