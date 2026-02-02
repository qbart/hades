package actions

import (
	"context"

	"github.com/SoftKiwiGames/hades/hades/types"
)

type Action interface {
	Execute(ctx context.Context, runtime *types.Runtime) error
	DryRun(ctx context.Context, runtime *types.Runtime) string
}
