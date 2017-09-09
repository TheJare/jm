@echo off
set JM_VER=1.1.1

del /S /Q dist\ 2> nul
rd /S /Q dist\ 2> nul

set GOOS=windows
go build -ldflags="-s -w" -o dist\%GOOS%\jm.exe
copy README.md dist\%GOOS%\
powershell Compress-Archive -Path dist\%GOOS%\* -DestinationPath dist\jm-%JM_VER%-%GOOS%-x64.zip -Force

set GOOS=linux
go build -ldflags="-s -w" -o dist\%GOOS%\jm
copy README.md dist\%GOOS%\
powershell Compress-Archive -Path dist\%GOOS%\* -DestinationPath dist\jm-%JM_VER%-%GOOS%-x64.zip -Force

set GOOS=darwin
go build -ldflags="-s -w" -o dist\%GOOS%\jm
copy README.md dist\%GOOS%\
powershell Compress-Archive -Path dist\%GOOS%\* -DestinationPath dist\jm-%JM_VER%-%GOOS%-x64.zip -Force
