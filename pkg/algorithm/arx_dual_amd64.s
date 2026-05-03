//go:build amd64
// +build amd64

#include "textflag.h"

DATA  offsets<>+0x00(SB)/4, $0
DATA  offsets<>+0x04(SB)/4, $1
DATA  offsets<>+0x08(SB)/4, $2
DATA  offsets<>+0x0c(SB)/4, $3
DATA  offsets<>+0x10(SB)/4, $4
DATA  offsets<>+0x14(SB)/4, $5
DATA  offsets<>+0x18(SB)/4, $6
DATA  offsets<>+0x1c(SB)/4, $7
GLOBL offsets<>+0x00(SB), RODATA, $32

// func arxFillDualAVX2(dst1, dst2, src1, src2 *[32]byte, idx1, idx2 uint32)
TEXT ·arxFillDualAVX2(SB), NOSPLIT, $0
    MOVQ dst1+0(FP), AX
    MOVQ dst2+8(FP), BX
    MOVQ src1+16(FP), CX
    MOVQ src2+24(FP), DX
    MOVL idx1+32(FP), SI
    MOVL idx2+36(FP), DI

    // Load sources
    VMOVDQU (CX), Y0
    VMOVDQU (DX), Y1

    // Prepare offsets [0, 1, ..., 7]
    VMOVDQU offsets<>(SB), Y2

    // Prepare indices
    VMOVQ SI, X3; VPBROADCASTD X3, Y3; VPADDD Y2, Y3, Y3 // idx1 vec
    VMOVQ DI, X4; VPBROADCASTD X4, Y4; VPADDD Y2, Y4, Y4 // idx2 vec

    // Block 1: v = src ^ index_vec
    VPXOR Y3, Y0, Y3
    // Block 2: v = src ^ index_vec
    VPXOR Y4, Y1, Y4

    // ROTL13(v) = (v << 13) | (v >> 19)
    VPSLLD $13, Y3, Y5; VPSRLD $19, Y3, Y6; VPOR Y5, Y6, Y3
    VPSLLD $13, Y4, Y7; VPSRLD $19, Y4, Y14; VPOR Y7, Y14, Y4 // Use X14/Y14 in dual version

    // v += src
    VPADDD Y0, Y3, Y3
    VPADDD Y1, Y4, Y4

    // Store
    VMOVDQU Y3, (AX)
    VMOVDQU Y4, (BX)

    VZEROUPPER
    RET

// func arxFillDualAVX512(dst1, dst2, src1, src2 *[32]byte, idx1, idx2 uint32)
TEXT ·arxFillDualAVX512(SB), NOSPLIT, $0
    // Requires AVX-512F
    MOVQ dst1+0(FP), AX
    MOVQ dst2+8(FP), BX
    MOVQ src1+16(FP), CX
    MOVQ src2+24(FP), DX
    MOVL idx1+32(FP), SI
    MOVL idx2+36(FP), DI

    // Load sources into one ZMM (src1 in low 256 bits, src2 in high 256 bits)
    VMOVDQU64 (CX), Y0
    VMOVDQU64 (DX), Y1
    VINSERTI64X4 $1, Y1, Z0, Z0

    // Prepare offsets [0, 1, ..., 7, 0, 1, ..., 7]
    VMOVDQU64 offsets<>(SB), Y1
    VINSERTI64X4 $1, Y1, Z1, Z1

    // Prepare indices [idx1, ..., idx1, idx2, ..., idx2]
    // Wait! This is harder to do in one step.
    // We'll use VBROADCASTI64X4 and then fixup?
    // Or just load them into Z.
    VMOVQ SI, X2; VPBROADCASTD X2, Y2
    VMOVQ DI, X3; VPBROADCASTD X3, Y3
    VINSERTI64X4 $1, Y3, Z2, Z2 // Z2 now has [idx1*8, idx2*8]
    VPADDD Z1, Z2, Z2 // Z2 has [idx1+0...7, idx2+0...7]

    // v = src ^ index_vec
    VPTERNLOGD $0x66, Z2, Z0, Z2 // Z2 = Z2 ^ Z0 (0x66 is XOR)

    // v = ROTL13(v)
    // AVX-512 has VPROLD!
    VPROLD $13, Z2, Z2

    // v += src
    VPADDD Z0, Z2, Z2

    // Store
    VEXTRACTI64X4 $0, Z2, Y0
    VEXTRACTI64X4 $1, Z2, Y1
    VMOVDQU64 Y0, (AX)
    VMOVDQU64 Y1, (BX)

    VZEROUPPER
    RET
