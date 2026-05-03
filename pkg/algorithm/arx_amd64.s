//go:build amd64
// +build amd64

#include "textflag.h"

// func arxFillAVX2(dst, src *[32]byte, index uint32)
TEXT ·arxFillAVX2(SB), NOSPLIT, $0
    MOVQ dst+0(FP), AX
    MOVQ src+8(FP), BX
    MOVL index+16(FP), CX

    // Load src into Y0
    VMOVDQU (BX), Y0

    // Prepare index vector: [idx+7, idx+6, idx+5, idx+4, idx+3, idx+2, idx+1, idx+0]
    // We can load [0, 1, 2, 3, 4, 5, 6, 7] and add CX to all elements.
    DATA  offsets<>+0x00(SB)/4, $0
    DATA  offsets<>+0x04(SB)/4, $1
    DATA  offsets<>+0x08(SB)/4, $2
    DATA  offsets<>+0x0c(SB)/4, $3
    DATA  offsets<>+0x10(SB)/4, $4
    DATA  offsets<>+0x14(SB)/4, $5
    DATA  offsets<>+0x18(SB)/4, $6
    DATA  offsets<>+0x1c(SB)/4, $7
    GLOBL offsets<>+0x00(SB), RODATA, $32

    VMOVDQU offsets<>(SB), Y1
    VMOVQ CX, X2
    VPBROADCASTD X2, Y2
    VPADDD Y2, Y1, Y1 // Y1 now has [idx, idx+1, ..., idx+7]

    // v = src ^ index_vec
    VPXOR Y1, Y0, Y1

    // v = ROTL13(v)
    // (v << 13) | (v >> 19)
    VPSLLD $13, Y1, Y2
    VPSRLD $19, Y1, Y3
    VPOR Y2, Y3, Y1

    // v += src
    VPADDD Y0, Y1, Y1

    // Store to dst
    VMOVDQU Y1, (AX)
    VZEROUPPER
    RET
