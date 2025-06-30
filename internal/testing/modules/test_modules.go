package modules

import (
	"payloop/internal/lib/logger"
	"payloop/internal/lib/pubsub"
	"payloop/internal/testing/mocks"

	"go.uber.org/fx"
)

// MockLoggerModule provides a mock logger for testing
var MockLoggerModule = fx.Module("mock_logger",
	fx.Provide(mocks.NewSilentLogger),
	fx.Decorate(func(mockLogger *mocks.MockLogger) logger.Logger {
		return mockLogger
	}),
)

// MockPubSubModule provides a mock pubsub for testing
var MockPubSubModule = fx.Module("mock_pubsub",
	fx.Provide(mocks.NewSilentPubSub),
	fx.Decorate(func(mockPubSub *mocks.MockPubSub) pubsub.PubSub {
		return mockPubSub
	}),
)

// VerboseLoggerModule provides a mock logger that prints all calls (useful for debugging)
var VerboseLoggerModule = fx.Module("verbose_logger",
	fx.Provide(mocks.NewMockLogger),
	fx.Decorate(func(mockLogger *mocks.MockLogger) logger.Logger {
		return mockLogger
	}),
)

// AllMocksModule combines all mock modules for comprehensive testing
var AllMocksModule = fx.Options(
	MockLoggerModule,
	MockPubSubModule,
)

// TestingModuleConfig allows configuring which modules to use for testing
type TestingModuleConfig struct {
	UseMockLogger bool
	UseMockPubSub bool
	UseVerboseLog bool
}

// GetTestModules returns FX modules based on the configuration
func GetTestModules(config TestingModuleConfig) fx.Option {
	var modules []fx.Option

	if config.UseMockLogger {
		if config.UseVerboseLog {
			modules = append(modules, VerboseLoggerModule)
		} else {
			modules = append(modules, MockLoggerModule)
		}
	}

	if config.UseMockPubSub {
		modules = append(modules, MockPubSubModule)
	}

	return fx.Options(modules...)
}