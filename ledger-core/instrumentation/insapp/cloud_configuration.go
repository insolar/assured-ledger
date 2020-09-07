// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insapp

import (
	"crypto"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/insolar/assured-ledger/ledger-core/application/genesisrefs"
	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/secrets"
	"github.com/insolar/assured-ledger/ledger-core/insolar/defaults"
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

func generateNodeKeys(num int, nodes []nodeInfo) error {
	for i := 0; i < num; i++ {
		node, err := makeNodeWithKeys()
		if err != nil {
			return errors.W(err, "couldn't make node with keys")
		}
		nodes[i] = node
	}

	return nil
}

type inMemoryKeyStore struct {
	key crypto.PrivateKey
}

func (ks inMemoryKeyStore) GetPrivateKey(string) (crypto.PrivateKey, error) {
	return ks.key, nil
}
func makeKeyFactory(nodes []nodeInfo) KeyStoreFactory {
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

func generateBaseCloudConfig(nodes []nodeInfo) configuration.BaseCloudConfig {
	pulsarConfig := configuration.NewPulsarConfiguration()
	var bootstrapNodes []string
	for _, el := range nodes {
		bootstrapNodes = append(bootstrapNodes, el.host)
	}
	pulsarConfig.Pulsar.PulseDistributor.BootstrapHosts = bootstrapNodes
	pulsarConfig.KeysPath = "pulsar.yaml"

	return configuration.BaseCloudConfig{
		Log:                 configuration.NewLog(),
		PulsarConfiguration: pulsarConfig,
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

func generateNodeConfigs(nodes []nodeInfo, virtual, light, heavy int) []configuration.Configuration {

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

	appConfigs := []configuration.Configuration{}
	for i := 0; i < len(nodes); i++ {
		role := getRole(&virtual, &light, &heavy)
		nodes[i].role = role.String()

		conf := configuration.NewConfiguration()
		{
			conf.Host.Transport.Address = defaultHost + ":" + strconv.Itoa(netPort)
			nodes[i].host = conf.Host.Transport.Address
			netPort += 1
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

		appConfigs = append(appConfigs, conf)
	}

	return appConfigs
}

func makeCertManagerFactory(certs map[string]*mandates.Certificate) CertManagerFactory {
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

// PrepareCloudConfiguration generates all configs and factories for launching cloud mode
// It doesn't use file system at all. Everything generated in memory
func PrepareCloudConfiguration(virtual, light, heavy int) (
	[]configuration.Configuration,
	configuration.BaseCloudConfig,
	CertManagerFactory,
	KeyStoreFactory,
) {
	totalNum := virtual + light + heavy
	if totalNum < 1 {
		panic("no nodes given")
	}
	nodes := make([]nodeInfo, totalNum)
	err := generateNodeKeys(totalNum, nodes)
	if err != nil {
		panic(throw.W(err, "Failed to gen keys"))
	}

	appConfigs := generateNodeConfigs(nodes, virtual, light, heavy)

	settings := netSettings{
		majorityRule: totalNum,
		minRoles: struct {
			virtual       uint
			lightMaterial uint
			heavyMaterial uint
		}{virtual: uint(virtual), lightMaterial: uint(light), heavyMaterial: uint(heavy)},
	}
	certs, err := generateCertificates(nodes, settings)
	if err != nil {
		panic(throw.W(err, "Failed to gen certificates"))
	}

	baseCloudConf := generateBaseCloudConfig(nodes)

	pulsarKeys, err := makeNodeWithKeys()
	if err != nil {
		panic(throw.W(err, "Failed to gen pulsar keys"))
	}
	pulsarKeys.keyName = baseCloudConf.PulsarConfiguration.KeysPath

	return appConfigs, baseCloudConf, makeCertManagerFactory(certs), makeKeyFactory(append(nodes, pulsarKeys))
}
