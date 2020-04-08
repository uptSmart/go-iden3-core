package eth

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethkeystore "github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"

	// "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	log "github.com/sirupsen/logrus"
)

var (
	ErrAccountNil = fmt.Errorf("Authorized calls can't be made when the account is nil")
	// ErrReceiptStatusFailed when receiving a failed transaction
	ErrReceiptStatusFailed = fmt.Errorf("receipt status is failed")
	// ErrReceiptNotRecieved when unable to retrieve a transaction
	ErrReceiptNotRecieved = fmt.Errorf("receipt not available")
)

const (
	errStrDeploy      = "deployment of %s failed: %w"
	errStrWaitReceipt = "wait receipt of %s deploy failed: %w"
)

// Client is an ethereum client to call Smart Contract methods.
type Client struct {
	client         *ethclient.Client
	account        *accounts.Account
	ks             *ethkeystore.KeyStore
	ReceiptTimeout time.Duration
}

// NewClient creates a Client instance.  The account is not mandatory (it can
// be nil).  If the account is nil, CallAuth will fail with ErrAccountNil.
func NewClient(client *ethclient.Client, account *accounts.Account, ks *ethkeystore.KeyStore) *Client {
	return &Client{client: client, account: account, ks: ks, ReceiptTimeout: 60 * time.Second}
}

// BalanceAt retieves information about the default account
func (c *Client) BalanceAt(addr common.Address) (*big.Int, error) {
	return c.client.BalanceAt(context.TODO(), addr, nil)
}

// Account returns the underlying ethereum account
func (c *Client) Account() *accounts.Account {
	return c.account
}

// CallAuth performs a Smart Contract method call that requires authorization.
// This call requires a valid account with Ether that can be spend during the
// call.
func (c *Client) CallAuth(gasLimit uint64,
	fn func(*ethclient.Client, *bind.TransactOpts) (*types.Transaction, error)) (*types.Transaction, error) {
	if c.account == nil {
		return nil, ErrAccountNil
	}

	gasPrice, err := c.client.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, err
	}
	inc := new(big.Int).Set(gasPrice)
	inc.Div(inc, new(big.Int).SetUint64(100))
	gasPrice.Add(gasPrice, inc)
	log.WithField("gasPrice", gasPrice).Debug("Transaction metadata")

	auth, err := bind.NewKeyStoreTransactor(c.ks, *c.account)
	if err != nil {
		return nil, err
	}
	auth.Value = big.NewInt(0) // in wei
	if gasLimit == 0 {
		auth.GasLimit = uint64(300000) // in units
	} else {
		auth.GasLimit = gasLimit // in units
	}
	auth.GasPrice = gasPrice

	tx, err := fn(c.client, auth)
	if tx != nil {
		log.WithField("tx", tx.Hash().Hex()).WithField("nonce", tx.Nonce()).Debug("Transaction")
	}
	return tx, err
}

type ContractData struct {
	Address common.Address
	Tx      *types.Transaction
	Receipt *types.Receipt
}

// Deploy a smart contract.  `name` is used to log deployment information.  fn
// is a wrapper to the deploy function generated by abigen.  In case of error,
// the returned `ContractData` may have some parameters filled depending on the
// kind of error that ocurred.
// successfull.
func (c *Client) Deploy(name string,
	fn func(c *ethclient.Client, auth *bind.TransactOpts) (common.Address, *types.Transaction, interface{}, error)) (ContractData, error) {
	var contractData ContractData
	log.WithField("contract", name).Infof("Deploying")
	tx, err := c.CallAuth(
		1000000,
		func(client *ethclient.Client, auth *bind.TransactOpts) (*types.Transaction, error) {
			addr, tx, _, err := fn(client, auth)
			if err != nil {
				return nil, err
			}
			contractData.Address = addr
			return tx, nil
		},
	)
	if err != nil {
		return contractData, fmt.Errorf(errStrDeploy, name, err)
	}
	log.WithField("tx", tx.Hash().Hex()).WithField("contract", name).Infof("Waiting receipt")
	contractData.Tx = tx
	receipt, err := c.WaitReceipt(tx)
	if err != nil {
		return contractData, fmt.Errorf(errStrWaitReceipt, name, err)
	}
	contractData.Receipt = receipt
	return contractData, nil
}

// Call performs a read only Smart Contract method call.
func (c *Client) Call(fn func(*ethclient.Client) error) error {
	return fn(c.client)
}

// WaitReceipt will block until a transaction is confirmed.  Internally it
// polls the state every 200 milliseconds.
func (c *Client) WaitReceipt(tx *types.Transaction) (*types.Receipt, error) {
	var err error
	var receipt *types.Receipt

	txid := tx.Hash()
	log.WithField("tx", txid.Hex()).Debug("Waiting for receipt")

	start := time.Now()
	for receipt == nil && time.Since(start) < c.ReceiptTimeout {
		receipt, err = c.client.TransactionReceipt(context.TODO(), txid)
		if receipt == nil {
			time.Sleep(200 * time.Millisecond)
		}
	}

	if receipt != nil && receipt.Status == types.ReceiptStatusFailed {
		log.WithField("tx", txid.Hex()).Error("WEB3 Failed transaction receipt")
		return receipt, ErrReceiptStatusFailed
	}

	if receipt == nil {
		log.WithField("tx", txid.Hex()).Error("WEB3 Failed transaction")
		return receipt, ErrReceiptNotRecieved
	}
	log.WithField("tx", txid.Hex()).Debug("WEB3 Success transaction")

	return receipt, err
}
