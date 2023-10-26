package pluginsintegration

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/plugins"
	"github.com/grafana/grafana/pkg/plugins/backendplugin"
	"github.com/grafana/grafana/pkg/plugins/backendplugin/coreplugin"
	"github.com/grafana/grafana/pkg/plugins/backendplugin/provider"
	pluginsCfg "github.com/grafana/grafana/pkg/plugins/config"
	"github.com/grafana/grafana/pkg/plugins/envvars"
	"github.com/grafana/grafana/pkg/plugins/manager/client"
	"github.com/grafana/grafana/pkg/plugins/manager/fakes"
	"github.com/grafana/grafana/pkg/plugins/manager/loader"
	"github.com/grafana/grafana/pkg/plugins/manager/loader/angular/angularinspector"
	"github.com/grafana/grafana/pkg/plugins/manager/loader/assetpath"
	"github.com/grafana/grafana/pkg/plugins/manager/loader/finder"
	"github.com/grafana/grafana/pkg/plugins/manager/pipeline/bootstrap"
	"github.com/grafana/grafana/pkg/plugins/manager/pipeline/discovery"
	"github.com/grafana/grafana/pkg/plugins/manager/pipeline/initialization"
	"github.com/grafana/grafana/pkg/plugins/manager/pipeline/termination"
	"github.com/grafana/grafana/pkg/plugins/manager/pipeline/validation"
	"github.com/grafana/grafana/pkg/plugins/manager/process"
	"github.com/grafana/grafana/pkg/plugins/manager/registry"
	"github.com/grafana/grafana/pkg/plugins/manager/signature"
	"github.com/grafana/grafana/pkg/plugins/manager/signature/statickey"
	"github.com/grafana/grafana/pkg/plugins/manager/sources"
	"github.com/grafana/grafana/pkg/plugins/pluginscdn"
	"github.com/grafana/grafana/pkg/services/featuremgmt"
	"github.com/grafana/grafana/pkg/services/pluginsintegration/config"
	"github.com/grafana/grafana/pkg/services/pluginsintegration/pipeline"
	"github.com/grafana/grafana/pkg/services/pluginsintegration/pluginerrs"
	"github.com/grafana/grafana/pkg/services/pluginsintegration/pluginstore"
	"github.com/grafana/grafana/pkg/setting"
)

type IntegrationTestCtx struct {
	PluginClient   plugins.Client
	PluginStore    *pluginstore.Service
	PluginRegistry registry.Service
}

func CreateIntegrationTestCtx(t *testing.T, cfg *setting.Cfg, coreRegistry *coreplugin.Registry) *IntegrationTestCtx {
	pCfg, err := config.ProvideConfig(setting.ProvideProvider(cfg), cfg, featuremgmt.WithFeatures())
	require.NoError(t, err)

	cdn := pluginscdn.ProvideService(pCfg)
	reg := registry.ProvideService()
	angularInspector := angularinspector.NewStaticInspector()
	proc := process.ProvideService()
	errTracker := pluginerrs.ProvideSignatureErrorTracker()

	env := envvars.NewProvider(pCfg, fakes.NewFakeLicensingService())
	clientReg := client.ProvideBackendClientRegistry(provider.ProvideService(coreRegistry), env, proc)

	disc := pipeline.ProvideDiscoveryStage(pCfg, finder.NewLocalFinder(true), reg)
	boot := pipeline.ProvideBootstrapStage(pCfg, signature.ProvideService(pCfg, statickey.New()), assetpath.ProvideService(pCfg, cdn))
	valid := pipeline.ProvideValidationStage(pCfg, signature.NewValidator(signature.NewUnsignedAuthorizer(pCfg)), angularInspector, errTracker)
	init := pipeline.ProvideInitializationStage(pCfg, reg, fakes.NewFakeLicensingService(), provider.ProvideService(coreRegistry), proc, &fakes.FakeAuthService{}, fakes.NewFakeRoleRegistry(), clientReg)
	term, err := pipeline.ProvideTerminationStage(pCfg, reg, proc, clientReg)
	require.NoError(t, err)

	l := CreateTestLoader(t, pCfg, LoaderOpts{
		Discoverer:   disc,
		Bootstrapper: boot,
		Validator:    valid,
		Initializer:  init,
		Terminator:   term,
	})

	ps, err := pluginstore.ProvideService(reg, sources.ProvideService(cfg, pCfg), l)
	require.NoError(t, err)

	return &IntegrationTestCtx{
		PluginClient:   client.ProvideService(pCfg, clientReg),
		PluginStore:    ps,
		PluginRegistry: reg,
	}
}

type LoaderOpts struct {
	Discoverer   discovery.Discoverer
	Bootstrapper bootstrap.Bootstrapper
	Validator    validation.Validator
	Terminator   termination.Terminator
	Initializer  initialization.Initializer
}

func CreateTestLoader(t *testing.T, cfg *pluginsCfg.Cfg, opts LoaderOpts) *loader.Loader {
	if opts.Discoverer == nil {
		opts.Discoverer = pipeline.ProvideDiscoveryStage(cfg, finder.NewLocalFinder(cfg.DevMode), registry.ProvideService())
	}

	if opts.Bootstrapper == nil {
		opts.Bootstrapper = pipeline.ProvideBootstrapStage(cfg, signature.ProvideService(cfg, statickey.New()), assetpath.ProvideService(cfg, pluginscdn.ProvideService(cfg)))
	}

	if opts.Validator == nil {
		opts.Validator = pipeline.ProvideValidationStage(cfg, signature.NewValidator(signature.NewUnsignedAuthorizer(cfg)), angularinspector.NewStaticInspector(), pluginerrs.ProvideSignatureErrorTracker())
	}

	reg := registry.ProvideService()
	coreRegistry := coreplugin.NewRegistry(make(map[string]backendplugin.PluginFactoryFunc))
	clientRegistry := client.ProvideBackendClientRegistry(provider.ProvideService(coreRegistry), envvars.NewProvider(cfg, fakes.NewFakeLicensingService()), process.ProvideService())

	if opts.Initializer == nil {
		opts.Initializer = pipeline.ProvideInitializationStage(cfg, reg, fakes.NewFakeLicensingService(), provider.ProvideService(coreRegistry), process.ProvideService(), &fakes.FakeAuthService{}, fakes.NewFakeRoleRegistry(), clientRegistry)
	}

	if opts.Terminator == nil {
		var err error
		opts.Terminator, err = pipeline.ProvideTerminationStage(cfg, reg, process.ProvideService(), clientRegistry)
		require.NoError(t, err)
	}

	return loader.New(opts.Discoverer, opts.Bootstrapper, opts.Validator, opts.Initializer, opts.Terminator)
}
