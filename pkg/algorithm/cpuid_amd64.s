//go:build amd64
// +build amd64

#include "textflag.h"

// func cpuid_sha() bool
TEXT ·cpuid_sha(SB), NOSPLIT, $0-1
	MOVL $7, AX
	MOVL $0, CX
	CPUID
	// EBX bit 29 is SHA
	SHRL $29, BX
	ANDL $1, BX
	MOVB BX, ret+0(FP)
	RET
