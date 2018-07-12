// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package pq

import "github.com/elastic/go-txfile"

// cursor holds state for iterating events in the queue.
type cursor struct {
	page     txfile.PageID
	off      int
	pageSize int
}

// txCursor is used to advance a cursor within a transaction.
type txCursor struct {
	*cursor
	accessor *access
	tx       *txfile.Tx
	page     *txfile.Page
}

// Nil checks if the cursor is pointing to a page. Returns true, if cursor is
// not pointing to any page in the queue.
func (c *cursor) Nil() bool {
	return c.page == 0
}

func makeTxCursor(tx *txfile.Tx, accessor *access, cursor *cursor) txCursor {
	return txCursor{
		tx:       tx,
		accessor: accessor,
		page:     nil,
		cursor:   cursor,
	}
}

func (c *txCursor) init(op string) reason {
	if c.page != nil {
		return nil
	}
	page, err := c.tx.Page(c.cursor.page)
	if err != nil {
		return c.errWrap(op, err)
	}

	c.page = page
	return nil
}

// Read reads more bytes from the current event into b.  If the end of the
// current event has reached, no bytes will be read.
func (c *txCursor) Read(b []byte) (int, reason) {
	const op = "pq/read-bytes"

	if err := c.init(op); err != nil {
		return 0, err
	}

	if c.Nil() {
		return 0, nil
	}

	to, err := c.readInto(op, b)
	return len(b) - len(to), err
}

// Skip skips the next n bytes.
func (c *txCursor) Skip(n int) reason {
	const op = "pq/skip"

	for n > 0 {
		if c.PageBytes() == 0 {
			ok, err := c.AdvancePage()
			if err != nil {
				return c.errWrap(op, err).of(SeekFail)
			}
			if !ok {
				return c.err(op).report("No page to seek to")
			}
		}

		max := n
		if L := c.PageBytes(); L < max {
			max = L
		}
		c.cursor.off += max
		n -= max
	}

	return nil
}

func (c *txCursor) readInto(op string, to []byte) ([]byte, reason) {
	for len(to) > 0 {
		// try to advance cursor to next page if last read did end at end of page
		if c.PageBytes() == 0 {
			ok, err := c.AdvancePage()
			if !ok {
				return to, nil
			}
			if err != nil {
				return to, c.errWrap(op, err)
			}
		}

		var n int
		err := c.WithBytes(op, func(b []byte) { n = copy(to, b) })
		to = to[n:]
		c.cursor.off += n
		if err != nil {
			return to, err
		}
	}

	return to, nil
}

func (c *txCursor) ReadEventHeader(op string) (hdr *eventHeader, err reason) {
	err = c.WithBytes(op, func(b []byte) {
		hdr = castEventHeader(b)
		c.off += szEventHeader
	})
	return
}

func (c *txCursor) PageHeader(op string) (hdr *eventPage, err reason) {
	err = c.WithHdr(op, func(h *eventPage) {
		hdr = h
	})
	return
}

func (c *txCursor) AdvancePage() (ok bool, err reason) {
	const op = "pq/cursor-next-page"

	err = c.WithHdr(op, func(hdr *eventPage) {
		nextID := txfile.PageID(hdr.next.Get())
		tracef("advance page from %v -> %v\n", c.cursor.page, nextID)
		ok = nextID != 0

		if ok {
			c.cursor.page = nextID
			c.cursor.off = szEventPageHeader
			c.page = nil
		}
	})
	return
}

func (c *txCursor) WithPage(op string, fn func([]byte)) reason {
	if err := c.init(op); err != nil {
		return err
	}

	buf, err := c.page.Bytes()
	if err != nil {
		return c.errWrap(op, err).of(ReadFail)
	}

	fn(buf)
	return nil
}

func (c *txCursor) WithHdr(op string, fn func(*eventPage)) reason {
	return c.WithPage(op, func(b []byte) {
		fn(castEventPageHeader(b))
	})
}

func (c *txCursor) WithBytes(op string, fn func([]byte)) reason {
	return c.WithPage(op, func(b []byte) {
		fn(b[c.off:])
	})
}

// PageBytes reports the amount of bytes still available in current page
func (c *cursor) PageBytes() int {
	return c.pageSize - c.off
}

func (c *cursor) Reset() {
	*c = cursor{}
}

func (c *txCursor) err(op string) *Error {
	return &Error{op: op, ctx: c.errCtx(c.cursor.page)}
}

func (c *txCursor) errWrap(op string, cause error) *Error {
	return c.err(op).causedBy(cause)
}

func (c *txCursor) errCtx(page txfile.PageID) errorCtx {
	return c.accessor.errPageCtx(page)
}
