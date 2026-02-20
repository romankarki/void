@echo off
setlocal
set "VOID_LABEL=%~1"
if "%VOID_LABEL%"=="" set "VOID_LABEL=VOID"
endlocal & set "VOID_ACTIVE_LABEL=%VOID_LABEL%"
echo [void] Active prompt label: %VOID_ACTIVE_LABEL%
echo [void] Run "set VOID_ACTIVE_LABEL=" to clear it in this terminal.
