package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

var (
	// ErrInvalidDID invalid did format.
	ErrInvalidDID = errors.New("invalid did format")
	// ErrDIDMethodNotSupported unsupported did method.
	ErrDIDMethodNotSupported = errors.New("not supported did method")
	// ErrBlockchainNotSupportedForDID unsupported network for did.
	ErrBlockchainNotSupportedForDID = errors.New("not supported blockchain")
	// ErrNetworkNotSupportedForDID unsupported network for did.
	ErrNetworkNotSupportedForDID = errors.New("not supported network")
)

// DIDSchema DID Schema
const DIDSchema = "did"

// DIDMethod represents did methods
type DIDMethod string

const (
	// DIDMethodIden3 DID method-name
	DIDMethodIden3 DIDMethod = "iden3"
	// DIDMethodPolygonID DID method-name
	DIDMethodPolygonID DIDMethod = "polygonid"
)

// Blockchain id of the network "eth", "polygon", etc.
type Blockchain string

const (
	// Ethereum is ethereum blockchain network
	Ethereum Blockchain = "eth"
	// Polygon is polygon blockchain network
	Polygon Blockchain = "polygon"
	// UnknownChain is used when it's not possible to retrieve blockchain type from identifier
	UnknownChain Blockchain = "unknown"
	// ReadOnly should be used for readonly identity to build readonly flag
	ReadOnly Blockchain = "readonly"
	// NoChain can be used for identity to build readonly flag
	NoChain Blockchain = ""
)

// NetworkID is method specific network identifier
type NetworkID string

const (
	// Main is ethereum main network
	Main NetworkID = "main"
	// Mumbai is polygon mumbai test network
	Mumbai NetworkID = "mumbai"

	// Goerli is ethereum goerli test network
	Goerli NetworkID = "goerli" // goerli
	// UnknownNetwork is used when it's not possible to retrieve network from identifier
	UnknownNetwork NetworkID = "unknown"

	// NoNetwork should be used for readonly identity to build readonly flag
	NoNetwork NetworkID = ""
)

// DIDMethodByte did method flag representation
var DIDMethodByte = map[DIDMethod]byte{
	DIDMethodIden3:     0b00000001,
	DIDMethodPolygonID: 0b00000010,
}

// DIDNetworkFlag is a structure to represent DID blockchain and network id
type DIDNetworkFlag struct {
	Blockchain Blockchain
	NetworkID  NetworkID
}

// DIDMethodNetwork is map for did methods and their blockchain networks
var DIDMethodNetwork = map[DIDMethod]map[DIDNetworkFlag]byte{
	DIDMethodIden3: {
		{Blockchain: ReadOnly, NetworkID: NoNetwork}: 0b00000000,

		{Blockchain: Polygon, NetworkID: Main}:   0b00010000 | 0b00000001,
		{Blockchain: Polygon, NetworkID: Mumbai}: 0b00010000 | 0b00000010,

		{Blockchain: Ethereum, NetworkID: Main}:   0b00100000 | 0b00000001,
		{Blockchain: Ethereum, NetworkID: Goerli}: 0b00100000 | 0b00000010,
	},
	DIDMethodPolygonID: {
		{Blockchain: ReadOnly, NetworkID: NoNetwork}: 0b00000000,

		{Blockchain: Polygon, NetworkID: Main}:   0b00010000 | 0b00000001,
		{Blockchain: Polygon, NetworkID: Mumbai}: 0b00010000 | 0b00000010,

		{Blockchain: Ethereum, NetworkID: Main}:   0b00100000 | 0b00000001,
		{Blockchain: Ethereum, NetworkID: Goerli}: 0b00100000 | 0b00000010,
	},
}

// BuildDIDType builds bytes type from chain and network
func BuildDIDType(method DIDMethod, blockchain Blockchain, network NetworkID) ([2]byte, error) {

	fb, ok := DIDMethodByte[method]
	if !ok {
		return [2]byte{}, ErrDIDMethodNotSupported
	}

	if blockchain == NoChain {
		blockchain = ReadOnly
	}

	sb, ok := DIDMethodNetwork[method][DIDNetworkFlag{Blockchain: blockchain, NetworkID: network}]
	if !ok {
		return [2]byte{}, ErrNetworkNotSupportedForDID
	}
	return [2]byte{fb, sb}, nil
}

// BuildDIDTypeOnChain builds bytes type from chain and network
func BuildDIDTypeOnChain(method DIDMethod, blockchain Blockchain, network NetworkID) ([2]byte, error) {
	typ, err := BuildDIDType(method, blockchain, network)
	if err != nil {
		return [2]byte{}, err
	}

	// set on-chain flag (first bit of first byte) to 1
	typ[0] |= MethodOnChainFlag

	return typ, nil
}

// FindNetworkIDForDIDMethodByValue finds network by byte value
func FindNetworkIDForDIDMethodByValue(method DIDMethod, _v byte) (NetworkID, error) {
	_, ok := DIDMethodNetwork[method]
	if !ok {
		return UnknownNetwork, ErrDIDMethodNotSupported
	}
	for k, v := range DIDMethodNetwork[method] {
		if v == _v {
			return k.NetworkID, nil
		}
	}
	return UnknownNetwork, ErrNetworkNotSupportedForDID
}

// FindBlockchainForDIDMethodByValue finds blockchain type by byte value
func FindBlockchainForDIDMethodByValue(method DIDMethod, _v byte) (Blockchain, error) {
	_, ok := DIDMethodNetwork[method]
	if !ok {
		return UnknownChain, ErrDIDMethodNotSupported
	}
	for k, v := range DIDMethodNetwork[method] {
		if v == _v {
			return k.Blockchain, nil
		}
	}
	return UnknownChain, ErrNetworkNotSupportedForDID
}

// FindDIDMethodByValue finds did method by its byte value
func FindDIDMethodByValue(_v byte) (DIDMethod, error) {
	for k, v := range DIDMethodByte {
		if v == _v {
			return k, nil
		}
	}
	return "", ErrDIDMethodNotSupported
}

func (did *DID) UnmarshalJSON(bytes []byte) error {
	var didStr string
	err := json.Unmarshal(bytes, &didStr)
	if err != nil {
		return err
	}

	did3, err := Parse(didStr)
	if err != nil {
		return err
	}
	*did = *did3
	return nil
}

func (did DID) MarshalJSON() ([]byte, error) {
	return json.Marshal(did.String())
}

// DIDGenesisFromIdenState calculates the genesis ID from an Identity State and returns it as DID
func DIDGenesisFromIdenState(typ [2]byte, state *big.Int) (*DID, error) {
	id, err := IdGenesisFromIdenState(typ, state)
	if err != nil {
		return nil, err
	}
	return ParseDIDFromID(*id)
}

func Decompose(did DID) (Blockchain, NetworkID, ID, error) {
	id, err := decodeIDFromDID(did)
	if err != nil {
		return UnknownChain, UnknownNetwork, id, err
	}

	method, blockchain, networkID, err := decodeDIDPartsFromID(id)
	if err != nil {
		return UnknownChain, UnknownNetwork, id, err
	}

	if string(method) != did.Method {
		return UnknownChain, UnknownNetwork, id,
			fmt.Errorf("%w: method mismatch: found %v in ID but %v in DID",
				ErrInvalidDID, method, did.Method)
	}

	if len(did.IDStrings) > 1 && string(blockchain) != did.IDStrings[0] {
		return UnknownChain, UnknownNetwork, id,
			fmt.Errorf("%w: blockchain mismatch: found %v in ID but %v in DID",
				ErrInvalidDID, blockchain, did.IDStrings[0])
	}

	if len(did.IDStrings) > 2 && string(networkID) != did.IDStrings[1] {
		return UnknownChain, UnknownNetwork, id,
			fmt.Errorf("%w: network ID mismatch: found %v in ID but %v in DID",
				ErrInvalidDID, networkID, did.IDStrings[1])
	}

	return blockchain, networkID, id, nil
}

func IDFromDID(did DID) (ID, error) {
	_, _, id, err := Decompose(did)
	return id, err
}

func decodeIDFromDID(did DID) (ID, error) {
	var id ID

	if len(did.IDStrings) > 3 {
		return id, fmt.Errorf("%w: too many fields", ErrInvalidDID)
	}

	if len(did.IDStrings) < 1 {
		return id, fmt.Errorf("%w: no ID field in DID", ErrInvalidDID)
	}

	var err error
	id, err = IDFromString(did.IDStrings[len(did.IDStrings)-1])
	if err != nil {
		return id, fmt.Errorf("%w: %v", ErrInvalidDID, err)
	}

	if !CheckChecksum(id) {
		return id, fmt.Errorf("%w: invalid checksum", ErrInvalidDID)
	}

	return id, nil
}

// ParseDIDFromID returns DID from ID
func ParseDIDFromID(id ID) (*DID, error) {

	if !CheckChecksum(id) {
		return nil, fmt.Errorf("%w: invalid checksum", ErrInvalidDID)
	}

	method, blockchain, networkID, err := decodeDIDPartsFromID(id)
	if err != nil {
		return nil, err
	}

	didParts := []string{DIDSchema, string(method), string(blockchain)}
	if string(networkID) != "" {
		didParts = append(didParts, string(networkID))
	}

	didParts = append(didParts, id.String())

	didString := strings.Join(didParts, ":")

	did, err := Parse(didString)
	if err != nil {
		return nil, err
	}
	return did, nil
}

func decodeDIDPartsFromID(id ID) (DIDMethod, Blockchain, NetworkID, error) {
	methodByte := id.MethodByte()
	networkByte := id.BlockchainNetworkByte()

	method, err := FindDIDMethodByValue(methodByte)
	if err != nil {
		return "", UnknownChain, UnknownNetwork, err
	}

	blockchain, err := FindBlockchainForDIDMethodByValue(method, networkByte)
	if err != nil {
		return "", UnknownChain, UnknownNetwork, err
	}

	networkID, err := FindNetworkIDForDIDMethodByValue(method, networkByte)
	if err != nil {
		return "", UnknownChain, UnknownNetwork, err
	}

	return method, blockchain, networkID, nil
}
