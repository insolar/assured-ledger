// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package launchnet

import (
	"crypto"
	"fmt"
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
	"github.com/insolar/assured-ledger/ledger-core/log/logoutput"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/mandates"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func makeNodeWithKeys() (nodeInfo, error) {
	pair, err := secrets.GenerateKeyPair()

	if err != nil {
		return nodeInfo{}, errors.W(err, "couldn't generate keys")
	}

	ks := platformpolicy.NewKeyProcessor()
	if err != nil {
		return nodeInfo{}, errors.W(err, "couldn't export private key")
	}

	pubKeyStr, err := ks.ExportPublicKeyPEM(pair.Public)
	if err != nil {
		return nodeInfo{}, errors.W(err, "couldn't export public key")
	}

	var node nodeInfo
	node.publicKey = string(pubKeyStr)
	node.privateKey = pair.Private

	return node, nil
}

func generateNodeKeys(num int) ([]nodeInfo, error) {
	nodes := make([]nodeInfo, 0, num)
	for i := 0; i < num; i++ {
		node, err := makeNodeWithKeys()
		if err != nil {
			return nil, errors.W(err, "couldn't make node with keys")
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}

type inMemoryKeyStore struct {
	key crypto.PrivateKey
}

func (ks inMemoryKeyStore) GetPrivateKey(string) (crypto.PrivateKey, error) {
	return ks.key, nil
}
func makeKeyFactory(nodes []nodeInfo) insapp.KeyStoreFactoryFunc {
	keysMap := make(map[string]crypto.PrivateKey)
	for _, n := range nodes {
		if _, ok := keysMap[n.keyName]; ok {
			panic("Key duplicate: " + n.keyName)
		}
		keysMap[n.keyName] = n.privateKey
	}

	return func(path string) (cryptography.KeyStore, error) {
		if _, ok := keysMap[path]; !ok {
			panic("NO KEY: " + path)
		}
		return &inMemoryKeyStore{keysMap[path]}, nil
	}
}

func generatePulsarConfig(nodes []nodeInfo, settings CloudSettings) configuration.PulsarConfiguration {
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

	if cloudFileLogging {
		pulsarConfig.Log.OutputType = logoutput.FileOutput.String()
		pulsarConfig.Log.OutputParams = launchnetPath("logs", "pulsar.log")
	}

	return pulsarConfig
}

func generateBaseCloudConfig(nodes []nodeInfo, settings CloudSettings) configuration.BaseCloudConfig {
	return configuration.BaseCloudConfig{
		Log:                 configuration.NewLog(),
		PulsarConfiguration: generatePulsarConfig(nodes, settings),
	}

}

func getRole(virtual, light, heavy *int) member.PrimaryRole {
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

func generateNodeConfigs(nodes []nodeInfo, cloudSettings CloudSettings) []configuration.Configuration {
	var (
		metricsPort      = 8001
		LRRPCPort        = 13001
		APIPort          = 18001
		adminAPIPort     = 23001
		testWalletPort   = 33001
		introspectorPort = 38001
		netPort          = 43001

		defaultHost     = "127.0.0.1"
		certificatePath = "cert_%d.json"
		keyPath         = "node_%d.json"
	)

	if cloudSettings.API.TestWalletAPIPortStart != 0 {
		testWalletPort = cloudSettings.API.TestWalletAPIPortStart
	}
	if cloudSettings.API.AdminPort != 0 {
		adminAPIPort = cloudSettings.API.AdminPort
	}

	appConfigs := make([]configuration.Configuration, 0, len(nodes))
	for i := 0; i < len(nodes); i++ {
		role := getRole(&cloudSettings.Virtual, &cloudSettings.Light, &cloudSettings.Heavy)
		nodes[i].role = role.String()

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
			conf.KeysPath = fmt.Sprintf(keyPath, i+1)
			conf.CertificatePath = fmt.Sprintf(certificatePath, i+1)
			nodes[i].certName = conf.CertificatePath
			nodes[i].keyName = conf.KeysPath
		}

		if cloudFileLogging {
			// prepare directory for logs
			if err := os.MkdirAll(launchnetPath("logs", "discoverynodes", strconv.Itoa(i+1)), os.ModePerm); err != nil {
				panic(err)
			}

			conf.Log.OutputType = logoutput.FileOutput.String()
			conf.Log.OutputParams = launchnetPath("logs", "discoverynodes", strconv.Itoa(i+1), "output.log")
		}

		appConfigs = append(appConfigs, conf)
	}

	return appConfigs
}

func makeCertManagerFactory(certs map[string]*mandates.Certificate) insapp.CertManagerFactoryFunc {
	return func(publicKey crypto.PublicKey, keyProcessor cryptography.KeyProcessor, certPath string) (*mandates.CertificateManager, error) {
		return mandates.NewCertificateManager(certs[certPath]), nil
	}
}

type nodeInfo struct {
	privateKey crypto.PrivateKey
	publicKey  string
	role       string
	host       string
	certName   string
	keyName    string
}

func (ni nodeInfo) reference() reference.Global {
	return genesisrefs.GenesisRef(ni.publicKey)
}

type netSettings struct {
	majorityRule int
	minRoles     struct {
		virtual       uint
		lightMaterial uint
		heavyMaterial uint
	}
}

func generateCertificates(nodesInfo []nodeInfo, settings netSettings) (map[string]*mandates.Certificate, error) {
	certs := make(map[string]*mandates.Certificate)
	kp := platformpolicy.NewKeyProcessor()
	for _, node := range nodesInfo {
		c := &mandates.Certificate{
			AuthorizationCertificate: mandates.AuthorizationCertificate{
				PublicKey: node.publicKey,
				Role:      node.role,
				Reference: node.reference().String(),
			},
			MajorityRule: settings.majorityRule,
		}

		c.MinRoles.Virtual = settings.minRoles.virtual
		c.MinRoles.HeavyMaterial = settings.minRoles.heavyMaterial
		c.MinRoles.LightMaterial = settings.minRoles.lightMaterial
		c.BootstrapNodes = []mandates.BootstrapNode{}

		for _, n2 := range nodesInfo {
			c.BootstrapNodes = append(c.BootstrapNodes, mandates.BootstrapNode{
				PublicKey: n2.publicKey,
				Host:      n2.host,
				NodeRef:   n2.reference().String(),
				NodeRole:  n2.role,
			})
		}

		certs[node.certName] = c
	}

	var err error
	for i := range nodesInfo {
		for j := range nodesInfo {
			dn := nodesInfo[j]

			certName := nodesInfo[i].certName

			certs[certName].BootstrapNodes[j].NetworkSign, err = certs[certName].SignNetworkPart(dn.privateKey)
			if err != nil {
				return nil, throw.W(err, "can't SignNetworkPart for %s",
					dn.reference())
			}

			certs[certName].BootstrapNodes[j].NodeSign, err = certs[certName].SignNodePart(dn.privateKey)
			if err != nil {
				return nil, throw.W(err, "can't SignNodePart for %s",
					dn.reference())
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

type CloudSettings struct {
	Virtual, Light, Heavy int

	API struct {
		TestWalletAPIPortStart int
		AdminPort              int
	}
	Pulsar struct {
		PulseTime int
	}
}

// PrepareCloudConfiguration generates all configs and factories for launching cloud mode
// It doesn't use file system at all. Everything generated in memory
func PrepareCloudConfiguration(cloudSettings CloudSettings) (
	[]configuration.Configuration,
	configuration.BaseCloudConfig,
	insapp.CertManagerFactoryFunc,
	insapp.KeyStoreFactoryFunc,
) {
	totalNum := cloudSettings.Virtual + cloudSettings.Light + cloudSettings.Heavy
	if totalNum < 1 {
		panic("no nodes given")
	}
	nodes, err := generateNodeKeys(totalNum)
	if err != nil {
		panic(throw.W(err, "Failed to gen keys"))
	}

	appConfigs := generateNodeConfigs(nodes, cloudSettings)

	settings := netSettings{
		majorityRule: totalNum,
		minRoles: struct {
			virtual       uint
			lightMaterial uint
			heavyMaterial uint
		}{virtual: uint(cloudSettings.Virtual), lightMaterial: uint(cloudSettings.Light), heavyMaterial: uint(cloudSettings.Heavy)},
	}
	certs, err := generateCertificates(nodes, settings)
	if err != nil {
		panic(throw.W(err, "Failed to gen certificates"))
	}

	baseCloudConf := generateBaseCloudConfig(nodes, cloudSettings)

	pulsarKeys, err := makeNodeWithKeys()
	if err != nil {
		panic(throw.W(err, "Failed to gen pulsar keys"))
	}
	pulsarKeys.keyName = baseCloudConf.PulsarConfiguration.KeysPath

	return appConfigs, baseCloudConf, makeCertManagerFactory(certs), makeKeyFactory(append(nodes, pulsarKeys))
}
