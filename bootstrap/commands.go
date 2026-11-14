package bootstrap

import (
	"github.com/AlexGalhardo/vukem"

	"github.com/AlexGalhardo/vukem-trading/app/commands"
)

func Commands() []vukem.Command {
	return []vukem.Command{commands.Worker{}, &commands.OwnerReset{}, commands.RecoDaily{}}
}
