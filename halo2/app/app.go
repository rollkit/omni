package app

import (
	engevmkeeper "github.com/omni-network/omni/halo2/evmengine/keeper"
	"github.com/omni-network/omni/lib/engine"
	"github.com/omni-network/omni/lib/errors"

	"cosmossdk.io/depinject"
	"cosmossdk.io/log"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	consensuskeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	// Blank imports for side-effects.
	_ "cosmossdk.io/api/cosmos/tx/config/v1"
	_ "github.com/cosmos/cosmos-sdk/x/auth"
	_ "github.com/cosmos/cosmos-sdk/x/auth/tx/config"
	_ "github.com/cosmos/cosmos-sdk/x/bank"
	_ "github.com/cosmos/cosmos-sdk/x/consensus"
	_ "github.com/cosmos/cosmos-sdk/x/distribution"
	_ "github.com/cosmos/cosmos-sdk/x/genutil"
	_ "github.com/cosmos/cosmos-sdk/x/mint"
	_ "github.com/cosmos/cosmos-sdk/x/staking"
	_ "github.com/omni-network/omni/halo2/evmengine/module"
)

const Name = "halo"

// App extends an ABCI application, but with most of its parameters exported.
// They are exported for convenience in creating helper functions, as object
// capabilities aren't needed for testing.
type App struct {
	*runtime.App

	appCodec          codec.Codec
	txConfig          client.TxConfig
	interfaceRegistry codectypes.InterfaceRegistry

	// keepers
	AccountKeeper         authkeeper.AccountKeeper
	BankKeeper            bankkeeper.Keeper
	StakingKeeper         *stakingkeeper.Keeper
	DistrKeeper           distrkeeper.Keeper
	ConsensusParamsKeeper consensuskeeper.Keeper
	EngEVMKeeper          engevmkeeper.Keeper
}

// newApp returns a reference to an initialized App.
func newApp(
	logger log.Logger,
	db dbm.DB,
	ethCl engine.API,
	appOpts servertypes.AppOptions,
	baseAppOptions ...func(*baseapp.BaseApp),
) (*App, error) {
	depCfg := depinject.Configs(
		DepConfig(),
		depinject.Supply(
			appOpts, logger, ethCl,
		),
	)

	var (
		app        = new(App)
		appBuilder = new(runtime.AppBuilder)
	)
	if err := depinject.Inject(depCfg,
		&appBuilder,
		&app.appCodec,
		&app.txConfig,
		&app.interfaceRegistry,
		&app.AccountKeeper,
		&app.BankKeeper,
		&app.StakingKeeper,
		&app.DistrKeeper,
		&app.ConsensusParamsKeeper,
		&app.EngEVMKeeper,
	); err != nil {
		return nil, errors.Wrap(err, "dep inject")
	}

	// Use evm engine to create and process block proposals.
	evmEngineOpts := func(bapp *baseapp.BaseApp) {
		bapp.SetPrepareProposal(app.EngEVMKeeper.PrepareProposal)
		bapp.SetProcessProposal(app.EngEVMKeeper.ProcessProposal)
	}
	baseAppOptions = append(baseAppOptions, evmEngineOpts)

	app.App = appBuilder.Build(db, nil, baseAppOptions...)

	if err := app.Load(true); err != nil {
		return nil, errors.Wrap(err, "load app")
	}

	return app, nil
}

func (App) LegacyAmino() *codec.LegacyAmino {
	return nil
}

func (App) ExportAppStateAndValidators(_ bool, _, _ []string) (servertypes.ExportedApp, error) {
	return servertypes.ExportedApp{}, errors.New("not implemented")
}

// SimulationManager implements the SimulationApp interface.
func (App) SimulationManager() *module.SimulationManager {
	return nil
}

var (
	_ runtime.AppI            = (*App)(nil)
	_ servertypes.Application = (*App)(nil)
)