package cli

import (
	"github.com/ZanzyTHEbar/firedragon-go/internal"
	"github.com/spf13/cobra"
)

// CmdParams holds all dependencies needed by command handlers
type CmdParams struct {
	Config  *internal.Config
	Logger  *internal.Logger
	Palette []*cobra.Command
	Use     string
	Alias   string
	Short   string
	Long    string
}

type CLICMD struct {
	Root *cobra.Command
}

func NewCMD(cmdRoot *cobra.Command) *CLICMD {
	return &CLICMD{
		Root: cmdRoot,
	}
}
