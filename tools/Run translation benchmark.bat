@echo off
rem Runs Parley's translation benchmark (offline vs DeepL). Results go to the benchmark folder.
cd /d "%~dp0"
rem Put this next to Parley.exe (or run it from tools\ after build.bat).
if exist Parley.exe (start "Parley benchmark" Parley.exe --benchmark) else (start "Parley benchmark" ..\Parley.exe --benchmark)
