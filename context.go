package pipers

import "context"

type PipersContext struct {
	context.Context
	Limit int
}
