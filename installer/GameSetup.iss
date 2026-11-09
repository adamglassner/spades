; Shared template for the standalone Ironite game setups.
; AppGuid is defined with Inno's escaped brace, e.g. "{{6DDF...}", which
; Inno reads as the literal AppId {6DDF...}.
; Don't compile this file directly: compile SpadesSetup.iss, which sets the
; #defines below and includes it.
;
; Each game installs into <Ironite folder>\Games\<Game>, the same place the
; combined Ironite Setup uses, so Ironite Launcher finds it automatically.
; Saves and settings stay in %APPDATA%\Ironite\<Game>.
;
; (c) 2026 Ironite

#if VER < EncodeVer(6,3,0)
  #error Inno Setup 6.3 or newer is required to build this installer.
#endif
#ifndef Bundle
  #error Compile SpadesSetup.iss, not this template.
#endif
#define Files "..\dist"

[Setup]
AppId={#AppGuid}
AppName={#GameName}
AppVersion={#GameVersion}
AppVerName={#GameName} v{#GameVersion}
AppPublisher=Ironite
AppCopyright=Copyright (c) 2026 Ironite. All rights reserved.
VersionInfoVersion={#GameVersion}
VersionInfoCompany=Ironite
VersionInfoProductName={#GameName}
VersionInfoDescription={#GameName} Setup
VersionInfoProductTextVersion=v{#GameVersion}
; Install next to Ironite Launcher when it's there, else in Program Files\Ironite.
DefaultDirName={reg:HKLM\Software\Ironite,InstallDir|{autopf}\Ironite}
DefaultGroupName=Ironite
DisableProgramGroupPage=yes
UsePreviousAppDir=yes
PrivilegesRequired=admin
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
MinVersion=10.0
WizardStyle=modern
WizardImageFile=assets\wizard-{#Art}-100.bmp,assets\wizard-{#Art}-150.bmp,assets\wizard-{#Art}-200.bmp
WizardSmallImageFile=assets\small-{#Art}-100.bmp,assets\small-{#Art}-150.bmp,assets\small-{#Art}-200.bmp
SetupIconFile=assets\{#Art}.ico
UninstallDisplayIcon={app}\Games\{#Bundle}\{#Exe}
UninstallDisplayName={#GameName}
OutputDir=..\Output
OutputBaseFilename={#Bundle}-Setup-v{#GameVersion}
Compression=lzma2/max
SolidCompression=yes
CloseApplications=yes
RestartApplications=no
SetupLogging=yes

[Messages]
SetupWindowTitle={#GameName} Setup v{#GameVersion}
WelcomeLabel1=Welcome to {#GameName} Setup v{#GameVersion}
WelcomeLabel2=This will install {#GameName} {#GameVersion} on your computer.%n%n{#Tagline}%n%nIf Ironite Launcher is installed, {#GameName} goes right next to it and shows up in the launcher, ready to play.%n%nYour saves and settings are kept in your user folder (AppData\Roaming\Ironite), so updates and reinstalls never wipe them.
FinishedHeadingLabel={#GameName} is ready to play
FinishedLabel={#GameName} is installed. Open it from the Start menu (Ironite folder) or from Ironite Launcher.

[Tasks]
Name: "desktopicon"; Description: "Create a &desktop shortcut for {#GameName}"; GroupDescription: "Shortcuts:"

[Files]
Source: "{#Files}\{#Bundle}\*"; DestDir: "{app}\Games\{#Bundle}"; Flags: ignoreversion sharedfile uninsnosharedfileprompt

[Icons]
Name: "{group}\{#GameName}"; Filename: "{app}\Games\{#Bundle}\{#Exe}"; WorkingDir: "{app}\Games\{#Bundle}"; Comment: "{#Tagline}"
Name: "{autodesktop}\{#GameName}"; Filename: "{app}\Games\{#Bundle}\{#Exe}"; WorkingDir: "{app}\Games\{#Bundle}"; Comment: "{#Tagline}"; Tasks: desktopicon

[Run]
Filename: "{app}\Games\{#Bundle}\{#Exe}"; Description: "Play {#GameName} now"; Flags: nowait postinstall skipifsilent runasoriginaluser

[UninstallRun]
Filename: "{sys}\taskkill.exe"; Parameters: "/F /T /IM {#Exe}"; Flags: runhidden; RunOnceId: "Stop{#Bundle}"

[UninstallDelete]
; The game's window cache (not saves: those live in AppData\Roaming\Ironite).
Type: filesandordirs; Name: "{localappdata}\Ironite\{#Bundle}"
Type: dirifempty; Name: "{localappdata}\Ironite"
Type: dirifempty; Name: "{app}\Games"
Type: dirifempty; Name: "{app}"

[Code]
#include "IroniteCommon.iss"

// When Ironite is already installed, the game always goes next to it.
function ShouldSkipPage(PageID: Integer): Boolean;
begin
  Result := (PageID = wpSelectDir) and RegValueExists(HKLM64, 'Software\Ironite', 'InstallDir');
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
begin
  if CurUninstallStep = usPostUninstall then
    if IroniteAskDeleteSaves('your {#SaveWhat}') then
      IroniteDeleteFolder(ExpandConstant('{userappdata}\Ironite\{#Bundle}'));
end;
