#define MyAppName "NaviServer"
#define MyAppVersion GetEnv("MYAPP_VERSION")
#define MyAppPublisher "Andre Carbajal"
#define MyAppCopyright "Copyright (C) 2026 Andre Carbajal"

[Setup]
AppId={{628b9b2c-84a9-4010-9a9c-10f3b32b538c}}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL=https://github.com/andre-carbajal/NaviServer
AppCopyright={#MyAppCopyright}
DefaultDirName={autopf}\{#MyAppName}
DefaultGroupName={#MyAppName}
OutputBaseFilename=NaviServer-{#MyAppVersion}-windows
Compression=lzma
SolidCompression=yes
PrivilegesRequired=admin
CloseApplications=force
VersionInfoVersion={#MyAppVersion}
VersionInfoCompany={#MyAppPublisher}
VersionInfoDescription={#MyAppName} Installer
VersionInfoTextVersion={#MyAppVersion}
VersionInfoCopyright={#MyAppCopyright}
VersionInfoProductName={#MyAppName}
VersionInfoProductVersion={#MyAppVersion}

SetupIconFile=cmd\server\icon.ico

[Files]
Source: "dist\naviserver-server.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "dist\naviserver-cli.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "cmd\server\icon.ico"; DestDir: "{app}"; Flags: ignoreversion
Source: "dist\web_dist\*"; DestDir: "{app}\web_dist"; Flags: ignoreversion recursesubdirs

[Icons]
Name: "{group}\{#MyAppName}"; Filename: "{app}\naviserver-server.exe"; IconFilename: "{app}\icon.ico"
Name: "{autodesktop}\{#MyAppName}"; Filename: "{app}\naviserver-server.exe"; IconFilename: "{app}\icon.ico"; Tasks: desktopicon

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"

[Registry]
Root: HKLM; Subkey: "SYSTEM\CurrentControlSet\Control\Session Manager\Environment"; ValueType: expandsz; ValueName: "Path"; ValueData: "{olddata};{app}"; Check: NeedsAddPath('{app}')

[Run]
Filename: "{app}\naviserver-server.exe"; Description: "Start NaviServer"; Flags: postinstall nowait skipifsilent

[UninstallRun]
Filename: "taskkill"; Parameters: "/IM naviserver-server.exe /F"; Flags: runhidden; StatusMsg: "Stopping NaviServer..."

[UninstallDelete]
Type: filesandordirs; Name: "{app}\web_dist"
Type: files; Name: "{app}\*"
Type: dirifempty; Name: "{app}"

[Code]
function NeedsAddPath(Param: string): boolean;
var
  Path: string;
begin
  if RegQueryStringValue(HKEY_LOCAL_MACHINE, 'SYSTEM\CurrentControlSet\Control\Session Manager\Environment', 'Path', Path) then
  begin
    Result := Pos(Uppercase(Param), Uppercase(Path)) = 0;
  end else Result := True;
end;