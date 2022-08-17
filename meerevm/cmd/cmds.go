// Copyright (c) 2017-2018 The qitmeer developers

package cmd

import (
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/urfave/cli/v2"
)

var (
	Commands = []*cli.Command{
		removedbCommand,
		// See accountcmd.go:
		accountCommand,
		walletCommand,
		// See consolecmd.go:
		attachCommand,
		// see dbcmd.go
		dbCommand,
		// See cmd/utils/flags_legacy.go
		utils.ShowDeprecated,
		// See snapshot.go
		snapshotCommand,
	}
)
