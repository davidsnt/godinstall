// Copyright 2015 Ulrich Kunitz. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lzma

// The type hashDict provides a dictionary with a hash table of all
// 4-byte strings in the dictionary.
type hashDict struct {
	buf  *buffer
	head int64
	size int64
	t4   *hashTable
}

// newhashDict creates a new hash dictionary.
func newHashDict(buf *buffer, head int64, size int64) (hd *hashDict, err error) {
	if !(buf.bottom <= head && head <= buf.top) {
		return nil, rangeError{"head", head}
	}
	if !(MinDictSize <= size && size <= int64(buf.capacity())) {
		return nil, rangeError{"size", size}
	}
	t4, err := newHashTable(size, 4)
	if err != nil {
		return nil, err
	}
	hd = &hashDict{buf: buf, head: head, size: size, t4: t4}
	return hd, nil
}

// offset returns the current offset of the head of the dictionary.
func (hd *hashDict) offset() int64 {
	return hd.head
}

// byteAt returns the byte at the distance. The method returns a zero
// byte if the distance exceeds the current length of the dictionary.
func (hd *hashDict) byteAt(dist int64) byte {
	if !(0 < dist && dist <= hd.size) {
		panic("dist out of range")
	}
	off := hd.head - dist
	if off < hd.buf.bottom {
		return 0
	}
	return hd.buf.data[hd.buf.index(off)]
}

// resets set the hash dictionary back to an empty status.
func (hd *hashDict) reset() {
	hd.buf.reset()
	hd.head = 0
	hd.t4.reset()
}

// move advances the head n bytes forward and record the new data in the
// hash table.
func (hd *hashDict) move(n int) (moved int, err error) {
	if n < 0 {
		return 0, negError{"n", n}
	}
	if !(hd.buf.bottom <= hd.head && hd.head <= hd.buf.top) {
		panic("head out of range")
	}
	off := add(hd.head, int64(n))
	if off > hd.buf.top {
		off = hd.buf.top
	}
	moved, err = hd.buf.writeRangeTo(hd.head, off, hd.t4)
	hd.head += int64(moved)
	return
}

// start returns the start of the dictionary.
func (hd *hashDict) start() int64 {
	start := hd.head - hd.size
	if start < hd.buf.bottom {
		start = hd.buf.bottom
	}
	return start
}

// sync synchronizes the write limit of the backing buffer with the
// current dictionary head.
func (hd *hashDict) sync() {
	hd.buf.writeLimit = add(hd.start(), int64(hd.buf.capacity()))
}
