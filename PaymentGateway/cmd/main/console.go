// +build windows

package main

import (
	"flag"
	"os"
	"syscall"
)

func init() {
	c := flag.Bool("console", false, "pass this flag in order to work in console")
	flag.Parse()
	if *c {
		offsetCliArgs()
	}
}

// _AttachParentProcess is a prent pid to attach to.
const _AttachParentProcess = ^uint32(0) // (DWORD)-1

var (
	kernel32DLL       = syscall.NewLazyDLL("kernel32.dll")
	attachConsoleProc = kernel32DLL.NewProc("AttachConsole")
	allocConsoleProc  = kernel32DLL.NewProc("AllocConsole")
	freeConsoleProc   = kernel32DLL.NewProc("FreeConsole")
)

// AttachConsole is a win api wrapper for AttachConsole function.
func AttachConsole(pid uint32) bool {
	ret, _, _ := syscall.Syscall(attachConsoleProc.Addr(), 1, uintptr(pid), 0, 0)
	return ret != 0
}

// AllocConsole is a win api wrapper for AllocConsole function.
func AllocConsole() bool {
	ret, _, _ := syscall.Syscall(allocConsoleProc.Addr(), 0, 0, 0, 0)
	return ret != 0
}

// FreeConsole is a win api wrapper for FreeConsole function.
func FreeConsole() bool {
	ret, _, _ := syscall.Syscall(freeConsoleProc.Addr(), 0, 0, 0, 0)
	return ret != 0
}

func offsetCliArgs() {
	if attached := AttachConsole(_AttachParentProcess); !attached {
		if allocated := AllocConsole(); !allocated {
			panic("cannot allocate a console")
		}
		defer FreeConsole()
	}
	stdOut, stdOutErr := syscall.GetStdHandle(syscall.STD_OUTPUT_HANDLE)
	stdErr, stdErrErr := syscall.GetStdHandle(syscall.STD_ERROR_HANDLE)
	if stdOutErr != nil || stdErrErr != nil {
		panic("cannot redirect stsandard output and error streams")
	}
	_, err := os.Stdout.Stat()
	if err != nil {
		os.Stdout = os.NewFile(uintptr(stdOut), "/dev/stdout")
	}
	_, err = os.Stderr.Stat()
	if os.Stderr == nil {
		os.Stderr = os.NewFile(uintptr(stdErr), "/dev/stderr")
	}
	oldArgs := os.Args
	os.Args = append([]string{}, oldArgs[0])
	os.Args = append(os.Args, oldArgs[2:]...)
}
