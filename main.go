package main

import (
	"log"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

// credits: https://github.com/GregoryDosh/automidically/blob/57c52df4621946c2665271361bf3248df7b9e1e2/internal/activewindow/activewindow.go

var (
	// listenerLock      = &sync.Mutex{}
	listener          *Listener
	lastForegroundApp string
)

type process struct {
	pid uint32
	exe string
}

type Listener struct {
	AllProcesses            map[win.HWND]process
	AllPIDs                 map[uint32]struct{}
	EVENT_OBJECT_CREATE     win.HWINEVENTHOOK
	EVENT_SYSTEM_FOREGROUND win.HWINEVENTHOOK
	EVENT_OBJECT_DESTROY    win.HWINEVENTHOOK
	mutex                   sync.Mutex
}

var STATUSCODES = map[uint32]string{
	win.EVENT_OBJECT_CREATE:     "EVENT_OBJECT_CREATE",
	win.EVENT_SYSTEM_FOREGROUND: "EVENT_SYSTEM_FOREGROUND",
	win.EVENT_OBJECT_DESTROY:    "EVENT_OBJECT_DESTROY",
}

const (
	OBJID_WINDOW = 0
	CHILDID_SELF = 0
	WM_APPEXIT   = 0x0400 + 1
)

// newActiveWindowCallback is passed to Windows to be called whenever the active window changes.
// When it is called it will attempt to find the process of an associated handle, then get the executable associated with that.
// https://docs.microsoft.com/en-us/windows/win32/api/winuser/nc-winuser-wineventproc
func (l *Listener) newActiveWindowCallback(
	hWinEventHook win.HWINEVENTHOOK, // Handle to an event hook function. This value is returned by SetWinEventHook when the hook function is installed and is specific to each instance of the hook function.
	event uint32, // Specifies the event that occurred. This value is one of the event constants.
	hwnd win.HWND, // Handle to the window that generates the event, or NULL if no window is associated with the event. For example, the mouse pointer is not associated with a window.
	idObject int32, // Identifies the object associated with the event. This is one of the object identifiers or a custom object ID.
	idChild int32, // Identifies whether the event was triggered by an object or a child element of the object. If this value is CHILDID_SELF, the event was triggered by the object; otherwise, this value is the child ID of the element that triggered the event.
	idEventThread uint32,
	dwmsEventTime uint32, // Specifies the time, in milliseconds, that the event was generated.
) (ret uintptr) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if idObject != OBJID_WINDOW || idChild != CHILDID_SELF {
		return
	}

	val, ok := l.AllProcesses[hwnd]
	if ok {
		if event == win.EVENT_OBJECT_DESTROY {
			_, oldPid := l.AllPIDs[val.pid]
			delete(l.AllPIDs, val.pid)
			if oldPid { // diese PID wird zum ersten mal destroyed
				OnProcessFinish(val.pid, val.exe)
			}
			delete(l.AllProcesses, hwnd)
		} else if event == win.EVENT_SYSTEM_FOREGROUND {
			if lastForegroundApp != val.exe {
				OnForeground(val.exe, val.pid)
				lastForegroundApp = val.exe
			}
		}
		return
	}

	if hwnd == 0 {
		return
	}

	if event == win.EVENT_OBJECT_DESTROY {
		// hwnd war noch nicht in der DB und muss da auch nicht mehr rein
		return
	}

	var pid uint32 = 0
	if v, ok := l.AllProcesses[hwnd]; ok {
		if v.pid != 0 {
			pid = v.pid
		}
	} else {
		win.GetWindowThreadProcessId(hwnd, &pid)
		if pid == 0 {
			return
		}
	}

	_, foundPid := l.AllPIDs[pid]
	if foundPid && event == win.EVENT_OBJECT_CREATE {
		return
	}

	pHndl, err := windows.OpenProcess(ProcessQueryInformation, false, pid)
	defer windows.CloseHandle(pHndl)
	if pHndl == 0 || err != nil {
		return
	}

	buf := make([]uint16, syscall.MAX_PATH)
	err = windows.GetModuleFileNameEx(pHndl, 0, &buf[0], uint32(len(buf)))
	if err != nil {
		return
	}
	processFilename := filepath.Base(strings.ToLower(syscall.UTF16ToString(buf)))

	if processFilename == "rundll32.exe" {
		return
	}

	l.AllPIDs[pid] = struct{}{}
	l.AllProcesses[hwnd] = process{
		pid: pid,
		exe: processFilename,
	}
	if event == win.EVENT_SYSTEM_FOREGROUND {
		if lastForegroundApp != processFilename {
			OnForeground(processFilename, pid)
			lastForegroundApp = processFilename
		}
	} else {
		OnProcessStart(pid, processFilename)
	}
	return 0
}

func startListenerMessageLoop() {
	var err error
	EVENT_OBJECT_CREATE, err := setActiveWindowWinEventHook(listener.newActiveWindowCallback, win.EVENT_OBJECT_CREATE)
	if err != nil {
		_ = err
		panic(err)
	}
	defer win.UnhookWinEvent(EVENT_OBJECT_CREATE)

	EVENT_SYSTEM_FOREGROUND, err := setActiveWindowWinEventHook(listener.newActiveWindowCallback, win.EVENT_SYSTEM_FOREGROUND)
	if err != nil {
		panic(err)
	}
	defer win.UnhookWinEvent(EVENT_SYSTEM_FOREGROUND)

	EVENT_OBJECT_DESTROY, err := setActiveWindowWinEventHook(listener.newActiveWindowCallback, win.EVENT_OBJECT_DESTROY)
	if err != nil {
		_ = err
		panic(err)
	}
	defer win.UnhookWinEvent(EVENT_OBJECT_DESTROY)

	var msg win.MSG
	for win.GetMessage(&msg, 0, 0, 0) != 0 {
		if msg.Message == WM_APPEXIT {
			break
		}
		win.TranslateMessage(&msg)
		win.DispatchMessage(&msg)
	}
}

// setActiveWindowWinEventHook is for informing windows which function should be called whenever a
// foreground window has changed. https://docs.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-setwineventhook
func setActiveWindowWinEventHook(callbackFunction win.WINEVENTPROC, event uint32) (win.HWINEVENTHOOK, error) {
	ret, err := win.SetWinEventHook(
		event,            // Specifies the event constant for the lowest event value in the range of events that are handled by the hook function. This parameter can be set to EVENT_MIN to indicate the lowest possible event value.
		event,            // Specifies the event constant for the highest event value in the range of events that are handled by the hook function. This parameter can be set to EVENT_MAX to indicate the highest possible event value.
		0,                // Handle to the DLL that contains the hook function at lpfnWinEventProc, if the WINEVENT_INCONTEXT flag is specified in the dwFlags parameter. If the hook function is not located in a DLL, or if the WINEVENT_OUTOFCONTEXT flag is specified, this parameter is NULL.
		callbackFunction, // Pointer to the event hook function. For more information about this function, see WinEventProc.
		0,                // Specifies the ID of the process from which the hook function receives events. Specify zero (0) to receive events from all processes on the current desktop.
		0,                // Specifies the ID of the thread from which the hook function receives events. If this parameter is zero, the hook function is associated with all existing threads on the current desktop.
		win.WINEVENT_OUTOFCONTEXT|win.WINEVENT_SKIPOWNPROCESS, // Flag values that specify the location of the hook function and of the events to be skipped. The following flags are valid:
	)
	if ret == 0 {
		return 0, err
	}

	return ret, nil
}

func main() {
	listener = &Listener{}
	listener.AllProcesses = make(map[win.HWND]process)
	listener.AllPIDs = make(map[uint32]struct{})

	go startListenerMessageLoop()

	select {}
}

func UTF16toString(p *uint16) string {
	return syscall.UTF16ToString((*[4096]uint16)(unsafe.Pointer(p))[:])

	// ptr := unsafe.Pointer(p)                   // necessary to arbitrarily cast to *[4096]uint16 (?)
	// uint16ptrarr := (*[4096]uint16)(ptr)[:]    // 4096 is arbitrary? could be smaller
	// return syscall.UTF16ToString(uint16ptrarr) // now uint16ptrarr is in a format to pass to the builtin converter
}

var lastForeground []Scripts

// EVENT_SYSTEM_FOREGROUND
func OnForeground(exe string, pid uint32) {
	log.Println(Green, "EVENT_SYSTEM_FOREGROUND", Reset, exe)

	if len(lastForeground) != 0 {
		for _, script := range lastForeground {
			runScript(pid, script)
		}
		lastForeground = []Scripts{}
	}

	if len(gamesList[exe].OnProcessStart) != 0 {
		for _, script := range gamesList[exe].OnProcessStart {
			if script.OnForeground {
				runScript(pid, script)
			}
		}
	}

	lastForeground = []Scripts{}
	if len(gamesList[exe].OnProcessFinish) != 0 {
		for _, script := range gamesList[exe].OnProcessFinish {
			if script.OnBackground {
				lastForeground = append(lastForeground, script)
			}
		}
	}
}

// EVENT_OBJECT_DESTROY
func OnProcessFinish(pid uint32, exe string) {
	log.Println(Red, "EVENT_OBJECT_DESTROY", Reset, exe)

	if len(gamesList[exe].OnProcessFinish) != 0 {
		for _, script := range gamesList[exe].OnProcessFinish {
			runScript(pid, script)
		}
	}
}

// EVENT_OBJECT_CREATE
func OnProcessStart(pid uint32, exe string) {
	log.Println(Cyan, "EVENT_OBJECT_CREATE", Reset, exe)

	if len(gamesList[exe].OnProcessStart) != 0 {
		for _, script := range gamesList[exe].OnProcessStart {
			runScript(pid, script)
		}
	}
}

func runScript(pid uint32, script Scripts) {
	var cmd_instance *exec.Cmd
	if script.Args == "" {
		log.Printf("exec.Command(%s)\n", script.Name)
		cmd_instance = exec.Command(script.Name)
	} else {
		tempArgs := strings.ReplaceAll(script.Args, "%pid%", strconv.Itoa(int(pid)))
		log.Printf("exec.Command(%s, %s)\n", script.Name, tempArgs)
		cmd_instance = exec.Command(script.Name, strings.Split(tempArgs, " ")...)
	}
	if script.HideWindow {
		cmd_instance.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}

	cmd_instance.Start()
	// output, err := cmd_instance.CombinedOutput()
	// if err != nil {
	// 	log.Println(err)
	// }
	// println("output", string(output))
}
