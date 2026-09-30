@echo off
rem Builds Parley.exe and Parley-Setup.exe (needs Go 1.24+ from https://go.dev/dl/).
rem bin\parley-mt.exe (offline translator) and bin\whisper-cli.exe are prebuilt; see engine\README.md to rebuild it.
cd /d "%~dp0"
go build -trimpath -ldflags "-s -w -H windowsgui" -o Parley.exe ./app
if errorlevel 1 (echo Build failed & pause & exit /b 1)
if not exist installer\payload mkdir installer\payload
copy /y Parley.exe installer\payload\ >nul
copy /y bin\whisper-cli.exe installer\payload\ >nul
copy /y bin\parley-mt.exe installer\payload\ >nul
copy /y bin\whisper-cli-LICENSE.txt installer\payload\ >nul
for %%f in (README.md INSTALL.txt LICENSE THIRD_PARTY_NOTICES.md PRIVACY.md CHANGELOG.md) do copy /y %%f installer\payload\ >nul
if not exist installer\payload\third_party mkdir installer\payload\third_party
copy /y third_party\* installer\payload\third_party\ >nul
if not exist installer\payload\addon\Parley mkdir installer\payload\addon\Parley
xcopy /e /i /y /q addon\Parley installer\payload\addon\Parley >nul
cd installer
go build -trimpath -ldflags "-s -w -H windowsgui" -o ..\Parley-Setup.exe .
if errorlevel 1 (echo Installer build failed & pause & exit /b 1)
cd ..
for /f "tokens=3" %%v in ('findstr /b "## Version:" addon\Parley\Parley.toc') do set ADDONVER=%%v
powershell -NoProfile -Command "Compress-Archive -Path addon\Parley -DestinationPath Parley-addon-%ADDONVER%.zip -Force"
echo Built Parley.exe, Parley-Setup.exe and Parley-addon-%ADDONVER%.zip
