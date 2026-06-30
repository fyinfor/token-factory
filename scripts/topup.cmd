@echo off
chcp 65001 >nul 2>&1
cd /d "%~dp0.."

if "%~1"=="" (
  set /p TF_USER=用户名: 
) else (
  set TF_USER=%~1
)

if "%~2"=="" (
  set /p TF_USD=充值美元: 
) else (
  set TF_USD=%~2
)

if "%TF_USER%"=="" goto usage
if "%TF_USD%"=="" goto usage

go run ./scripts/sim-topup "%TF_USER%" "%TF_USD%"
set EXIT_CODE=%ERRORLEVEL%
if %EXIT_CODE% neq 0 exit /b %EXIT_CODE%
exit /b 0

:usage
echo.
echo 用法: scripts\topup.cmd 用户名 充值美元
echo 示例: scripts\topup.cmd test1 10
echo 或直接双击本文件，按提示输入
exit /b 1
