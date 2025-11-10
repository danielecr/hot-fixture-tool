; Hot Fixture Tool Client (hfit) Installer
; Simple NSIS installer for Windows testing environments

!define APPNAME "Hot Fixture Tool Client"
!define COMPANYNAME "Daniele Cruciani"
!define DESCRIPTION "Command-line client for database fixture testing"
!define VERSION "{{.Version}}"
!define VERSIONMAJOR "{{.Major}}"
!define VERSIONMINOR "{{.Minor}}" 
!define VERSIONBUILD "{{.Patch}}"
!define HELPURL "https://github.com/danielecr/hot-fixture-tool"
!define UPDATEURL "https://github.com/danielecr/hot-fixture-tool/releases"
!define ABOUTURL "https://github.com/danielecr/hot-fixture-tool"

; Request admin rights for system-wide installation
RequestExecutionLevel admin

; Modern UI
!include "MUI2.nsh"

; Installer settings
Name "${APPNAME}"
OutFile "hfit-setup.exe"
InstallDir "$PROGRAMFILES\${APPNAME}"
InstallDirRegKey HKLM "Software\${APPNAME}" "InstallLocation"

; Interface Settings
!define MUI_ABORTWARNING

; Pages
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "LICENSE"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

; Uninstaller pages
!insertmacro MUI_UNPAGE_WELCOME
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_UNPAGE_FINISH

; Languages
!insertmacro MUI_LANGUAGE "English"

; Main install section
Section "Install"
    ; Set output path to installation directory
    SetOutPath $INSTDIR
    
    ; Install files
    File "hfit.exe"
    File "LICENSE"
    File "README.md"
    
    ; Add to PATH environment variable
    EnVar::SetHKLM
    EnVar::AddValue "PATH" "$INSTDIR"
    
    ; Create uninstaller
    WriteUninstaller "$INSTDIR\uninstall.exe"
    
    ; Add to Programs and Features
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "DisplayName" "${APPNAME}"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "UninstallString" "$\"$INSTDIR\uninstall.exe$\""
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "QuietUninstallString" "$\"$INSTDIR\uninstall.exe$\" /S"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "InstallLocation" "$\"$INSTDIR$\""
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "DisplayIcon" "$\"$INSTDIR\hfit.exe$\""
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "Publisher" "${COMPANYNAME}"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "HelpLink" "${HELPURL}"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "URLUpdateInfo" "${UPDATEURL}"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "URLInfoAbout" "${ABOUTURL}"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "DisplayVersion" "${VERSION}"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "VersionMajor" "${VERSIONMAJOR}"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "VersionMinor" "${VERSIONMINOR}"
    WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "NoModify" 1
    WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "NoRepair" 1
    
    ; Store installation location
    WriteRegStr HKLM "Software\${APPNAME}" "InstallLocation" "$INSTDIR"
    
    ; Success message
    MessageBox MB_OK "hfit has been installed successfully!$\n$\nYou can now use 'hfit' command from any Command Prompt or PowerShell window."
SectionEnd

; Uninstaller section
Section "Uninstall"
    ; Remove files
    Delete "$INSTDIR\hfit.exe"
    Delete "$INSTDIR\LICENSE"
    Delete "$INSTDIR\README.md"
    Delete "$INSTDIR\uninstall.exe"
    
    ; Remove from PATH
    EnVar::SetHKLM
    EnVar::DeleteValue "PATH" "$INSTDIR"
    
    ; Remove directory
    RMDir "$INSTDIR"
    
    ; Remove registry entries
    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}"
    DeleteRegKey HKLM "Software\${APPNAME}"
SectionEnd