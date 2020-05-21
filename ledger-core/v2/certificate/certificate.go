// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package certificate

import (
	"crypto"
	"encoding/json"
	"io"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	errors "github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/v2/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/v2/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/global"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
)

// BootstrapNode holds info about bootstrap nodes
type BootstrapNode struct {
	PublicKey   string `json:"public_key"`
	Host        string `json:"host"`
	NetworkSign []byte `json:"network_sign"`
	NodeSign    []byte `json:"node_sign"`
	NodeRef     string `json:"node_ref"`
	NodeRole    string `json:"node_role"`
	// preprocessed fields
	nodePublicKey crypto.PublicKey
}

func NewBootstrapNode(pubKey crypto.PublicKey, publicKey, host, noderef, role string) *BootstrapNode {
	return &BootstrapNode{
		PublicKey:     publicKey,
		Host:          host,
		NodeRef:       noderef,
		nodePublicKey: pubKey,
		NodeRole:      role,
	}
}

// GetNodeRef returns reference of bootstrap node
func (bn *BootstrapNode) GetNodeRef() reference.Global {
	ref, err := reference.GlobalFromString(bn.NodeRef)
	if err != nil {
		global.Errorf("Invalid bootstrap node reference: %s. Error: %s", bn.NodeRef, err.Error())
		return reference.Global{}
	}
	return ref
}

// GetPublicKey returns public key reference of bootstrap node
func (bn *BootstrapNode) GetPublicKey() crypto.PublicKey {
	return bn.nodePublicKey
}

// GetHost returns host of bootstrap node
func (bn *BootstrapNode) GetHost() string {
	return bn.Host
}

// GetRole returns role of bootstrap node
func (bn *BootstrapNode) GetRole() node.StaticRole {
	return node.GetStaticRoleFromString(bn.NodeRole)
}

// NodeSign returns signed information about some node
func (bn *BootstrapNode) GetNodeSign() []byte {
	return bn.NodeSign
}

var scheme = platformpolicy.NewPlatformCryptographyScheme()

// Certificate holds info about certificate
type Certificate struct {
	AuthorizationCertificate
	MajorityRule int `json:"majority_rule"`
	MinRoles     struct {
		Virtual       uint `json:"virtual"`
		HeavyMaterial uint `json:"heavy_material"`
		LightMaterial uint `json:"light_material"`
	} `json:"min_roles"`
	PulsarPublicKeys []string        `json:"pulsar_public_keys"`
	BootstrapNodes   []BootstrapNode `json:"bootstrap_nodes"`

	// preprocessed fields
	pulsarPublicKey []crypto.PublicKey
}

func newCertificate(publicKey crypto.PublicKey, keyProcessor cryptography.KeyProcessor, data []byte) (*Certificate, error) {
	cert := Certificate{}
	err := json.Unmarshal(data, &cert)
	if err != nil {
		return nil, errors.Wrap(err, "[ newCertificate ] failed to parse certificate json")
	}

	pub, err := keyProcessor.ExportPublicKeyPEM(publicKey)
	if err != nil {
		return nil, errors.Wrap(err, "[ newCertificate ] failed to retrieve public key from node private key")
	}

	if cert.PublicKey != string(pub) {
		return nil, errors.New("[ newCertificate ] Different public keys")
	}

	err = cert.fillExtraFields(keyProcessor)
	if err != nil {
		return nil, errors.Wrap(err, "[ newCertificate ] Incorrect fields")
	}

	cert.DiscoverySigns = make(map[reference.Global][]byte)
	for _, node := range cert.BootstrapNodes {
		cert.DiscoverySigns[node.GetNodeRef()] = node.NodeSign
	}

	return &cert, nil
}

func (cert *Certificate) SerializeNetworkPart() []byte {
	out := strconv.Itoa(cert.MajorityRule) + strconv.Itoa(int(cert.MinRoles.Virtual)) +
		strconv.Itoa(int(cert.MinRoles.HeavyMaterial)) + strconv.Itoa(int(cert.MinRoles.LightMaterial))

	sort.Strings(cert.PulsarPublicKeys)
	out += strings.Join(cert.PulsarPublicKeys, "")
	nodes := make([]string, len(cert.BootstrapNodes))
	for i, node := range cert.BootstrapNodes {
		nodes[i] = node.PublicKey + node.NodeRef + node.Host + node.NodeRole
	}
	sort.Strings(nodes)
	out += strings.Join(nodes, "")

	return []byte(out)
}

// SignNetworkPart signs network part in certificate
func (cert *Certificate) SignNetworkPart(key crypto.PrivateKey) ([]byte, error) {
	signer := scheme.DataSigner(key, scheme.IntegrityHasher())
	sign, err := signer.Sign(cert.SerializeNetworkPart())
	if err != nil {
		return nil, errors.Wrap(err, "[ SignNetworkPart ] Can't Sign")
	}
	return sign.Bytes(), nil
}

func (cert *Certificate) fillExtraFields(keyProcessor cryptography.KeyProcessor) error {
	importedNodePubKey, err := keyProcessor.ImportPublicKeyPEM([]byte(cert.PublicKey))
	if err != nil {
		return errors.Wrapf(err, "[ fillExtraFields ] Bad PublicKey: %s", cert.PublicKey)
	}
	cert.nodePublicKey = importedNodePubKey

	for _, pulsarKey := range cert.PulsarPublicKeys {
		importedPulsarPubKey, err := keyProcessor.ImportPublicKeyPEM([]byte(pulsarKey))
		if err != nil {
			return errors.Wrapf(err, "[ fillExtraFields ] Bad pulsarKey: %s", pulsarKey)
		}
		cert.pulsarPublicKey = append(cert.pulsarPublicKey, importedPulsarPubKey)
	}

	for i := 0; i < len(cert.BootstrapNodes); i++ {
		currentNode := &cert.BootstrapNodes[i]
		importedBNodePubKey, err := keyProcessor.ImportPublicKeyPEM([]byte(currentNode.PublicKey))
		if err != nil {
			return errors.Wrapf(err, "[ fillExtraFields ] Bad Bootstrap PublicKey: %s", currentNode.PublicKey)
		}
		currentNode.nodePublicKey = importedBNodePubKey
	}

	return nil
}

// GetDiscoveryNodes return bootstrap nodes array
func (cert *Certificate) GetDiscoveryNodes() []node.DiscoveryNode {
	result := make([]node.DiscoveryNode, 0)
	for i := 0; i < len(cert.BootstrapNodes); i++ {
		// we get node by pointer, so ranged for loop does not suite
		result = append(result, &cert.BootstrapNodes[i])
	}
	return result
}

func (cert *Certificate) GetMinRoles() (uint, uint, uint) {
	return cert.MinRoles.Virtual, cert.MinRoles.HeavyMaterial, cert.MinRoles.LightMaterial
}

// GetMajorityRule returns majority rule number
func (cert *Certificate) GetMajorityRule() int {
	return cert.MajorityRule
}

// Dump returns all info about certificate in json format
func (cert *Certificate) Dump() (string, error) {
	result, err := json.MarshalIndent(cert, "", "    ")
	if err != nil {
		return "", errors.Wrap(err, "[ Certificate::Dump ]")
	}

	return string(result), nil
}

// ReadCertificate constructor creates new Certificate component
func ReadCertificate(publicKey crypto.PublicKey, keyProcessor cryptography.KeyProcessor, certPath string) (*Certificate, error) {
	data, err := ioutil.ReadFile(filepath.Clean(certPath))
	if err != nil {
		return nil, errors.Wrapf(err, "[ ReadCertificate ] failed to read certificate from: %s", certPath)
	}
	cert, err := newCertificate(publicKey, keyProcessor, data)
	if err != nil {
		return nil, errors.Wrap(err, "[ ReadCertificate ]")
	}
	return cert, nil
}

// ReadCertificateFromReader constructor creates new Certificate component
func ReadCertificateFromReader(publicKey crypto.PublicKey, keyProcessor cryptography.KeyProcessor, reader io.Reader) (*Certificate, error) {
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, errors.Wrap(err, "[ ReadCertificateFromReader ] failed to read certificate data")
	}
	cert, err := newCertificate(publicKey, keyProcessor, data)
	if err != nil {
		return nil, errors.Wrap(err, "[ ReadCertificateFromReader ]")
	}
	return cert, nil
}

// SignCert is used for signing certificate by Discovery node
func SignCert(signer cryptography.Signer, pKey, role, registeredNodeRef string) (*cryptography.Signature, error) {
	data := []byte(pKey + registeredNodeRef + role)
	sign, err := signer.Sign(data)
	if err != nil {
		return nil, errors.Wrap(err, "[ SignCert ] Couldn't sign")
	}
	return sign, nil
}
