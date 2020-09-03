// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

import (
	"context"
	"crypto"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/application/genesisrefs"
	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/secrets"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insapp"
	"github.com/insolar/assured-ledger/ledger-core/network/mandates"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/server"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func generateKeys(num int, nodes *[]nodeInfo) error {
	for i := 0; i < num; i++ {
		pair, err := secrets.GenerateKeyPair()

		if err != nil {
			return errors.W(err, "couldn't generate keys")
		}

		ks := platformpolicy.NewKeyProcessor()
		if err != nil {
			return errors.W(err, "couldn't export private key")
		}

		pubKeyStr, err := ks.ExportPublicKeyPEM(pair.Public)
		if err != nil {
			return errors.W(err, "couldn't export public key")
		}

		(*nodes)[i].publicKey = string(pubKeyStr)
		(*nodes)[i].privateKey = pair.Private
	}

	return nil
}

type inMemoryKeyStore struct {
	key crypto.PrivateKey
}

func (ks inMemoryKeyStore) GetPrivateKey(string) (crypto.PrivateKey, error) {
	return ks.key, nil
}
func makeKeyFactory(nodes []nodeInfo) insapp.KeyStoreFactory {
	keysMap := make(map[string]crypto.PrivateKey)
	for _, n := range nodes {
		keysMap[n.keyName] = n.privateKey
	}

	return func(path string) (cryptography.KeyStore, error) {
		if _, ok := keysMap[path]; !ok {
			panic("NO KEY: " + path)
		}
		return &inMemoryKeyStore{keysMap[path]}, nil
	}
}

func prepareStuff(num int, dataPath string) ([]configuration.Configuration, insapp.CertManagerFactory, insapp.KeyStoreFactory) {
	nodes := make([]nodeInfo, num)
	err := generateKeys(num, &nodes)
	if err != nil {
		panic(throw.W(err, "Failed to gen keys"))
	}

	appConfigs := makeConfigs(num, dataPath, &nodes)

	settings := netSettings{
		majorityRule: num,
		minRoles: struct {
			virtual       uint
			lightMaterial uint
			heavyMaterial uint
		}{virtual: uint(num), lightMaterial: 0, heavyMaterial: 0},
	}
	certs, err := generateCertificates(nodes, settings)
	if err != nil {
		panic(throw.W(err, "Failed to gen certificates"))
	}
	return appConfigs, makeCertManagerFactory(certs), makeKeyFactory(nodes)
}

func Test_RunMulti(t *testing.T) {
	var multiFn insapp.MultiNodeConfigFunc

	dataPath := "/Users/ivansibitov/go/src/github.com/insolar/assured-ledger/ledger-core/"
	NUM_NODES := 5

	appConfigs, certFactory, keyFactory := prepareStuff(NUM_NODES, dataPath)
	multiFn = func(cfgPath string, baseCfg configuration.Configuration) ([]configuration.Configuration, insapp.NetworkInitFunc) {
		return appConfigs, nil
	}

	s := server.NewMultiServer(dataPath+"TEST_CONF.yaml", multiFn, certFactory, keyFactory)
	s.Serve()
}

func makeConfigs(numNodes int, dataPath string, nodes *[]nodeInfo) []configuration.Configuration {

	var (
		metricsPortStart      = 8001
		LRRPCPortStart        = 13001
		APIPortStart          = 18001
		adminAPIPortStart     = 23001
		testWalletPortStart   = 33001
		introspectorPortStart = 38001
		netPortStart          = 43001

		defaultHost     = "127.0.0.1"
		certificatePath = "cert_%d.json"
		keyPath         = "node_%d.json"
	)

	origNodes := *nodes

	appConfigs := []configuration.Configuration{}
	for i := 0; i < numNodes; i++ {
		origNodes[i].role = "virtual"

		conf := configuration.NewConfiguration()
		{
			conf.Host.Transport.Address = defaultHost + ":" + strconv.Itoa(netPortStart)
			origNodes[i].host = conf.Host.Transport.Address
			netPortStart += 1
		}
		{
			conf.Metrics.ListenAddress = defaultHost + ":" + strconv.Itoa(metricsPortStart)
			metricsPortStart++
		}
		{
			conf.LogicRunner.RPCListen = defaultHost + ":" + strconv.Itoa(LRRPCPortStart)
			conf.LogicRunner.GoPlugin.RunnerListen = defaultHost + ":" + strconv.Itoa(LRRPCPortStart+1)
			LRRPCPortStart += 2
		}
		{
			conf.APIRunner.Address = defaultHost + ":" + strconv.Itoa(APIPortStart)
			conf.APIRunner.SwaggerPath = dataPath + conf.APIRunner.SwaggerPath
			APIPortStart++

			conf.AdminAPIRunner.Address = defaultHost + ":" + strconv.Itoa(adminAPIPortStart)
			conf.AdminAPIRunner.SwaggerPath = dataPath + conf.AdminAPIRunner.SwaggerPath
			adminAPIPortStart++

			conf.TestWalletAPI.Address = defaultHost + ":" + strconv.Itoa(testWalletPortStart)
			testWalletPortStart++
		}
		{
			conf.Introspection.Addr = defaultHost + ":" + strconv.Itoa(introspectorPortStart)
			introspectorPortStart++
		}
		{
			conf.KeysPath = fmt.Sprintf(keyPath, i+1)
			conf.CertificatePath = fmt.Sprintf(certificatePath, i+1)
			origNodes[i].certName = conf.CertificatePath
			origNodes[i].keyName = conf.KeysPath
		}

		appConfigs = append(appConfigs, conf)
	}

	return appConfigs
}

func makeCertManagerFactory(certs map[string]*mandates.Certificate) insapp.CertManagerFactory {
	return func(ctx context.Context, certPath string, comps insapp.PreComponents) nodeinfo.CertificateManager {
		return mandates.NewCertificateManager(certs[certPath])
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
