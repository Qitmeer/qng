/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package eth

import (
	"bytes"
	"context"
	"fmt"
	qlog "github.com/Qitmeer/qng/log"
	"github.com/ethereum/go-ethereum/log"
	"io"
	"log/slog"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"unicode/utf8"
)

const (
	termTimeFormat    = "2006-01-02|15:04:05.000"
	termMsgJust       = 35
	termCtxMaxPadding = 35
)

var spaces = []byte("                                        ")
var glogger *log.GlogHandler
var agentTH *agentTerminalHandler

func InitLog(DebugLevel string, DebugPrintOrigins bool) {
	lvl, err := qlog.LvlFromString(DebugLevel)
	if err != nil {
		log.Warn(err.Error())
	}
	if glogger == nil {
		usecolor := qlog.LogWrite().IsUseColor()
		agentTH = NewTerminalHandler(qlog.LogWrite(), lvl, usecolor, DebugPrintOrigins)
		glogger = log.NewGlogHandler(agentTH)
		log.SetDefault(log.NewLogger(glogger))
	} else {
		agentTH.lvl = lvl
	}
	glogger.Verbosity(log.FromLegacyLevel(int(lvl)))
	qlog.LocationTrims = append(qlog.LocationTrims, "github.com/ethereum/go-ethereum/")
}

type agentTerminalHandler struct {
	mu       sync.Mutex
	wr       io.Writer
	lvl      qlog.Lvl
	useColor bool
	attrs    []slog.Attr
	// fieldPadding is a map with maximum field value lengths seen until now
	// to allow padding log contexts in a bit smarter way.
	fieldPadding map[string]int

	buf []byte

	locationEnabled bool
}

func NewTerminalHandler(wr io.Writer, lvl qlog.Lvl, useColor bool, locationEnabled bool) *agentTerminalHandler {
	return &agentTerminalHandler{
		wr:              wr,
		lvl:             lvl,
		useColor:        useColor,
		fieldPadding:    make(map[string]int),
		locationEnabled: locationEnabled,
	}
}

func (h *agentTerminalHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.enabledRecord(&r) {
		return nil
	}
	buf := h.format(h.buf, r, h.useColor)
	h.wr.Write(buf)
	h.buf = buf[:0]
	return nil
}

func (h *agentTerminalHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= log.FromLegacyLevel(int(h.lvl))
}

func (h *agentTerminalHandler) WithGroup(name string) slog.Handler {
	panic("not implemented")
}

func (h *agentTerminalHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &agentTerminalHandler{
		wr:           h.wr,
		lvl:          h.lvl,
		useColor:     h.useColor,
		attrs:        append(h.attrs, attrs...),
		fieldPadding: make(map[string]int),
	}
}

func (t *agentTerminalHandler) ResetFieldPadding() {
	t.mu.Lock()
	t.fieldPadding = make(map[string]int)
	t.mu.Unlock()
}

func (h *agentTerminalHandler) format(buf []byte, r slog.Record, usecolor bool) []byte {
	lvlstr := log.LevelString(r.Level)
	level, _ := qlog.LvlFromString(lvlstr)
	var color = 0
	if usecolor {
		switch level {
		case qlog.LvlCrit:
			color = 35
		case qlog.LvlError:
			color = 31
		case qlog.LvlWarn:
			color = 33
		case qlog.LvlInfo:
			color = 32
		case qlog.LvlDebug:
			color = 36
		case qlog.LvlTrace:
			color = 34
		}
	}

	b := &bytes.Buffer{}
	lvl := level.AlignedString()

	if h.locationEnabled {
		// Log origin printing was requested, format the location path and line number
		location := source(r)
		for _, prefix := range qlog.LocationTrims {
			location = strings.TrimPrefix(location, prefix)
		}
		// Maintain the maximum location length for fancyer alignment
		align := int(atomic.LoadUint32(&qlog.LocationLength))
		if align < len(location) {
			align = len(location)
			atomic.StoreUint32(&qlog.LocationLength, uint32(align))
		}
		padding := strings.Repeat(" ", align-len(location))

		// Assemble and print the log heading
		if color > 0 {
			fmt.Fprintf(b, "%s [\x1b[%dm%s|%s\x1b[0m]%s %s", r.Time.Format(termTimeFormat), color, lvl, location, padding, r.Message)
		} else {
			fmt.Fprintf(b, "%s [%s|%s]%s %s ", r.Time.Format(termTimeFormat), lvl, location, padding, r.Message)
		}
	} else {
		if color > 0 {
			fmt.Fprintf(b, "%s [\x1b[%dm%s\x1b[0m] %s ", r.Time.Format(termTimeFormat), color, lvl, r.Message)
		} else {
			fmt.Fprintf(b, "%s [%s] %s ", r.Time.Format(termTimeFormat), lvl, r.Message)
		}
	}
	// try to justify the log output for short messages
	length := utf8.RuneCountInString(r.Message)
	if len(h.attrs)+r.NumAttrs() > 0 && length < termMsgJust {
		b.Write(bytes.Repeat([]byte{' '}, termMsgJust-length))
	}
	h.formatAttributes(b, r, fmt.Sprintf("\x1b[%dm", color))
	return b.Bytes()
}

func (h *agentTerminalHandler) formatAttributes(buf *bytes.Buffer, r slog.Record, color string) {
	// tmp is a temporary buffer we use, until bytes.Buffer.AvailableBuffer() (1.21)
	// can be used.
	var tmp = make([]byte, 40)
	writeAttr := func(attr slog.Attr, first, last bool) {
		buf.WriteByte(' ')

		if color != "" {
			buf.WriteString(color)
			//buf.Write(appendEscapeString(buf.AvailableBuffer(), attr.Key))
			buf.Write(appendEscapeString(tmp[:0], attr.Key))
			buf.WriteString("\x1b[0m=")
		} else {
			//buf.Write(appendEscapeString(buf.AvailableBuffer(), attr.Key))
			buf.Write(appendEscapeString(tmp[:0], attr.Key))
			buf.WriteByte('=')
		}
		//val := FormatSlogValue(attr.Value, true, buf.AvailableBuffer())
		val := log.FormatSlogValue(attr.Value, tmp[:0])

		padding := h.fieldPadding[attr.Key]

		length := utf8.RuneCount(val)
		if padding < length && length <= termCtxMaxPadding {
			padding = length
			h.fieldPadding[attr.Key] = padding
		}
		buf.Write(val)
		if !last && padding > length {
			buf.Write(spaces[:padding-length])
		}
	}
	var n = 0
	var nAttrs = len(h.attrs) + r.NumAttrs()
	for _, attr := range h.attrs {
		writeAttr(attr, n == 0, n == nAttrs-1)
		n++
	}
	r.Attrs(func(attr slog.Attr) bool {
		writeAttr(attr, n == 0, n == nAttrs-1)
		n++
		return true
	})
	buf.WriteByte('\n')
}

func appendEscapeString(dst []byte, s string) []byte {
	needsQuoting := false
	needsEscaping := false
	for _, r := range s {
		// If it contains spaces or equal-sign, we need to quote it.
		if r == ' ' || r == '=' {
			needsQuoting = true
			continue
		}
		// We need to escape it, if it contains
		// - character " (0x22) and lower (except space)
		// - characters above ~ (0x7E), plus equal-sign
		if r <= '"' || r > '~' {
			needsEscaping = true
			break
		}
	}
	if needsEscaping {
		return strconv.AppendQuote(dst, s)
	}
	// No escaping needed, but we might have to place within quote-marks, in case
	// it contained a space
	if needsQuoting {
		dst = append(dst, '"')
		dst = append(dst, []byte(s)...)
		return append(dst, '"')
	}
	return append(dst, []byte(s)...)
}

func source(r slog.Record) string {
	fs := runtime.CallersFrames([]uintptr{r.PC})
	f, _ := fs.Next()
	return fmt.Sprintf("%s:%d", f.Function, f.Line)
}
