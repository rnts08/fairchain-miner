//go:build amd64
// +build amd64

#include "textflag.h"

// func prefetcht0(addr uintptr)
TEXT ·prefetcht0(SB), NOSPLIT, $0-8
	MOVQ addr+0(FP), AX
	PREFETCHT0 (AX)
	RET
