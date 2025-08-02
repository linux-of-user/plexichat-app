@echo off
echo 🚀 PlexiChat Desktop - Quick Deploy
echo ===================================

echo Building application...
go build -o plexichat.exe .
if errorlevel 1 (
    echo ❌ Build failed
    pause
    exit /b 1
)
echo ✅ Build successful

echo Testing application...
plexichat.exe --version
echo.

echo Initializing Git...
git init
git add .
git commit -m "feat: PlexiChat Desktop - The Discord Killer"

echo Creating GitHub repo...
gh repo create plexichat-desktop --public --description "🚀 Modern Discord-killer desktop app for PlexiChat"
git branch -M main
git remote add origin https://github.com/%USERNAME%/plexichat-desktop.git
git push -u origin main

echo Creating release...
gh release create v2.0.0-alpha --title "🚀 PlexiChat Desktop v2.0.0-alpha" --notes "The Discord Killer is here!" --prerelease plexichat.exe

echo ✅ Deployment complete!
echo Repository: https://github.com/%USERNAME%/plexichat-desktop
pause
