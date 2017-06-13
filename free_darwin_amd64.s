#include "textflag.h"

// ·madviseAddress is a helper function to get the actual address of the
// runtime·madvise function. It must be in assembly, since runtime·madvise is
// not exposed publicly via Go.
TEXT ·madviseAddress(SB),NOSPLIT,$0
	LEAQ runtime·madvise(SB), AX
	MOVQ AX, ret+0(FP)
	RET

// callbackAddress is a helper function to get the actual address of the
// callback function, since you cannot take the address of a function in Go.
TEXT ·callbackAddress(SB),NOSPLIT,$0
	LEAQ ·callback(SB), AX
	MOVQ AX, ret+0(FP)
	RET

// callback is invoked in place of any call to runtime.madvise
// (see src/runtime/sys_darwin_amd64.s:100). That function is patched at
// runtime to invoke this one.
TEXT ·callback(SB),NOSPLIT,$0
	SUBQ $32, SP
	MOVQ BP, 24(SP)
	LEAQ 24(SP), BP
	MOVQ addr+32(FP), AX
	MOVQ AX, (SP)
	MOVQ n+40(FP), AX
	MOVQ AX, 8(SP)
	MOVL flags+48(FP), AX
	MOVL AX, 16(SP)
	CALL ·runtimeMadviseCallback(SB)
	MOVQ 24(SP), BP
	ADDQ $32, SP
	RET

// rawMadvise is a raw syscall version of madvise; it must be used instead of
// syscall.Syscall in Go due to locking. It is copied from src/runtime/sys_darwin_amd64.s:100
TEXT ·rawMadvise(SB), NOSPLIT, $0
	MOVQ	addr+0(FP), DI      // arg 1 addr
	MOVQ	n+8(FP), SI         // arg 2 len
	MOVL	flags+16(FP), DX    // arg 3 advice
	MOVL	$(0x2000000+75), AX	// syscall entry madvise
	SYSCALL
	// ignore failure - maybe pages are locked
	RET

// rawMprotect is a raw syscall version of mprotect; it must be used instead of
// syscall.Syscall in Go due to locking.
TEXT ·rawMprotect(SB), NOSPLIT, $0
	MOVQ	addr+0(FP), DI      // arg 1 addr
	MOVQ	n+8(FP), SI         // arg 2 len
	MOVL	prot+16(FP), DX     // arg 3 prot
	MOVL	$(0x2000000+74), AX // syscall entry madvise
	SYSCALL
	// ignore failure
	RET
