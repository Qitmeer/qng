package eth

import (
	"github.com/ethereum/go-ethereum/log"
	"log/slog"
	"strings"
)

var packLevelMap = map[string]slog.Level{
	"github.com/ethereum/go-ethereum/miner": slog.LevelDebug, // The original level is info
}

func (h *agentTerminalHandler) enabledRecord(r *slog.Record) bool {
	if len(packLevelMap) <= 0 {
		return true
	}
	location := source(*r)
	if len(location) <= 0 {
		return true
	}
	temp0 := strings.Split(location, "/")
	if len(temp0) <= 0 {
		return true
	}
	temp1 := strings.Split(temp0[len(temp0)-1], ".")
	if len(temp1) <= 0 {
		return true
	}
	pack := strings.Join(temp0[:len(temp0)-1], "/") + "/" + temp1[0]

	level, ok := packLevelMap[pack]
	if !ok {
		return true
	}
	r.Level = level
	if r.Level >= log.FromLegacyLevel(int(h.lvl)) {
		return true
	}
	return false
}
