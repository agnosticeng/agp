package signer

import (
	"encoding/hex"
	"fmt"

	"github.com/agnosticeng/agp/internal/signer"
	"github.com/urfave/cli/v2"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:  "signer",
		Usage: "signer <secret> <data>",
		Action: func(ctx *cli.Context) error {
			var (
				signer = signer.HMAC256Signer([]byte(ctx.Args().Get(0)))
				sig    = signer([]byte(ctx.Args().Get(1)))
			)

			if ctx.Args().Len() != 2 {
				return fmt.Errorf("secret and data must be provided")
			}

			fmt.Println("secret", ctx.Args().Get(0))
			fmt.Println("data", ctx.Args().Get(1))
			fmt.Println("signature (hex)", hex.EncodeToString(sig))
			return nil
		},
	}
}
