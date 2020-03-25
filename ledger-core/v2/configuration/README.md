Insolar – Configuration
===============

[![GoDoc](https://godoc.org/github.com/insolar/assured-ledger/ledger-core/v2/configuration?status.svg)](https://godoc.org/github.com/insolar/assured-ledger/ledger-core/v2/configuration)


Package provides configuration params for all Insolar components and helper for config resources management.

### Configuration

Configuration struct is a root registry for all components config.
It provides constructor method `NewConfiguration()` which creates new instance of configuration object filled with default values.

Each root level Insolar component has a constructor with config as argument.
Each components should have its own config struct with the same name in this package.
Each config struct should have constructor which returns instance with default params.

### Holder

Package also provides [Holder](https://godoc.org/github.com/insolar/assured-ledger/ledger-core/v2/configuration#Holder) to easily manage config resources. 
It based on [insconfig package](https://github.com/insolar/insconfig) and helps to Marshal\Unmarshal config structs, manage files, ENV and command line variables.

Holder provides functionality to merge configuration from different sources.

