@echo off

for /f "usebackq delims=" %%i in (`"C:\PROGRA~1\Git\usr\bin\date.exe +^"%%Y.%%m.%%d %%H:%%M:%%S^"`) do set BUILDTIME=%%i
for /F %%i in ('git rev-parse --short HEAD') do set REV=%%i

go build -ldflags "-X main.buildtime '%BUILDTIME%' -X main.version '%REV%'"