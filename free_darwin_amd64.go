// Package darfree uses black magic to release memory on Darwin.
//
// darfree is a package that uses black magic (assembly and runtime monkey
// patching) to lower the perceived memory consumption of Go programs on
// Darwin / amd64 (Mac OS) systems. Since it is based on page size, the actual
// 'gains' are dependent upon the program itself. They can be as much as your
// largest heap allocation. See cmd/darfree for an example which uses ~500MB
// without darfree, and only ~17MB with.
//
// Background
//
// Go allocates memory in pages, and later advises the operating system that it
// may free unneeded memory using madvise(..., MADV_FREE) calls. On Linux, this
// system call immediately frees the memory page. However, on Darwin machines
// the kernel will decide to retain that memory as part of your program's RSS
// (residential set size) until memory pressure by your application or another
// running on the system causes the kernel to _truly_ free that memory.
//
// In reality, Darwin does the right thing. Keeping the memory as part of your
// program's RSS causes no harm, and actually benefits your program by making
// subsequent allocations faster. However, from a user's perspective looking at
// their "Activity Monitor" it will look like your application is leaking or
// using much more memory than it actually is. That is what darfree solves.
//
// Beware
//
// I advise you really do not use this package at all, unless you really
// understand what is going on and what the trade-offs are. At any point in
// time this package may become unmaintained, and may break with future Go
// versions.
//
package darfree

import (
	"encoding/binary"
	"fmt"
	"runtime"
	"syscall"
	"unsafe"
)

// stubs for functions defined in free_darwin_amd64.s
func madviseAddress() uintptr
func callbackAddress() uintptr
func callback()
func rawMadvise(addr unsafe.Pointer, n uintptr, flags int32)
func rawMprotect(addr unsafe.Pointer, n uintptr, prot int32)

// runtimeMadviseCallback is invoked from assembly (see free_darwin_amd64.s 'callback')
// whenever the runtime would normally make a runtime.madvise call. This gives
// us the chance to intercept MADV_FREE calls which unfortunately do not free
// the memory page immediately on OS X.
//
// The solution is to make two mprotect calls directly after an MADV_FREE call,
// with arguments PROT_NONE and PROT_READ|PROT_WRITE, respectively. These calls
// force the kernel to free the memory pages (for some reason, see here:
// https://stackoverflow.com/a/9003648).
func runtimeMadviseCallback(addr unsafe.Pointer, n uintptr, flags int32) {
	rawMadvise(addr, n, flags)
	if flags == syscall.MADV_FREE {
		rawMprotect(addr, n, syscall.PROT_NONE)
		rawMprotect(addr, n, syscall.PROT_READ|syscall.PROT_WRITE)
	}
}

func init() {
	// Get runtime·madvise address.
	madvisePtr := madviseAddress()

	// Validate the function pointer.
	if f := runtime.FuncForPC(madvisePtr); f == nil {
		panic("darfree: failed to find runtime·madvise address")
	}

	// Generate jump code from madvise to our own code.
	jmp := ccJmp(callbackAddress())

	// Mark the memory read+write+executable.
	err := mprotect(pageOf(madvisePtr), uintptr(syscall.Getpagesize()), syscall.PROT_READ|syscall.PROT_WRITE|syscall.PROT_EXEC)
	if err != nil {
		panic(fmt.Sprintf("darfree: mprotect: %v", err))
	}

	// Copy our jump code into madvise, overwriting the runtime version.
	for i, b := range jmp {
		*(*byte)(unsafe.Pointer(madvisePtr + uintptr(i))) = b
	}

	// Mark memory back as read+executable (not strictly needed).
	err = mprotect(pageOf(madvisePtr), uintptr(syscall.Getpagesize()), syscall.PROT_READ|syscall.PROT_EXEC)
	if err != nil {
		panic(fmt.Sprintf("darfree: mprotect: %v", err))
	}
}

// mprotect implements the mprotect syscall, since it is not already defined
// in the syscall package on darwin.
func mprotect(addr, len uintptr, prot int) error {
	_, _, errno := syscall.Syscall(syscall.SYS_MPROTECT, addr, len, uintptr(prot))
	if errno != 0 {
		return errno
	}
	return nil
}

// ccJmp will compile the following assembly with custom 0x41414141 and
// 0x42424242 literals.
//
// 	PUSH 0
// 	MOV DWORD PTR [RSP+0], 0x41414141
// 	MOV DWORD PTR [RSP+4], 0x42424242
// 	RET
//
// You can compile this assembly using something like e.g. https://defuse.ca/online-x86-assembler.htm#disassembly
// which produces:
//
// 	0:  6a 00                   push   0x0
// 	2:  c7 04 24 41 41 41 41    mov    DWORD PTR [rsp],0x41414141
// 	9:  c7 44 24 04 42 42 42    mov    DWORD PTR [rsp+0x4],0x42424242
// 	10: 42
// 	11: c3                      ret
//
func ccJmp(to uintptr) []byte {
	// Convert the address to bytes.
	addr := make([]byte, 8)
	binary.LittleEndian.PutUint64(addr, uint64(to))

	// PUSH 0; create room on the stack
	asm := []byte{0x6A, 0x00}

	// MOV DWORD PTR [RSP+0], 0x41414141; move into low 32 bits of return
	asm = append(asm, []byte{0xC7, 0x04, 0x24}...)
	asm = append(asm, addr[0:4]...)

	// MOV DWORD PTR [RSP+4], 0x42424242; move into high 32 bits of return
	asm = append(asm, []byte{0xC7, 0x44, 0x24, 0x04}...)
	asm = append(asm, addr[4:8]...)

	// RET
	asm = append(asm, 0xC3)
	return asm
}

// pageOf returns the page boundary of p.
func pageOf(p uintptr) uintptr {
	return p & ^(uintptr(syscall.Getpagesize()) - 1)
}
