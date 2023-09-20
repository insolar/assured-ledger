package cloud

import (
	"crypto"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/insolar/assured-ledger/ledger-core/application/genesisrefs"
	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/secrets"
	"github.com/insolar/assured-ledger/ledger-core/insolar/defaults"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insapp"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/log/logoutput"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/mandates"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func preparePulsarKeys(cfg configuration.BaseCloudConfig) nodeInfo {
	pulsarKeys, err := makeNodeWithKeys()
	if err != nil {
		panic(throw.W(err, "Failed to gen pulsar keys"))
	}
	pulsarKeys.stringReference = cfg.PulsarConfiguration.KeysPath // hack for pulsar keys
	return pulsarKeys
}

func makeNodeWithKeys() (nodeInfo, error) {
	pair, err := secrets.GenerateKeyPair()

	if err != nil {
		return nodeInfo{}, throw.W(err, "couldn't generate keys")
	}

	ks := platformpolicy.NewKeyProcessor()
	if err != nil {
		return nodeInfo{}, throw.W(err, "couldn't export private key")
	}

	pubKeyStr, err := ks.ExportPublicKeyPEM(pair.Public)
	if err != nil {
		return nodeInfo{}, throw.W(err, "couldn't export public key")
	}

	var node nodeInfo
	node.publicKey = string(pubKeyStr)
	node.privateKey = pair.Private
	node.reference = genesisrefs.GenesisRef(node.publicKey)
	node.stringReference = node.reference.String()

	return node, nil
}

func generateNodeKeys(cfg *NodeConfiguration) ([]nodeInfo, error) {
	num := cfg.Virtual + cfg.LightMaterial + cfg.HeavyMaterial
	nodes := make([]nodeInfo, 0, num)
	for i := uint(0); i < num; i++ {
		node, err := makeNodeWithKeys()
		if err != nil {
			return nil, throw.W(err, "couldn't make node with keys")
		}
		node.role = getRole(&cfg.Virtual, &cfg.LightMaterial, &cfg.HeavyMaterial)
		nodes = append(nodes, node)
	}

	return nodes, nil
}

func makeKeyFactory(nodes []nodeInfo) insapp.KeyStoreFactoryFunc {
	keysMap := make(map[string]crypto.PrivateKey)
	for _, n := range nodes {
		if _, ok := keysMap[n.stringReference]; ok {
			panic("Key duplicate: " + n.reference.String())
		}
		keysMap[n.stringReference] = n.privateKey
	}

	return func(path string) (cryptography.KeyStore, error) {
		if _, ok := keysMap[path]; !ok {
			panic("NO KEY: " + path)
		}
		return &InMemoryKeyStore{Key: keysMap[path]}, nil
	}
}

func getRole(virtual, light, heavy *uint) member.PrimaryRole {
	switch {
	case *virtual > 0:
		*virtual--
		return member.PrimaryRoleVirtual
	case *light > 0:
		*light--
		return member.PrimaryRoleLightMaterial
	case *heavy > 0:
		*heavy--
		return member.PrimaryRoleHeavyMaterial
	default:
		panic(throw.IllegalValue())
	}
}

func generateNodeConfigs(nodes []nodeInfo, cloudSettings Settings) map[reference.Global]configuration.Configuration {
	var (
		metricsPort      = 8001
		LRRPCPort        = 13001
		APIPort          = 18001
		adminAPIPort     = 19001
		testWalletPort   = 33001
		introspectorPort = 38001
		netPort          = 43001

		defaultHost = "127.0.0.1"
		logLevel    = log.DebugLevel.String()
	)

	if cloudSettings.Log.Level != "" {
		logLevel = cloudSettings.Log.Level
	}

	appConfigs := make(map[reference.Global]configuration.Configuration, len(nodes))
	for i := 0; i < len(nodes); i++ {
		conf := configuration.NewConfiguration()
		conf.AvailabilityChecker.Enabled = false
		{
			conf.Host.Transport.Address = defaultHost + ":" + strconv.Itoa(netPort)
			nodes[i].host = conf.Host.Transport.Address
			netPort++
		}
		{
			conf.Metrics.ListenAddress = defaultHost + ":" + strconv.Itoa(metricsPort)
			metricsPort++
		}
		{
			conf.LogicRunner.RPCListen = defaultHost + ":" + strconv.Itoa(LRRPCPort)
			conf.LogicRunner.GoPlugin.RunnerListen = defaultHost + ":" + strconv.Itoa(LRRPCPort+1)
			LRRPCPort += 2
		}
		{
			conf.APIRunner.Address = defaultHost + ":" + strconv.Itoa(APIPort)
			conf.APIRunner.SwaggerPath = filepath.Join(defaults.RootModuleDir(), conf.APIRunner.SwaggerPath)
			APIPort++

			conf.AdminAPIRunner.Address = defaultHost + ":" + strconv.Itoa(adminAPIPort)
			conf.AdminAPIRunner.SwaggerPath = filepath.Join(defaults.RootModuleDir(), conf.AdminAPIRunner.SwaggerPath)
			adminAPIPort++

			conf.TestWalletAPI.Address = defaultHost + ":" + strconv.Itoa(testWalletPort)
			testWalletPort++
		}
		{
			conf.Introspection.Addr = defaultHost + ":" + strconv.Itoa(introspectorPort)
			introspectorPort++
		}
		{
			conf.KeysPath = nodes[i].reference.String()
			conf.CertificatePath = nodes[i].reference.String()
		}
		conf.Log.Level = logLevel

		if cloudSettings.CloudFileLogging {
			// prepare directory for logs
			if err := os.MkdirAll(filepath.Join(defaults.RootModuleDir(), ".artifacts", "launchnet", "logs", "discoverynodes", strconv.Itoa(i+1)), os.ModePerm); err != nil {
				panic(err)
			}

			conf.Log.OutputType = logoutput.FileOutput.String()
			conf.Log.OutputParams = filepath.Join(defaults.RootModuleDir(), ".artifacts", "launchnet", "logs", "discoverynodes", strconv.Itoa(i+1), "output.log")
		}

		appConfigs[nodes[i].reference] = conf
	}

	return appConfigs
}

func makeCertManagerFactory(certs map[string]*mandates.Certificate) insapp.CertManagerFactoryFunc {
	return func(publicKey crypto.PublicKey, keyProcessor cryptography.KeyProcessor, certPath string) (*mandates.CertificateManager, error) {
		return mandates.NewCertificateManager(certs[certPath]), nil
	}
}

type nodeInfo struct {
	stringReference string

	reference reference.Global

	privateKey crypto.PrivateKey
	publicKey  string
	role       member.PrimaryRole
	host       string
}

type netSettings struct {
	majorityRule int
	minRoles     NodeConfiguration
}

func generateCertificates(nodesInfo []nodeInfo, settings netSettings) (map[string]*mandates.Certificate, error) {
	certs := make(map[string]*mandates.Certificate)
	kp := platformpolicy.NewKeyProcessor()
	for _, node := range nodesInfo {
		c := &mandates.Certificate{
			AuthorizationCertificate: mandates.AuthorizationCertificate{
				PublicKey: node.publicKey,
				Role:      node.role.String(),
				Reference: node.reference.String(),
			},
			MajorityRule: settings.majorityRule,
		}

		c.MinRoles.Virtual = settings.minRoles.Virtual
		c.MinRoles.HeavyMaterial = settings.minRoles.HeavyMaterial
		c.MinRoles.LightMaterial = settings.minRoles.LightMaterial
		c.BootstrapNodes = []mandates.BootstrapNode{}

		for _, n2 := range nodesInfo {
			c.BootstrapNodes = append(c.BootstrapNodes, mandates.BootstrapNode{
				PublicKey: n2.publicKey,
				Host:      n2.host,
				NodeRef:   n2.reference.String(),
				NodeRole:  n2.role.String(),
			})
		}

		certs[node.reference.String()] = c
	}

	var err error
	for i := range nodesInfo {
		for j := range nodesInfo {
			dn := nodesInfo[j]

			certName := nodesInfo[i].reference.String()

			certs[certName].BootstrapNodes[j].NetworkSign, err = certs[certName].SignNetworkPart(dn.privateKey)
			if err != nil {
				return nil, throw.W(err, "can't SignNetworkPart for %s",
					dn.reference)
			}

			certs[certName].BootstrapNodes[j].NodeSign, err = certs[certName].SignNodePart(dn.privateKey)
			if err != nil {
				return nil, throw.W(err, "can't SignNodePart for %s",
					dn.reference)
			}
		}
	}

	// Required to fill internal fields of Certificate
	for key, cert := range certs {
		certRaw, err := cert.Dump()
		if err != nil {
			return nil, throw.W(err, "can't dump cert for %s",
				cert.Reference)
		}
		publicKey, err := kp.ImportPublicKeyPEM([]byte(cert.PublicKey))
		if err != nil {
			return nil, throw.W(err, "can't import pub key %s",
				cert.Reference)
		}
		fullCert, err := mandates.ReadCertificateFromReader(publicKey, kp, strings.NewReader(certRaw))
		if err != nil {
			return nil, throw.W(err, "can't reread cert for %s",
				cert.Reference)
		}
		certs[key] = fullCert
	}

	return certs, nil
}

// prepareCloudConfiguration generates all configs and factories for launching cloud mode
// It doesn't use file system at all. Everything generated in memory
func prepareCloudConfiguration(cloudSettings Settings) (
	map[reference.Global]configuration.Configuration,
	configuration.BaseCloudConfig,
	map[string]*mandates.Certificate,
	[]nodeInfo,
) {
	nodes, err := generateNodeKeys(&cloudSettings.Prepared)
	if err != nil {
		panic(throw.W(err, "Failed to gen keys"))
	}

	settings := netSettings{
		majorityRule: cloudSettings.MajorityRule,
		minRoles:     cloudSettings.MinRoles}
	certs, err := generateCertificates(nodes, settings)
	if err != nil {
		panic(throw.W(err, "Failed to gen certificates"))
	}

	baseCloudConf := generateBaseCloudConfig(nodes, cloudSettings)

	appConfigs := generateNodeConfigs(nodes, cloudSettings)

	return appConfigs, baseCloudConf, certs, nodes
}

func generatePulsarConfig(nodes []nodeInfo, settings Settings) configuration.PulsarConfiguration {
	pulsarConfig := configuration.NewPulsarConfiguration()
	var bootstrapNodes []string
	for _, el := range nodes {
		bootstrapNodes = append(bootstrapNodes, el.host)
	}
	pulsarConfig.Pulsar.PulseDistributor.BootstrapHosts = bootstrapNodes
	pulsarConfig.KeysPath = "pulsar.yaml"
	if settings.Pulsar.PulseTime != 0 {
		pulsarConfig.Pulsar.PulseTime = int32(settings.Pulsar.PulseTime)
	}

	if settings.CloudFileLogging {
		pulsarConfig.Log.OutputType = logoutput.FileOutput.String()
		pulsarConfig.Log.OutputParams = filepath.Join(defaults.RootModuleDir(), ".artifacts", "launchnet", "logs", "pulsar.log")
	}

	return pulsarConfig
}

func generateBaseCloudConfig(nodes []nodeInfo, settings Settings) configuration.BaseCloudConfig {
	return configuration.BaseCloudConfig{
		Log:                 configuration.NewLog(),
		PulsarConfiguration: generatePulsarConfig(nodes, settings),
	}
}
