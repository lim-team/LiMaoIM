package lmproto

import (
	"bytes"
	"fmt"
	"sync"
)

var (
	decOne = sync.Pool{
		New: func() interface{} {
			return make([]byte, 1)
		},
	}
	decTwo = sync.Pool{
		New: func() interface{} {
			return make([]byte, 2)
		},
	}
	decFour = sync.Pool{
		New: func() interface{} {
			return make([]byte, 4)
		},
	}
	decEight = sync.Pool{
		New: func() interface{} {
			return make([]byte, 8)
		},
	}
)

// Decoder 解码
type Decoder struct {
	r *bytes.Reader
}

// NewDecoder NewDecoder
func NewDecoder(p []byte) *Decoder {
	return &Decoder{
		r: bytes.NewReader(p),
	}
}

// Len 长度
func (d *Decoder) Len() int {
	return d.r.Len()
}

// Int Int
// func (d *Decoder) Int() (int, error) {
// 	b := decOne.Get().([]byte)
// 	defer func() {
// 		decOne.Put(b)
// 	}()
// 	if n, err := d.r.Read(b); err != nil {
// 		return 0, err
// 	} else if n != 1 {
// 		return 0, fmt.Errorf("Decoder couldn't read expect bytes %d of 1", n)
// 	}
// 	return int(b[0]), nil
// }

// Uint8 Uint8
func (d *Decoder) Uint8() (uint8, error) {
	b := decOne.Get().([]byte)
	defer func() {
		decOne.Put(b)
	}()
	if n, err := d.r.Read(b); err != nil {
		return 0, err
	} else if n != 1 {
		return 0, fmt.Errorf("Decoder couldn't read expect bytes %d of 1", n)
	}
	return b[0], nil
}

// Int16 Int16
func (d *Decoder) Int16() (int16, error) {
	b := decTwo.Get().([]byte)
	defer func() {
		decTwo.Put(b)
	}()
	if n, err := d.r.Read(b); err != nil {
		return 0, err
	} else if n != 2 {
		return 0, fmt.Errorf("Decoder couldn't read expect bytes %d of 2", n)
	}
	return (int16(b[0]) << 8) | int16(b[1]), nil
}

// Uint16 Uint16
func (d *Decoder) Uint16() (uint16, error) {
	if i, err := d.Int16(); err != nil {
		return 0, err
	} else {
		return uint16(i), nil
	}
}

// Bytes Bytes
func (d *Decoder) Bytes(num int) ([]byte, error) {
	b := make([]byte, num)
	_, err := d.r.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// Int64 Int64
func (d *Decoder) Int64() (int64, error) {
	b := decEight.Get().([]byte)
	defer func() {
		decEight.Put(b)
	}()
	if n, err := d.r.Read(b); err != nil {
		return 0, err
	} else if n != 8 {
		return 0, fmt.Errorf("Decoder couldn't read expect bytes %d of 8", n)
	}
	return (int64(b[0]) << 56) | (int64(b[1]) << 48) | (int64(b[2]) << 40) | int64(b[3])<<32 | int64(b[4])<<24 | int64(b[5])<<16 | int64(b[6])<<8 | int64(b[7]), nil
}

// Uint64 Uint64
func (d *Decoder) Uint64() (uint64, error) {
	b := decEight.Get().([]byte)
	defer func() {
		decEight.Put(b)
	}()
	if n, err := d.r.Read(b); err != nil {
		return 0, err
	} else if n != 8 {
		return 0, fmt.Errorf("Decoder couldn't read expect bytes %d of 8", n)
	}
	return (uint64(b[0]) << 56) | (uint64(b[1]) << 48) | (uint64(b[2]) << 40) | uint64(b[3])<<32 | uint64(b[4])<<24 | uint64(b[5])<<16 | uint64(b[6])<<8 | uint64(b[7]), nil
}

// Int32 Int32
func (d *Decoder) Int32() (int32, error) {
	b := decFour.Get().([]byte)
	defer func() {
		decFour.Put(b)
	}()
	if n, err := d.r.Read(b); err != nil {
		return 0, err
	} else if n != 4 {
		return 0, fmt.Errorf("Decoder couldn't read expect bytes %d of 4", n)
	}
	return (int32(b[0]) << 24) | (int32(b[1]) << 16) | (int32(b[2]) << 8) | int32(b[3]), nil
}

// Uint32 Uint32
func (d *Decoder) Uint32() (uint32, error) {
	if i, err := d.Int32(); err != nil {
		return 0, err
	} else {
		return uint32(i), nil
	}
}

func (d *Decoder) String() (string, error) {
	if buf, err := d.Binary(); err != nil {
		return "", err
	} else {
		return string(buf), nil
	}
}

// StringAll StringAll
func (d *Decoder) StringAll() (string, error) {
	if buf, err := d.BinaryAll(); err != nil {
		return "", err
	} else {
		return string(buf), nil
	}
}

// Binary Binary
func (d *Decoder) Binary() ([]byte, error) {
	size, err := d.Int16()
	if err != nil {
		return nil, err
	}
	buf := make([]byte, size)
	if size == 0 {
		return buf, nil
	}
	if n, err := d.r.Read(buf); err != nil {
		return nil, err
	} else if n != int(size) {
		return nil, fmt.Errorf("Decoder couldn't read expect bytes %d of %d", n, size)
	}
	return buf, nil
}

// BinaryAll BinaryAll
func (d *Decoder) BinaryAll() ([]byte, error) {
	remains := d.r.Len()
	if remains == 0 {
		return []byte{}, nil
	}
	buf := make([]byte, remains)
	if n, err := d.r.Read(buf); err != nil {
		return nil, err
	} else if n != remains {
		return nil, fmt.Errorf("Decoder couldn't read expect bytes %d of %d", n, remains)
	}
	return buf, nil
}

// Variable Variable
func (d *Decoder) Variable() (uint64, error) {
	var (
		size uint64
		mul  uint64 = 1
	)
	for {
		i, err := d.Uint8()
		if err != nil {
			return 0, err
		}
		size += uint64(i&0x7F) * mul
		mul *= 0x80
		if i&0x80 == 0 {
			break
		}
	}
	return size, nil
}
