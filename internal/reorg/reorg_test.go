package reorg

import (
	"testing"
)

func TestIsSafe_BelowConfirmations(t *testing.T) {
	d := New(3)
	d.UpdateTip(100)

	// Block 97 is 3 confirmations behind tip (100-3=97)
	if !d.IsSafe(97) {
		t.Error("block 97 should be safe (3 confirmations behind tip 100)")
	}
}

func TestIsSafe_AboveConfirmations(t *testing.T) {
	d := New(3)
	d.UpdateTip(100)

	// Block 98 is only 2 confirmations behind tip
	if d.IsSafe(98) {
		t.Error("block 98 should NOT be safe (only 2 confirmations behind tip 100)")
	}
}

func TestSafeHead(t *testing.T) {
	d := New(6)
	d.UpdateTip(100)

	expected := uint64(94) // 100 - 6
	if got := d.SafeHead(); got != expected {
		t.Errorf("SafeHead = %d, want %d", got, expected)
	}
}

func TestSafeHead_TipTooLow(t *testing.T) {
	d := New(10)
	d.UpdateTip(5)

	// Tip < confirmations, safe head should be 0
	if got := d.SafeHead(); got != 0 {
		t.Errorf("SafeHead = %d, want 0 (tip < confirmations)", got)
	}
}
