package main // https://github.com/jet/damon/blob/8b2f833924dcfa53fc7196ad85f99d977d947e45/win32/process_win32.go

import (
	"log"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32DLL = windows.NewLazySystemDLL("kernel32.dll")
	ntdllDLL    = windows.NewLazySystemDLL("ntdll.dll")

	procSetProcessAffinityMask  = kernel32DLL.NewProc("SetProcessAffinityMask")
	procSetProcessPriorityBoost = kernel32DLL.NewProc("SetProcessPriorityBoost")
	procSetPriorityClass        = kernel32DLL.NewProc("SetPriorityClass")

	procNtSetInformationProcess   = ntdllDLL.NewProc("NtSetInformationProcess")
	procNtQueryInformationProcess = ntdllDLL.NewProc("NtQueryInformationProcess")
)

type ULONG uint32

// const ( // IO_PRIORITY_HINT
// 	IoPriorityVeryLow  = iota // Defragging, content indexing and other background I/Os.
// 	IoPriorityLow             // Prefetching for applications.
// 	IoPriorityNormal          // Normal I/Os.
// 	IoPriorityHigh            // Used by filesystems for checkpoint I/O.
// 	IoPriorityCritical        // Used by memory manager. Not available for applications.
// )

const (
	ProcessVMRead                  = 0x0010
	ProcessQueryLimitedInformation = 0x0100 // https://docs.microsoft.com/en-us/windows/win32/procthread/process-security-and-access-rights
	ProcessQueryInformation        = 0x0400 // https://docs.microsoft.com/en-us/windows/win32/procthread/process-security-and-access-rights
	ProcessSetIinformation         = 0x0200

	ProcessIoPriority   = 0x21 // https://www.pinvoke.net/default.aspx/ntdll/PROCESSINFOCLASS.html
	ProcessPagePriority = 0x27
)

// https://docs.microsoft.com/en-us/windows/desktop/api/winbase/nf-winbase-setprocessaffinitymask
func SetProcessAffinityMask(hProcess windows.Handle, dwProcessAffinityMask uint64) error {
	r1, _, e1 := procSetProcessAffinityMask.Call(
		uintptr(hProcess),
		uintptr(dwProcessAffinityMask), // uintptr(unsafe.Pointer(&sam)),
	)

	if int(r1) == 0 {
		return os.NewSyscallError("GetProcessAffinityMask", e1)
	}
	return nil // testReturnCodeNonZero(ret, errno)
}

func SetProcessPriorityBoost(process windows.Handle, disable bool) (err error) {
	var _p0 uint32
	if disable {
		_p0 = 1
	}
	r1, _, e1 := procSetProcessPriorityBoost.Call(
		uintptr(process),
		uintptr(_p0))

	if int(r1) == 0 {
		err = os.NewSyscallError("SetProcessPriorityBoost", e1)
	}
	return
}

// The Processinfoclass constants have been derived from the PROCESSINFOCLASS enum definition.
type Processinfoclass uint32

// https://github.com/hillu/go-ntdll/blob/a6f426aa8d92e55860a843a12b13d16824a082ad/process_generated.go
func NtSetInformationProcess(
	processHandle windows.Handle,
	processInformationClass Processinfoclass,
	processInformation *uint32,
	processInformationLength uint32,
) error {
	r1, _, e1 := procNtSetInformationProcess.Call(
		uintptr(processHandle),
		uintptr(processInformationClass),
		uintptr(unsafe.Pointer(processInformation)),
		uintptr(processInformationLength))
	if int(r1) == 0 {
		return os.NewSyscallError("SetInformationProcess", e1)
	}
	return nil
}

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

func SetPriorityClass(process syscall.Handle, priorityClass uint32) (err error) {
	r1, r2, e1 := syscall.Syscall(procSetPriorityClass.Addr(), 2, uintptr(process), uintptr(priorityClass), 0)
	log.Println(r1, r2, e1)
	if r1 == 0 {
		if e1 != 0 {
			err = e1
		} else {
			err = syscall.EINVAL
		}
	}
	return
}
