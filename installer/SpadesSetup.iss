; Spades - standalone setup. Build: bash build.sh, then ISCC.exe /DGameVersion=<version> installer\SpadesSetup.iss
; (c) 2026 Ironite
#define GameName    "Spades"
#define Bundle      "Spades"
#define Exe         "Spades.exe"
#ifndef GameVersion
  #define GameVersion "1.0.0"
#endif
#define AppGuid     "{{E447DB56-3A88-4218-A995-80DEFE45A448}"
#define Art         "spades"
#define Tagline     "A Brooklyn rooftop card game. Partner up against the rooftop regulars, share one PC, or play friends online."
#define SaveWhat    "Spades record and settings"
#include "GameSetup.iss"

[Run]
; Let Spades host online games without a firewall prompt.
Filename: "{sys}\netsh.exe"; Parameters: "advfirewall firewall add rule name=""Spades"" dir=in action=allow program=""{app}\Games\{#Bundle}\{#Exe}"" enable=yes"; Flags: runhidden; StatusMsg: "Letting Spades host online games..."

[UninstallRun]
Filename: "{sys}\netsh.exe"; Parameters: "advfirewall firewall delete rule name=""Spades"""; Flags: runhidden; RunOnceId: "SpadesFirewall"
