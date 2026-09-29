@echo off
rem Runs Parley's translation benchmark (offline vs DeepL). Results go to the benchmark folder.
cd /d "%~dp0"
start "Parley benchmark" Parley.exe --benchmark
