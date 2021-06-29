package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBitMap(t *testing.T) {
	bm := NewSlotBitMap(256)
	bm.SetSlot(7, true)
	bm.SetSlot(10, true)
	bm.SetSlot(234, true)
	assert.Equal(t, true, bm.GetSlot(7))
	assert.Equal(t, true, bm.GetSlot(10))
	assert.Equal(t, true, bm.GetSlot(234))

	assert.Equal(t, 3, bm.GetVaildSlotNum())

	bm.SetSlot(7, false)
	assert.Equal(t, false, bm.GetSlot(7))
	assert.Equal(t, 2, bm.GetVaildSlotNum())

}

func TestBitMapExportSlots(t *testing.T) {
	bm := NewSlotBitMap(256)
	bm.SetSlot(1, true)
	bm.SetSlot(10, true)
	bm.SetSlot(187, true)
	bm.SetSlot(200, true)
	bm.SetSlot(234, true)

	exportSlots := bm.ExportSlots(3)

	assert.Equal(t, true, bm.GetSlot(1))
	assert.Equal(t, true, bm.GetSlot(10))
	assert.Equal(t, false, bm.GetSlot(187))
	assert.Equal(t, false, bm.GetSlot(200))
	assert.Equal(t, false, bm.GetSlot(234))

	ebm := NewSlotBitMapWithBits(exportSlots)

	assert.Equal(t, false, ebm.GetSlot(1))
	assert.Equal(t, false, ebm.GetSlot(10))
	assert.Equal(t, true, ebm.GetSlot(187))
	assert.Equal(t, true, ebm.GetSlot(200))
	assert.Equal(t, true, ebm.GetSlot(234))

}

func TestBitMapCleanSlots(t *testing.T) {
	bm := NewSlotBitMap(256)
	bm.SetSlot(1, true)
	bm.SetSlot(100, true)
	bm.SetSlot(255, true)

	bm2 := NewSlotBitMap(256)
	bm2.SetSlot(1, true)
	bm2.SetSlot(200, true)
	bm2.SetSlot(255, true)

	bm.CleanSlots(bm2.GetBits())

	assert.Equal(t, false, bm.GetSlot(1))
	assert.Equal(t, true, bm.GetSlot(100))
	assert.Equal(t, false, bm.GetSlot(200))
	assert.Equal(t, false, bm.GetSlot(255))
}

func TestSlotsContains(t *testing.T) {
	bm := NewSlotBitMap(256)
	bm.SetSlot(1, true)
	bm.SetSlot(100, true)
	bm.SetSlot(255, true)

	bm2 := NewSlotBitMap(255)
	bm2.SetSlot(1, true)
	bm2.SetSlot(200, true)
	bm2.SetSlot(255, true)

	assert.Equal(t, false, SlotsContains(bm.GetBits(), bm2.GetBits()))

	bm3 := NewSlotBitMap(255)
	bm3.SetSlot(1, true)
	bm3.SetSlot(255, true)

	assert.Equal(t, true, SlotsContains(bm.GetBits(), bm3.GetBits()))

}
