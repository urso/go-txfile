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

package txfile

import (
	"fmt"

	"github.com/elastic/go-txfile/internal/strbld"
	"github.com/elastic/go-txfile/internal/vfs"
	"github.com/elastic/go-txfile/txerr"
)

type reason interface {
	txerr.Error
}

type Error struct {
	op    string
	kind  error
	cause error

	ctx errorCtx
	msg string
}

type errorCtx struct {
	file string

	isOff  bool
	offset int64

	isTx bool
	txid uint

	isPage bool
	page   PageID
}

var _ reason = &Error{}

func (e *Error) Error() string   { return txerr.Report(e) }
func (e *Error) Op() string      { return e.op }
func (e *Error) Kind() error     { return e.kind }
func (e *Error) Cause() error    { return e.cause }
func (e *Error) Context() string { return e.ctx.String() }
func (e *Error) Message() string { return e.msg }

type ErrKind int

// internal txfile error kinds

//go:generate stringer -type=ErrKind -linecomment=true
const (
	NoError            ErrKind = iota // no error
	InternalError                     // internal error
	FileCreationFailed                // can not create file
	InitFailed                        // failed to initialize from file
	InvalidConfig                     // configuration error
	InvalidFileSize                   // invalid file size
	InvalidMetaPage                   // meta page invalid
	InvalidOp                         // invalid operation
	InvalidPageID                     // page id out of bounds
	InvalidParam                      // invalid parameter
	OutOfMemory                       // out of memory
	TxCommitFail                      // transaction failed during commit
	TxRollbackFail                    // transaction failed during rollback
	TxFailed                          // transaction failed
	TxFinished                        // finished transaction
	TxReadOnly                        // readonly transaction
	endOfErrKind                      // unknown error kind
)

// re-export file system error kinds (from internal/vfs)

const (
	PermissionError       = vfs.ErrPermission
	FileExists            = vfs.ErrExist
	FileDoesNotExist      = vfs.ErrNotExist
	FileClosed            = vfs.ErrClosed
	NoDiskSpace           = vfs.ErrNoSpace
	FDLimit               = vfs.ErrFDLimit
	CantResolvePath       = vfs.ErrResolvePath
	IOError               = vfs.ErrIO
	OSOtherError          = vfs.ErrOSOther
	OperationNotSupported = vfs.ErrNotSupported
	LockFailed            = vfs.ErrLockFailed
)

func (k ErrKind) Error() string {
	if k > endOfErrKind {
		k = endOfErrKind
	}
	return k.String()
}

func (e *Error) of(kind ErrKind) *Error { e.kind = kind; return e }

func (e *Error) report(m string) *Error                     { e.msg = m; return e }
func (e *Error) reportf(m string, vs ...interface{}) *Error { return e.report(fmt.Sprintf(m, vs...)) }

func (e *Error) causedBy(cause error) *Error {
	e.cause = cause
	other, ok := cause.(*Error)
	if !ok {
		return e
	}

	errCtx := &e.ctx
	causeCtx := &other.ctx
	if errCtx.file == causeCtx.file {
		causeCtx.file = ""
	}
	if errCtx.isTx && causeCtx.isTx && errCtx.txid == causeCtx.txid {
		causeCtx.isTx = false // delete common tx id from cause context
	}
	if errCtx.isPage && causeCtx.isPage && errCtx.page == causeCtx.page {
		causeCtx.isPage = false // delete common page id from cause context
	}
	if errCtx.isOff && causeCtx.isOff && errCtx.offset == causeCtx.offset {
		causeCtx.isOff = false // delete common page id from cause context
	}

	return e
}

func (ctx *errorCtx) String() string {
	buf := &strbld.Builder{}
	if ctx.file != "" {
		buf.Fmt("file='%s'", ctx.file)
	}
	if ctx.isTx {
		buf.Pad(" ")
		buf.Fmt("tx=%v", ctx.txid)
	}
	if ctx.isPage {
		buf.Pad(" ")
		buf.Fmt("page=%v", ctx.page)
	}
	if ctx.isOff {
		buf.Pad(" ")
		buf.Fmt("offset=%v", ctx.offset)
	}
	return buf.String()
}

func (ctx *errorCtx) SetPage(id PageID) {
	ctx.isPage, ctx.page = true, id
}

func (ctx *errorCtx) SetOffset(off int64) {
	ctx.isOff, ctx.offset = true, off
}

func errOp(op string) *Error {
	return &Error{op: op}
}

func errOf(kind ErrKind) *Error {
	return &Error{kind: kind}
}

func wrapErr(err error) *Error {
	return &Error{cause: err}
}

func raiseInvalidParam(msg string) reason {
	return &Error{kind: InvalidParam, msg: msg}
}

func raiseInvalidParamf(msg string, vs ...interface{}) reason {
	return raiseInvalidParam(fmt.Sprintf(msg, vs...))
}

func raiseOutOfBounds(id PageID) reason {
	return &Error{
		kind: InvalidPageID,
		ctx: errorCtx{
			isPage: true,
			page:   id,
		},
		msg: "out put bounds page id",
	}
}
