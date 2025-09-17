@echo off
setlocal

set APP_NAME=onyrat
set BIN_DIR=bin

:all
call :client
call :server
goto :eof

:client

garble -seed=random -literals -tiny build -tags=client -o "%BIN_DIR%\%APP_NAME%-client.exe" .\cmd\client

goto :eof

:server

garble -seed=random -literals -tiny build -tags=server -o "%BIN_DIR%\%APP_NAME%-server.exe" .\cmd\server

goto :eof

:clean
rmdir /S /Q "%BIN_DIR%"
goto :eof
