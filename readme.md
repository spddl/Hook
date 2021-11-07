# Hook

Hook uses only the Win32 API [SetWinEventHook](https://docs.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-setwineventhook) to be notified when programs are opened, closed or come to the foreground.
You can start other programs/scripts from the config.

## example config's

### changes the power plan when the game is started
```toml
[[games]]
exe = 'Game.exe'

# Powerplan relies on high performance
[[games.OnProcessStart]]
Name = 'powercfg.exe'
Args = '/setactive 8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c'
HideWindow = true
OnForeground = true # is also taken over when the window gets the focus

# Powerplan relies on Balanced
[[games.OnProcessFinish]]
Name = 'powercfg.exe'
Args = '/setactive 381b4222-f694-41f0-9685-ff5bb260df2e'
HideWindow = true
OnBackground = true # is also adopted if the window loses focus
```

### kills dwm when the game is started (requires [GoSetProcessAffinity](https://github.com/spddl/GoSetProcessAffinity))
```toml
[[games]]
exe = 'Game.exe'

[[games.OnProcessStart]]
Name = 'GoSetProcessAffinity.exe'
Args = '-pid %pid% -priorityClass 4 -ioPriority 3 -memoryPriority 5' # %pid% is replaced with the ProcessID

[[games.OnProcessStart]]
Name = 'GoSetProcessAffinity.exe'
Args = '-pid %pid% -boost'
OnForeground = true

# prevents dwm from being started again
[[games.OnProcessStart]]
Name = 'GoSetProcessAffinity.exe'
Args = '-proc winlogon.exe -suspend'

# kills dwm after 3 seconds
[[games.OnProcessStart]]
Name = 'powershell.exe'
Args = '-NoProfile -NonInteractive -Command &{Start-Sleep -s 3; Get-Process -Name "dwm" | Stop-Process -Force | Out-Null}'

# allows dwm to be started again
[[games.OnProcessFinish]]
Name = 'GoSetProcessAffinity.exe'
Args = '-proc winlogon.exe -resume'

[[games.OnProcessFinish]]
Name = 'GoSetProcessAffinity.exe'
Args = '-proc dwm.exe -priorityClass 0 -ioPriority 0 -memoryPriority 1'
```
