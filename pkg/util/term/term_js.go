//go:build js

/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package term

import (
	"errors"
	"io"

	wordwrap "github.com/mitchellh/go-wordwrap"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/kubectl/pkg/util/interrupt"
)

// SafeFunc is a function to be invoked by TTY.
type SafeFunc func() error

// TTY helps invoke a function and preserve terminal behavior.
type TTY struct {
	In     io.Reader
	Out    io.Writer
	Raw    bool
	TryDev bool
	Parent *interrupt.Handler
}

// IsTerminalIn returns true if t.In is a terminal.
func (t TTY) IsTerminalIn() bool {
	return printers.IsTerminal(t.In)
}

// IsTerminalOut returns true if t.Out is a terminal.
func (t TTY) IsTerminalOut() bool {
	return printers.IsTerminal(t.Out)
}

// IsTerminal returns whether the passed object is a terminal.
var IsTerminal = printers.IsTerminal

// AllowsColorOutput returns true if color output is supported and desired.
var AllowsColorOutput = printers.AllowsColorOutput

// Safe invokes fn without terminal mutation on js targets.
func (t TTY) Safe(fn SafeFunc) error {
	return interrupt.Chain(t.Parent).Run(fn)
}

// NewDetachableReader returns the original reader on js targets.
func NewDetachableReader(r io.Reader, detachKeys string) (io.Reader, error) {
	return r, nil
}

// TerminalSize represents the width and height of a terminal.
type TerminalSize struct {
	Width  uint16
	Height uint16
}

// TerminalSizeQueue is capable of returning terminal resize events.
type TerminalSizeQueue interface {
	Next() *TerminalSize
}

// GetSize returns nil on js targets.
func (t TTY) GetSize() *TerminalSize {
	return nil
}

// GetSize returns nil on js targets.
func GetSize(fd uintptr) *TerminalSize {
	return nil
}

// MonitorSize returns nil on js targets.
func (t *TTY) MonitorSize(initialSizes ...*TerminalSize) TerminalSizeQueue {
	return nil
}

type wordWrapWriter struct {
	limit  uint
	writer io.Writer
}

// NewResponsiveWriter returns w on js targets.
func NewResponsiveWriter(w io.Writer) io.Writer {
	return w
}

// NewWordWrapWriter wraps content to the supplied line limit.
func NewWordWrapWriter(w io.Writer, limit uint) io.Writer {
	return &wordWrapWriter{
		limit:  limit,
		writer: w,
	}
}

// GetWordWrapperLimit returns an unsupported error on js targets.
func GetWordWrapperLimit() (uint, error) {
	return 0, errors.New("file descriptor is not a terminal")
}

func (w wordWrapWriter) Write(p []byte) (nn int, err error) {
	if w.limit == 0 {
		return w.writer.Write(p)
	}
	return w.writer.Write([]byte(wordwrap.WrapString(string(p), w.limit)))
}

// NewPunchCardWriter limits output to 80 columns.
func NewPunchCardWriter(w io.Writer) io.Writer {
	return NewWordWrapWriter(w, 80)
}

type maxWidthWriter struct {
	maxWidth     uint
	currentWidth uint
	written      uint
	writer       io.Writer
}

// NewMaxWidthWriter creates a writer that enforces a maximum width.
func NewMaxWidthWriter(w io.Writer, maxWidth uint) io.Writer {
	return &maxWidthWriter{
		maxWidth: maxWidth,
		writer:   w,
	}
}

func (m *maxWidthWriter) Write(p []byte) (nn int, err error) {
	for _, b := range p {
		if m.currentWidth == m.maxWidth {
			_, err := m.writer.Write([]byte{'\n'})
			if err != nil {
				return int(m.written), err
			}
			m.currentWidth = 0
		}
		if b == '\n' {
			m.currentWidth = 0
		}
		_, err := m.writer.Write([]byte{b})
		if err != nil {
			return int(m.written), err
		}
		m.written++
		m.currentWidth++
	}
	return len(p), nil
}
