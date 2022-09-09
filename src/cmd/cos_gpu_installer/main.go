// Package main is the program entrance.
package main

import (
	"context"
	"flag"
	"os"

	log "github.com/golang/glog"
	"github.com/google/subcommands"

	"cos.googlesource.com/cos/tools.git/src/cmd/cos_gpu_installer/internal/commands"
	"cos.googlesource.com/cos/tools.git/src/pkg/utils"
)

func main() {
	// Always log to stderr for easy debugging.
	flag.Set("alsologtostderr", "true")
	flag.Parse()

	log.V(2).Info("Checking if this is the only cos_gpu_installer that is running.")
	f := utils.Flock()
	defer f.Close()

	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(subcommands.FlagsCommand(), "")
	subcommands.Register(subcommands.CommandsCommand(), "")
	subcommands.Register(&commands.InstallCommand{}, "")
	subcommands.Register(&commands.ListCommand{}, "")

	ctx := context.Background()
	os.Exit(int(subcommands.Execute(ctx)))
}
