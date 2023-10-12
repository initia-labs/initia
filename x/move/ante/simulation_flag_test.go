package ante_test

import (
	"context"
	"testing"

	"github.com/initia-labs/initia/x/move/ante"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestSimulationFlagDecorator(t *testing.T) {
	specs := map[string]struct {
		simulation bool
		expErr     interface{}
	}{
		"simulation mode": {
			simulation: true,
		},
		"non simulation mode": {
			simulation: false,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			ctx := sdk.Context{}.WithContext(context.Background())
			decorator := ante.NewSimulationFlagDecorator()
			ctx, err := decorator.AnteHandle(ctx, nil, spec.simulation, nil)
			require.NoError(t, err)
			require.Equal(t, spec.simulation, ctx.Value(ante.SimulationFlagContextKey).(bool))
		})
	}
}
