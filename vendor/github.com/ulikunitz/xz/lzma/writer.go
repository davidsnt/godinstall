// Copyright 2015 Ulrich Kunitz. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lzma

import "io"

// NewWriter creates a new writer. It writes the LZMA header. It will use the
// Default Parameters.
//
// Don't forget to call Close() for the writer after all data has been written.
//
// For high performance use a buffered writer. But be aware that Close will not
// flush it.
func NewWriter(lzma io.Writer) (w *Writer, err error) {
	return NewWriterParams(lzma, Default)
}

// NewWriterParams creates a new writer using the provided parameters.
// The function writer the LZMA header.
//
// Don't forget to call Close() for the writer after all data has been written.
//
// For high performance use a buffered writer. But be aware that Close will not
// flush it.
func NewWriterParams(lzma io.Writer, p Parameters) (w *Writer, err error) {
	p.normalizeWriterSizes()
	if err = p.Verify(); err != nil {
		return nil, err
	}
	if !p.SizeInHeader {
		p.EOS = true
	}
	if err = writeHeader(lzma, &p); err != nil {
		return nil, err
	}
	w, err = NewStreamWriter(lzma, p)
	return
}
