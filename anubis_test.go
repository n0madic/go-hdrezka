package hdrezka

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"testing"
)

func TestSolveAnubisPoW(t *testing.T) {
	t.Parallel()

	// randomData is arbitrary but fixed so the search is deterministic.
	const randomData = "e51062c1069fa79efe851624c26c8a61745ed41cceb2ef3674f1abbc2ca31966"

	for _, difficulty := range []int{1, 2, 3, 4} {
		t.Run("difficulty_"+strconv.Itoa(difficulty), func(t *testing.T) {
			t.Parallel()
			hash, nonce, err := solveAnubisPoW(randomData, difficulty)
			if err != nil {
				t.Fatalf("solveAnubisPoW() error: %v", err)
			}

			// The returned hash must equal SHA-256(randomData + nonce)...
			sum := sha256.Sum256([]byte(randomData + strconv.Itoa(nonce)))
			if got := hex.EncodeToString(sum[:]); got != hash {
				t.Fatalf("hash mismatch: solver=%s recomputed=%s", hash, got)
			}
			// ...and satisfy the leading zero-nibble difficulty.
			for i := range difficulty {
				nibble := sum[i/2] >> 4
				if i%2 != 0 {
					nibble = sum[i/2] & 0x0f
				}
				if nibble != 0 {
					t.Fatalf("nibble %d of %s is not zero for difficulty %d", i, hash, difficulty)
				}
			}
		})
	}
}

func TestSolveAnubisPoWInvalidDifficulty(t *testing.T) {
	t.Parallel()
	if _, _, err := solveAnubisPoW("abcd", 0); err == nil {
		t.Fatal("expected error for zero difficulty, got nil")
	}
}
