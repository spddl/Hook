package main // https://github.com/jet/damon/blob/8b2f833924dcfa53fc7196ad85f99d977d947e45/win32/process_win32.go

import (
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	ntdllDLL                      = windows.NewLazySystemDLL("ntdll.dll")
	procNtQueryInformationProcess = ntdllDLL.NewProc("NtQueryInformationProcess")
)

const ProcessQueryInformation = 0x0400 // https://docs.microsoft.com/en-us/windows/win32/procthread/process-security-and-access-rights

// NtQueryInformationProcess is a wrapper for ntdll.NtQueryInformationProcess.
// The handle must have the PROCESS_QUERY_INFORMATION access right.
// Returns an error of type NTStatus.
func NtQueryInformationProcess(processHandle windows.Handle, processInformationClass int32, processInformation windows.Pointer, processInformationLength uint32, returnLength *uint32) error {
	r1, _, err := procNtQueryInformationProcess.Call(
		uintptr(processHandle),
		uintptr(processInformationClass),
		uintptr(unsafe.Pointer(processInformation)),
		uintptr(processInformationLength),
		uintptr(unsafe.Pointer(returnLength)))
	if int(r1) < 0 {
		return os.NewSyscallError("NtQueryInformationProcess", err)
	}
	return nil
}
