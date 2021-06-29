package util

// SlotBitMap SlotBitMap
type SlotBitMap struct {
	bits    []byte
	slotNum int
}

// NewSlotBitMap NewSlotBitMap
func NewSlotBitMap(slotNum int) *SlotBitMap {
	var bits []byte
	if slotNum%8 == 0 {
		bits = make([]byte, (slotNum / 8))
	} else {
		bits = make([]byte, (slotNum/8)+1)
	}
	return &SlotBitMap{bits: bits, slotNum: slotNum}
}

// NewSlotBitMapWithBits NewSlotBitMapWithBits
func NewSlotBitMapWithBits(bits []byte) *SlotBitMap {

	return &SlotBitMap{bits: bits}
}

// SetSlot SetSlot
func (s *SlotBitMap) SetSlot(num uint, v bool) {
	index := num / 8
	pos := num % 8
	if v {
		s.bits[index] |= 1 << pos
	} else {
		s.bits[index] = s.bits[index] & ^(1 << pos)
	}
}

// SetSlotForRange SetSlotForRange
func (s *SlotBitMap) SetSlotForRange(start, end uint, v bool) {
	for i := start; i <= end; i++ {
		s.SetSlot(i, v)
	}
}

// GetSlot GetSlot
func (s *SlotBitMap) GetSlot(num uint) bool {
	index := num / 8
	pos := num % 8
	return s.bits[index]&(1<<pos) != 0
}

// Reset Reset
func (s *SlotBitMap) Reset() {
	var bits []byte
	if s.slotNum%8 == 0 {
		bits = make([]byte, (s.slotNum / 8))
	} else {
		bits = make([]byte, (s.slotNum/8)+1)
	}
	s.bits = bits
}

// GetBits GetBits
func (s *SlotBitMap) GetBits() []byte {
	return s.bits
}

// GetVaildSlotNum GetVaildSlotNum
func (s *SlotBitMap) GetVaildSlotNum() int {
	var count = 0
	for i := 0; i < len(s.bits); i++ {
		b := s.bits[i]
		for j := 0; j < 8; j++ {
			vaild := (b >> j & 0x01) == 1
			if vaild {
				count++
			}
		}
	}
	return count
}

// ExportSlots ExportSlots
func (s *SlotBitMap) ExportSlots(num int) []byte {
	exportBits := make([]byte, len(s.bits))
	exportNum := num
	for i := len(s.bits) - 1; i >= 0; i-- {
		if exportNum <= 0 {
			break
		}
		b := s.bits[i]
		eb := exportBits[i]
		for j := 8 - 1; j >= 0; j-- {
			if exportNum <= 0 {
				break
			}
			vaild := (b >> j & 0x01) == 1
			if vaild {
				eb = eb | (0x01 << j)
				b = b & (^(0x01 << j))
				exportNum--
			}
		}
		s.bits[i] = b
		exportBits[i] = eb
	}
	return exportBits
}

// CleanSlots CleanSlots
func (s *SlotBitMap) CleanSlots(slots []byte) {
	if len(slots) == 0 {
		return
	}
	for i := len(s.bits) - 1; i >= 0; i-- {
		b := s.bits[i]
		if len(slots)-(len(s.bits)-i) >= 0 {
			v := slots[len(slots)-(len(s.bits)-i)]
			b = b & (^v)
		}
		s.bits[i] = b
	}
}

// MergeSlots MergeSlots
func (s *SlotBitMap) MergeSlots(bs ...[]byte) {
	if len(bs) == 0 {
		return
	}
	for i := 0; i < len(s.bits); i++ {
		b := s.bits[i]

		for j := 0; j < len(bs); j++ {
			if i < len(bs[j]) {
				v := bs[j][i]
				b = b | v
			}
		}
		s.bits[i] = b
	}
	return

}

// SlotsContains SlotsContains
func SlotsContains(b, subslice []byte) bool {
	if len(b) < len(subslice) {
		return false
	}
	for i := 0; i < len(b); i++ {
		b1 := b[i]
		s1 := subslice[i]
		for j := 0; j < 8; j++ {
			b11 := b1 >> j & 0x01
			s11 := s1 >> j & 0x01
			if s11 == 1 && b11 == 0 {
				return false
			}

		}

	}
	return true
}
