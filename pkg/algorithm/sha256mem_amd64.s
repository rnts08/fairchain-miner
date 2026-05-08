//go:build amd64 && sha_ni
// +build amd64,sha_ni

// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.
//
// This file contains placeholder assembly for SHA-NI accelerated SHA-256 compression.
// A full implementation would involve detailed knowledge of SHA-NI instruction set
// and careful optimization.

#include "textflag.h"

// Mask for PSHUFB to convert Big-Endian message blocks to Little-Endian XMM words
DATA flip_mask<>+0x00(SB)/8, $0x0405060700010203
DATA flip_mask<>+0x08(SB)/8, $0x0c0d0e0f08090a0b
GLOBL flip_mask<>(SB), RODATA, $16

// SHA-256 Constants (K256)
DATA k256<>+0x00(SB)/4, $0x428a2f98; DATA k256<>+0x04(SB)/4, $0x71374491; DATA k256<>+0x08(SB)/4, $0xb5c0fbcf; DATA k256<>+0x0c(SB)/4, $0xe9b5dba5
DATA k256<>+0x10(SB)/4, $0x3956c25b; DATA k256<>+0x14(SB)/4, $0x59f111f1; DATA k256<>+0x18(SB)/4, $0x923f82a4; DATA k256<>+0x1c(SB)/4, $0xab1c5ed5
DATA k256<>+0x20(SB)/4, $0xd807aa98; DATA k256<>+0x24(SB)/4, $0x12835b01; DATA k256<>+0x28(SB)/4, $0x243185be; DATA k256<>+0x2c(SB)/4, $0x550c7dc3
DATA k256<>+0x30(SB)/4, $0x72be5d74; DATA k256<>+0x34(SB)/4, $0x80deb1fe; DATA k256<>+0x38(SB)/4, $0x9bdc06a7; DATA k256<>+0x3c(SB)/4, $0xc19bf174
DATA k256<>+0x40(SB)/4, $0xe49b69c1; DATA k256<>+0x44(SB)/4, $0xefbe4786; DATA k256<>+0x48(SB)/4, $0x0fc19dc6; DATA k256<>+0x4c(SB)/4, $0x240ca1cc
DATA k256<>+0x50(SB)/4, $0x2de92c6f; DATA k256<>+0x54(SB)/4, $0x4a7484aa; DATA k256<>+0x58(SB)/4, $0x5cb0a9dc; DATA k256<>+0x5c(SB)/4, $0x76f988da
DATA k256<>+0x60(SB)/4, $0x983e5152; DATA k256<>+0x64(SB)/4, $0xa831c66d; DATA k256<>+0x68(SB)/4, $0xb00327c8; DATA k256<>+0x6c(SB)/4, $0xbf597fc7
DATA k256<>+0x70(SB)/4, $0xc6e00bf3; DATA k256<>+0x74(SB)/4, $0xd5a79147; DATA k256<>+0x78(SB)/4, $0x06ca6351; DATA k256<>+0x7c(SB)/4, $0x14292967
DATA k256<>+0x80(SB)/4, $0x27b70a85; DATA k256<>+0x84(SB)/4, $0x2e1b2138; DATA k256<>+0x88(SB)/4, $0x4d2c6dfc; DATA k256<>+0x8c(SB)/4, $0x53380d13
DATA k256<>+0x90(SB)/4, $0x650a7354; DATA k256<>+0x94(SB)/4, $0x766a0abb; DATA k256<>+0x98(SB)/4, $0x81c2c92e; DATA k256<>+0x9c(SB)/4, $0x92722c85
DATA k256<>+0xa0(SB)/4, $0xa2bfe8a1; DATA k256<>+0xa4(SB)/4, $0xa81a664b; DATA k256<>+0xa8(SB)/4, $0xc24b8b70; DATA k256<>+0xac(SB)/4, $0xc76c51a3
DATA k256<>+0xb0(SB)/4, $0xd192e819; DATA k256<>+0xb4(SB)/4, $0xd6990624; DATA k256<>+0xb8(SB)/4, $0xf40e3585; DATA k256<>+0xbc(SB)/4, $0x106aa070
DATA k256<>+0xc0(SB)/4, $0x19a4c116; DATA k256<>+0xc4(SB)/4, $0x1e376c08; DATA k256<>+0xc8(SB)/4, $0x2748774c; DATA k256<>+0xcc(SB)/4, $0x34b0bcb5
DATA k256<>+0xd0(SB)/4, $0x391c0cb3; DATA k256<>+0xd4(SB)/4, $0x4ed8aa4a; DATA k256<>+0xd8(SB)/4, $0x5b9cca4f; DATA k256<>+0xdc(SB)/4, $0x682e6ff3
DATA k256<>+0xe0(SB)/4, $0x748f82ee; DATA k256<>+0xe4(SB)/4, $0x78a5636f; DATA k256<>+0xe8(SB)/4, $0x84c87814; DATA k256<>+0xec(SB)/4, $0x8cc70208
DATA k256<>+0xf0(SB)/4, $0x90befffa; DATA k256<>+0xf4(SB)/4, $0xa4506ceb; DATA k256<>+0xf8(SB)/4, $0xbef9a3f7; DATA k256<>+0xfc(SB)/4, $0xc67178f2
GLOBL k256<>(SB), (RODATA|NOPTR), $256

// func prefetcht0(addr uintptr)
TEXT ·prefetcht0(SB), NOSPLIT, $0-8
    MOVQ addr+0(FP), AX
    PREFETCHT0 (AX)
    RET

// func sha256CompressSHA_NI(state *[8]uint32, block *[64]byte)
TEXT ·sha256CompressSHA_NI(SB), NOSPLIT, $0-16
    MOVQ state+0(FP), DI
    MOVQ block+8(FP), SI

    // Load initial state
    MOVOU 0(DI), X0      // X0 = [D, C, B, A]
    MOVOU 16(DI), X1     // X1 = [H, G, F, E]

    // Save initial state for add-back
    MOVOA X0, X10
    MOVOA X1, X11

    // SHA256RNDS2 expects state split into ABEF and CDGH
    PSHUFD $0xB1, X0, X0 // [D,C,B,A] -> [C,D,A,B]
    PSHUFD $0xB1, X1, X1 // [H,G,F,E] -> [G,H,E,F]
    MOVOA X0, X2
    PUNPCKLQDQ X1, X0    // X0 = ABEF
    PUNPCKHQDQ X1, X2    // X2 = CDGH
    MOVOA X0, X1

    // Load and shuffle message blocks (W0-W15)
    MOVOU 0(SI), X4;  PSHUFB flip_mask<>(SB), X4
    MOVOU 16(SI), X5; PSHUFB flip_mask<>(SB), X5
    MOVOU 32(SI), X6; PSHUFB flip_mask<>(SB), X6
    MOVOU 48(SI), X7; PSHUFB flip_mask<>(SB), X7

    // Rounds 0-63
    // Using a macro-like structure for the 4-round blocks (16 blocks total)
    // Each block of 4 rounds: 
    // 1. Add constants to message
    // 2. Perform Rounds (using SHA256RNDS2 twice)
    // 3. Update message schedule (using SHA256MSG1/2)

#define ROUNDS_00_15(msg, next, msg2, msg3, k_off) \
    MOVOA msg, X0; PADDD k256<>+k_off(SB), X0; \
    SHA256RNDS2 X0, X2, X1; \
    PSHUFD $0x0E, X0, X0; \
    SHA256RNDS2 X0, X1, X2; \
    SHA256MSG1 msg3, msg; \
    MOVOA msg2, X3; \
    PALIGNR $4, msg, X3; \
    PADDD X3, msg; \
    SHA256MSG2 next, msg

    ROUNDS_00_15(X4, X7, X5, X6, 0x00)
    ROUNDS_00_15(X5, X4, X6, X7, 0x10)
    ROUNDS_00_15(X6, X5, X7, X4, 0x20)
    ROUNDS_00_15(X7, X6, X4, X5, 0x30)

    // Final rounds (48-63) do not need message schedule updates for the next block
    ROUNDS_00_15(X4, X7, X5, X6, 0x40)
    ROUNDS_00_15(X5, X4, X6, X7, 0x50)
    ROUNDS_00_15(X6, X5, X7, X4, 0x60)
    ROUNDS_00_15(X7, X6, X4, X5, 0x70)
    ROUNDS_00_15(X4, X7, X5, X6, 0x80)
    ROUNDS_00_15(X5, X4, X6, X7, 0x90)
    ROUNDS_00_15(X6, X5, X7, X4, 0xa0)
    ROUNDS_00_15(X7, X6, X4, X5, 0xb0)
    ROUNDS_00_15(X4, X7, X5, X6, 0xc0)
    ROUNDS_00_15(X5, X4, X6, X7, 0xd0)
    ROUNDS_00_15(X6, X5, X7, X4, 0xe0)
    
    // Final 4 rounds (60-63)
    MOVOA X7, X0; PADDD k256<>+0xf0(SB), X0; SHA256RNDS2 X0, X2, X1; PSHUFD $0x0E, X0, X0; SHA256RNDS2 X0, X1, X2

    // Add back to initial state and store
    MOVOA X1, X0
    PUNPCKLQDQ X2, X1
    PUNPCKHQDQ X2, X0
    PADDD X10, X1
    PADDD X11, X0
    MOVOU X1, 0(DI)
    MOVOU X0, 16(DI)

    RET

    