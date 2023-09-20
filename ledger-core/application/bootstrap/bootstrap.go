package bootstrap

import (
	"context"
	"crypto"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"strconv"

	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/application"
	"github.com/insolar/assured-ledger/ledger-core/application/genesisrefs"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/network/mandates"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

// Generator is a component for generating bootstrap files required for discovery nodes bootstrap and heavy genesis.
type Generator struct {
	config             *Config
	certificatesOutDir string
}

// NewGenerator parses config file and creates new generator on success.
func NewGenerator(configFile, certificatesOutDir string) (*Generator, error) {
	config, err := ParseConfig(configFile)
	if err != nil {
		return nil, err
	}

	return NewGeneratorWithConfig(config, certificatesOutDir), nil
}

// NewGeneratorWithConfig creates new Generator with provided config.
func NewGeneratorWithConfig(config *Config, certificatesOutDir string) *Generator {
	return &Generator{
		config:             config,
		certificatesOutDir: certificatesOutDir,
	}
}

// Run generates bootstrap data.
//
// 1. builds Go plugins for genesis contracts
//    (gone when built-in contracts (INS-2308) would be implemented)
// 2. read root keys file and generates keys and certificates for discovery nodes.
// 3. generates genesis config for heavy node.
func (g *Generator) Run(ctx context.Context, properNames bool) error {
	fmt.Printf("[ bootstrap ] config:\n%v\n", dumpAsJSON(g.config))

	inslog := inslogger.FromContext(ctx)

	inslog.Info("[ bootstrap ] create discovery keys ...")
	discoveryNodes, err := createKeysInDir(
		ctx,
		g.config.DiscoveryKeysDir,
		g.config.KeysNameFormat,
		g.config.DiscoveryNodes,
		g.config.ReuseKeys,
		properNames,
	)
	if err != nil {
		return errors.W(err, "create keys step failed")
	}

	inslog.Info("[ bootstrap ] create discovery certificates ...")
	err = g.makeCertificates(ctx, discoveryNodes, discoveryNodes)
	if err != nil {
		return errors.W(err, "generate discovery certificates failed")
	}

	if g.config.NotDiscoveryKeysDir != "" {
		inslog.Info("[ bootstrap ] create not discovery keys ...")
		nodes, err := createKeysInDir(
			ctx,
			g.config.NotDiscoveryKeysDir,
			g.config.KeysNameFormat,
			g.config.Nodes,
			g.config.ReuseKeys,
			properNames,
		)
		if err != nil {
			return errors.W(err, "create keys step failed")
		}

		inslog.Info("[ bootstrap ] create not discovery certificates ...", nodes)
		err = g.makeCertificates(ctx, nodes, discoveryNodes)
		if err != nil {
			return errors.W(err, "generate not discovery certificates failed")
		}
	}

	if err := g.makeEmptyGenesisConfig(); err != nil {
		return errors.W(err, "generate empty genesis config failed")
	}

	return nil
}

func (g *Generator) makeEmptyGenesisConfig() error {
	cfg := &application.GenesisHeavyConfig{}
	b, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		return errors.W(err, "failed to decode heavy config to json")
	}

	err = ioutil.WriteFile(g.config.HeavyGenesisConfigFile, b, 0600)
	return errors.Wrapf(err,
		"failed to write heavy config %v", g.config.HeavyGenesisConfigFile)
}

type nodeInfo struct {
	privateKey crypto.PrivateKey
	publicKey  string
	role       string
	certName   string
}

func (ni nodeInfo) reference() reference.Global {
	return genesisrefs.GenesisRef(ni.publicKey)
}

func (g *Generator) makeCertificates(ctx context.Context, nodesInfo []nodeInfo, discoveryNodes []nodeInfo) error {
	certs := make([]mandates.Certificate, 0, len(g.config.DiscoveryNodes))
	for _, node := range nodesInfo {
		c := mandates.Certificate{
			AuthorizationCertificate: mandates.AuthorizationCertificate{
				PublicKey: node.publicKey,
				Role:      node.role,
				Reference: node.reference().String(),
			},
			MajorityRule: g.config.MajorityRule,
		}
		c.MinRoles.Virtual = g.config.MinRoles.Virtual
		c.MinRoles.HeavyMaterial = g.config.MinRoles.HeavyMaterial
		c.MinRoles.LightMaterial = g.config.MinRoles.LightMaterial
		c.BootstrapNodes = []mandates.BootstrapNode{}

		for j, n2 := range discoveryNodes {
			host := g.config.DiscoveryNodes[j].Host
			c.BootstrapNodes = append(c.BootstrapNodes, mandates.BootstrapNode{
				PublicKey: n2.publicKey,
				Host:      host,
				NodeRef:   n2.reference().String(),
				NodeRole:  n2.role,
			})
		}

		certs = append(certs, c)
	}

	var err error
	for i, node := range nodesInfo {
		for j := range g.config.DiscoveryNodes {
			dn := discoveryNodes[j]

			certs[i].BootstrapNodes[j].NetworkSign, err = certs[i].SignNetworkPart(dn.privateKey)
			if err != nil {
				return errors.Wrapf(err, "can't SignNetworkPart for %s",
					dn.reference())
			}

			certs[i].BootstrapNodes[j].NodeSign, err = certs[i].SignNodePart(dn.privateKey)
			if err != nil {
				return errors.Wrapf(err, "can't SignNodePart for %s",
					dn.reference())
			}
		}

		// save cert to disk
		cert, err := json.MarshalIndent(certs[i], "", "  ")
		if err != nil {
			return errors.W(err, "can't MarshalIndent")
		}

		if len(node.certName) == 0 {
			return errors.New("cert_name must not be empty for node number " + strconv.Itoa(i+1))
		}

		certFile := path.Join(g.certificatesOutDir, node.certName)
		err = ioutil.WriteFile(certFile, cert, 0600)
		if err != nil {
			return errors.Wrapf(err, "failed to create certificate: %v", certFile)
		}

		inslogger.FromContext(ctx).Infof("[ bootstrap ] write certificate file: %v", certFile)
	}
	return nil
}

func dumpAsJSON(data interface{}) string {
	b, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		panic(err)
	}
	return string(b)
}
