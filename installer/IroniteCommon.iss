// Shared [Code] for every Ironite setup. Included from the [Code] section.
// (c) 2026 Ironite

// Saved games, chip balances, stats and settings live in the user's
// AppData\Roaming\Ironite folder, never in the install folder. Uninstalling
// keeps them unless the player explicitly chooses to delete them here.
function IroniteAskDeleteSaves(const What: string): Boolean;
begin
  Result := False;
  if UninstallSilent then
    Exit;
  Result := SuppressibleMsgBox(
    'Do you also want to delete ' + What + '?' + #13#10#13#10 +
    'They are stored in your user folder (AppData\Roaming\Ironite), not with the program, ' +
    'so they are kept unless you choose Yes. If you reinstall later, everything will be right where you left it.' + #13#10#13#10 +
    'Choose Yes to delete them, or No to keep them.',
    mbConfirmation, MB_YESNO or MB_DEFBUTTON2, IDNO) = IDYES;
end;

procedure IroniteDeleteFolder(const Path: string);
begin
  if DirExists(Path) then
  begin
    Log('Deleting ' + Path);
    DelTree(Path, True, True, True);
  end;
end;
