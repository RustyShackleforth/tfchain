package wallet

import (
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/threefoldfoundation/tfchain/cmd/tfchaint/explorer"
	"github.com/threefoldtech/rivine/crypto"
	"github.com/threefoldtech/rivine/modules"
	"github.com/threefoldtech/rivine/types"
)

type (
	// Wallet represents a seed, and some derived info used to spend the associated funds
	Wallet struct {
		// seed is the seed of the wallet
		seed modules.Seed
		// keys are all generated addresses and the spendableKey's used to spend them
		keys map[types.UnlockHash]spendableKey
		// firstAddress is the first address generated from the seed, which is the default refund address
		firstAddress types.UnlockHash
		// backend used to interact with the chain
		backend Backend

		// name is the name of the wallet
		name string
	}

	// SpendableOutputs maps CoinOutputID's to their corresponding actual output
	SpendableOutputs map[types.CoinOutputID]types.CoinOutput

	// spendableKey is the required information to spend an input associated with a key
	spendableKey struct {
		PublicKey crypto.PublicKey
		SecretKey crypto.SecretKey
	}

	// jobContext stores information used to fetch an address on the explorer
	jobContext struct {
		currentChainHeight types.BlockHeight
		maturityDelay      types.BlockHeight
		address            types.UnlockHash
	}
)

const (
	// ArbitraryDataMaxSize is the maximum size of the arbitrary data field on a transaction
	ArbitraryDataMaxSize = 83

	// WorkerCount is the number of workers used to process information about explorer addresses
	WorkerCount = 25
)

var (
	// ErrWalletExists indicates that a wallet with that name allready exists when trying to create a new wallet
	ErrWalletExists = errors.New("A wallet with that name already exists")
	// ErrNoSuchWallet indicates that there is no wallet for a given name when trying to load a wallet
	ErrNoSuchWallet = errors.New("A wallet with that name does not exist")
	// ErrTooMuchData indicates that the there is too much data to add to the transction
	ErrTooMuchData = errors.New("Too much data is being supplied to the transaction")
	// ErrInsufficientWalletFunds indicates that the wallet does not have sufficient funds to fund the transaction
	ErrInsufficientWalletFunds = errors.New("Insufficient funds to create this transaction")
)

// New creates a new wallet with a random seed
func New(name string, keysToLoad uint64, backendName string) (*Wallet, error) {
	seed := modules.Seed{}
	_, err := rand.Read(seed[:])
	if err != nil {
		return nil, err
	}

	return NewWalletFromSeed(name, seed, keysToLoad, backendName)
}

// NewWalletFromMnemonic creates a new wallet from a given mnemonic
func NewWalletFromMnemonic(name string, mnemonic string, keysToLoad uint64, backendName string) (*Wallet, error) {
	seed, err := modules.InitialSeedFromMnemonic(mnemonic)
	if err != nil {
		return nil, err
	}
	return NewWalletFromSeed(name, seed, keysToLoad, backendName)
}

// NewWalletFromSeed creates a new wallet with a given seed
func NewWalletFromSeed(name string, seed modules.Seed, keysToLoad uint64, backendName string) (*Wallet, error) {
	exists, err := walletExists(name)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrWalletExists
	}

	backend := loadBackend(backendName)

	w := &Wallet{
		seed:    seed,
		name:    name,
		backend: backend,
	}

	w.generateKeys(keysToLoad)

	if err = save(w); err != nil {
		return nil, err
	}

	return w, nil
}

// LoadBackend loads a backend with the given name
func loadBackend(name string) Backend {
	switch name {
	case "standard":
		return explorer.NewMainnetGroupedExplorer()
	case "testnet":
		return explorer.NewTestnetGroupedExplorer()
	case "devnet":
		return explorer.NewDevnetGroupedExplorer()
	default:
		// for now anything else will also default to devnet
		return explorer.NewDevnetGroupedExplorer()
	}
}

// Load loads persistent data for a wallet with a given name, and restores the wallets state
func Load(name string) (*Wallet, error) {
	data, err := load(name)
	if err != nil {
		return nil, err
	}
	w := &Wallet{
		name:    name,
		seed:    data.Seed,
		backend: loadBackend(data.Backend),
	}

	w.generateKeys(data.KeysToLoad)

	return w, nil
}

// GetChainConstants returns the chainconstatns of the underlying network
func (w *Wallet) GetChainConstants() (modules.DaemonConstants, error) {
	return w.backend.GetChainConstants()
}

// GetBalance returns the current unlocked and locked balance for the wallet
func (w *Wallet) GetBalance() (types.Currency, types.Currency, error) {
	outputs, err := w.getUnspentCoinOutputs()
	if err != nil {
		return types.Currency{}, types.Currency{}, err
	}

	unlocked, locked, err := w.splitTimeLockedOutputs(outputs)
	return w.getBalance(unlocked), w.getBalance(locked), err
}

func (w *Wallet) getBalance(outputs SpendableOutputs) types.Currency {
	balance := types.NewCurrency64(0)
	for _, uco := range outputs {
		balance = balance.Add(uco.Value)
	}
	return balance
}

// TransferCoins transfers coins by creating and submitting a V1 transaction.
// Data can optionally be included.
func (w *Wallet) TransferCoins(amount types.Currency, to types.UnlockConditionProxy, data []byte, newRefundAddress bool) (types.TransactionID, error) {
	return w.TransferCoinsMulti([]types.Currency{amount}, []types.UnlockConditionProxy{to}, data, newRefundAddress)
}

// TransferCoinsMulti transfers coins by creating and submitting a V1 transaction,
// with multiple outputs. Data can optionally be included.
func (w *Wallet) TransferCoinsMulti(amounts []types.Currency, conditions []types.UnlockConditionProxy, data []byte, newRefundAddress bool) (types.TransactionID, error) {
	// check data length
	if len(data) > ArbitraryDataMaxSize {
		return types.TransactionID{}, ErrTooMuchData
	}
	if len(amounts) == 0 {
		return types.TransactionID{}, errors.New("at least one amount is required")
	}
	if len(amounts) != len(conditions) {
		return types.TransactionID{}, errors.New("the amount of of amounts does not match the amount of conditions")
	}

	chainCts, err := w.backend.GetChainConstants()
	if err != nil {
		return types.TransactionID{}, err
	}

	outputs, err := w.getUnspentCoinOutputs()
	if err != nil {
		return types.TransactionID{}, err
	}

	// only continue with unlocked outputs
	outputs, _, err = w.splitTimeLockedOutputs(outputs)
	if err != nil {
		return types.TransactionID{}, err
	}

	walletBalance := w.getBalance(outputs)

	// we give only the minimum fee
	txFee := chainCts.MinimumTransactionFee

	// Since this is only for demonstration purposes, lets give a fixed 10 hastings fee
	// minerfee := types.NewCurrency64(10)

	// The total funds we will be spending in this transaction
	requiredFunds := (types.Currency{}).Add(txFee)
	fmt.Printf("fee: %s\n", requiredFunds.String())
	for i := range amounts {
		requiredFunds = requiredFunds.Add(amounts[i])
	}
	fmt.Printf("available funds: %s\n", walletBalance.String())
	fmt.Printf("required funds (with %d outputs): %s\n", len(amounts), requiredFunds.String())

	// Verify that we actually have enough funds available in the wallet to complete the transaction
	if walletBalance.Cmp(requiredFunds) == -1 {
		return types.TransactionID{}, ErrInsufficientWalletFunds
	}

	// Create the transaction object
	var txn types.Transaction
	txn.Version = chainCts.DefaultTransactionVersion

	// Greedily add coin inputs until we have enough to fund the output and minerfee
	inputs := []types.CoinInput{}

	// Track the amount of coins we already added via the inputs
	inputValue := types.ZeroCurrency

	for id, utxo := range outputs {
		// If the inputValue is not smaller than the requiredFunds we added enough inputs to fund the transaction
		if inputValue.Cmp(requiredFunds) != -1 {
			break
		}
		// Append the input
		inputs = append(inputs, types.CoinInput{
			ParentID: id,
			Fulfillment: types.NewFulfillment(types.NewSingleSignatureFulfillment(
				types.Ed25519PublicKey(w.keys[utxo.Condition.UnlockHash()].PublicKey))),
		})
		// And update the value in the transaction
		inputValue = inputValue.Add(utxo.Value)
	}
	// Set the inputs
	txn.CoinInputs = inputs

	// sanity checking
	for _, inp := range inputs {
		if _, exists := w.keys[outputs[inp.ParentID].Condition.UnlockHash()]; !exists {
			return types.TransactionID{}, errors.New("Trying to spend unexisting output")
		}
	}

	// Add our first output
	for i, condition := range conditions {
		amount := amounts[i]
		txn.CoinOutputs = append(txn.CoinOutputs, types.CoinOutput{
			Value:     amount,
			Condition: condition,
		})
	}

	// So now we have enough inputs to fund everything. But we might have overshot it a little bit, so lets check that
	// and add a new output to ourself if required to consume the leftover value
	remainder := inputValue.Sub(requiredFunds)
	if !remainder.IsZero() {
		var refundAddr types.UnlockHash
		// We have leftover funds, so add a new output
		if !newRefundAddress {
			refundAddr = w.firstAddress
		} else {
			// generate a new address
			key, err := generateSpendableKey(w.seed, uint64(len(w.keys)))
			if err != nil {
				return types.TransactionID{}, err
			}
			refundAddr, err = key.UnlockHash()
			if err != nil {
				return types.TransactionID{}, err
			}
			w.keys[refundAddr] = key
			// make sure to save so we update the key count in the persistent data
			if err = save(w); err != nil {
				return types.TransactionID{}, err
			}
		}
		outputToSelf := types.CoinOutput{
			Value:     remainder,
			Condition: types.NewCondition(types.NewUnlockHashCondition(refundAddr)),
		}
		// add our self referencing output to the transaction
		txn.CoinOutputs = append(txn.CoinOutputs, outputToSelf)
	}

	// Add the miner fee to the transaction
	txn.MinerFees = []types.Currency{txFee}

	// Make sure to set the data
	txn.ArbitraryData = data

	// sign transaction
	if err := w.signTxn(txn, outputs); err != nil {
		return types.TransactionID{}, err
	}

	// finally commit
	return w.backend.SendTxn(txn)
}

// ListAddresses returns all currently loaded addresses
func (w *Wallet) ListAddresses() []types.UnlockHash {
	var addresses []types.UnlockHash
	for key := range w.keys {
		addresses = append(addresses, key)
	}
	return addresses
}

// LoadKeys loads `amount` additional keys in the wallet and saves the wallet state
func (w *Wallet) LoadKeys(amount uint64) error {
	currentKeys := len(w.keys)
	w.generateKeys(uint64(currentKeys) + amount)
	return save(w)
}

func (w *Wallet) getUnspentCoinOutputs() (SpendableOutputs, error) {
	workerCount := WorkerCount

	if len(w.keys) < workerCount {
		workerCount = len(w.keys)
	}

	currentChainHeight, err := w.backend.CurrentHeight()
	if err != nil {
		return nil, err
	}

	chainCts, err := w.backend.GetChainConstants()
	if err != nil {
		return nil, err
	}

	jobs := make(chan jobContext, len(w.keys))
	results := make(chan SpendableOutputs)
	errChan := make(chan error, len(w.keys))

	for worker := 1; worker <= workerCount; worker++ {
		go w.checkAddress(jobs, results, errChan)
	}

	for address := range w.keys {
		jobs <- jobContext{
			address:            address,
			currentChainHeight: currentChainHeight,
			maturityDelay:      chainCts.MaturityDelay,
		}
	}
	close(jobs)

	ucos := make(SpendableOutputs)
	errs := make([]string, 0)

	for i := 0; i < len(w.keys); i++ {
		select {
		case res := <-results:
			for k, v := range res {
				ucos[k] = v
			}
		case err := <-errChan:
			errs = append(errs, err.Error())
		}
	}

	close(results)
	close(errChan)

	if len(errs) > 0 {
		err = errors.New(strings.Join(errs, "\n"))
	}

	return ucos, err
}

func (w *Wallet) checkAddress(jobs <-chan jobContext, results chan<- SpendableOutputs, errChan chan<- error) {
	for context := range jobs {
		blocks, transactions, err := w.backend.CheckAddress(context.address)
		if err != nil {
			errChan <- err
			continue
		}
		tempMap := make(SpendableOutputs)

		// We scann the blocks here for the miner fees, and the transactions for actual transactions
		for _, block := range blocks {
			// Collect the miner fees
			// But only those that have matured already
			if block.Height+context.maturityDelay >= context.currentChainHeight {
				// ignore miner payout which hasn't yet matured
				continue
			}
			for i, minerPayout := range block.RawBlock.MinerPayouts {
				if minerPayout.UnlockHash == context.address {
					tempMap[block.MinerPayoutIDs[i]] = types.CoinOutput{
						Value: minerPayout.Value,
						Condition: types.UnlockConditionProxy{
							Condition: types.NewUnlockHashCondition(minerPayout.UnlockHash),
						},
					}
				}
			}
		}

		// Collect the transaction outputs
		for _, txn := range transactions {
			for i, utxo := range txn.RawTransaction.CoinOutputs {
				if utxo.Condition.UnlockHash() == context.address {
					tempMap[txn.CoinOutputIDs[i]] = utxo
				}
			}
		}
		// Remove the ones we've spent already
		for _, txn := range transactions {
			for _, ci := range txn.RawTransaction.CoinInputs {
				delete(tempMap, ci.ParentID)
			}
		}

		results <- tempMap
	}
}

// splitTimeLockedOutputs separates a list of SpendableOutputs into a list of outputs which can be spent right now (no timelock or
// timelock has passed), and outputs which are still timelocked
func (w *Wallet) splitTimeLockedOutputs(outputs SpendableOutputs) (SpendableOutputs, SpendableOutputs, error) {
	unlocked := make(SpendableOutputs)
	locked := make(SpendableOutputs)

	ctx, err := w.getFulfillableContextForLatestBlock()
	if err != nil {
		return unlocked, locked, err
	}

	// sort the outputs
	for id, co := range outputs {
		if co.Condition.Fulfillable(ctx) {
			unlocked[id] = co
		} else {
			locked[id] = co
		}
	}

	return unlocked, locked, nil
}

// generateKeys clears all existing keys and generates up to amount keys. If amount <= len(w.keys), no new keys will be generated
func (w *Wallet) generateKeys(amount uint64) error {
	w.keys = make(map[types.UnlockHash]spendableKey)

	for i := 0; i < int(amount); i++ {
		key, err := generateSpendableKey(w.seed, uint64(i))
		if err != nil {
			return err
		}
		uh, err := key.UnlockHash()
		if err != nil {
			return err
		}
		w.keys[uh] = key
		if i == 0 {
			w.firstAddress = uh
		}
	}

	return nil
}

// signTxn signs a transaction
func (w *Wallet) signTxn(txn types.Transaction, usedOutputIDs SpendableOutputs) error {
	// sign every coin input
	for idx, input := range txn.CoinInputs {
		// coinOutput has been checked during creation time, in the parent function,
		// hence we no longer need to check it here
		key := w.keys[usedOutputIDs[input.ParentID].Condition.UnlockHash()]
		err := input.Fulfillment.Sign(types.FulfillmentSignContext{
			ExtraObjects: []interface{}{uint64(idx)},
			Transaction:  txn,
			Key:          key.SecretKey,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// Mnemonic returns the human readable form of the seed
func (w *Wallet) Mnemonic() (string, error) {
	return modules.NewMnemonic(w.seed)
}

func (w *Wallet) getFulfillableContextForLatestBlock() (types.FulfillableContext, error) {
	height, err := w.backend.CurrentHeight()
	if err != nil {
		return types.FulfillableContext{}, nil
	}
	timestamp := time.Now()
	return types.FulfillableContext{
		BlockHeight: height,
		BlockTime:   types.Timestamp(uint64(timestamp.Unix())),
	}, nil
}

func generateSpendableKey(seed modules.Seed, index uint64) (spendableKey, error) {
	// Generate the keys and unlock conditions.
	entropy, err := crypto.HashAll(seed, index)
	if err != nil {
		return spendableKey{}, err
	}
	sk, pk := crypto.GenerateKeyPairDeterministic(entropy)
	return spendableKey{
		PublicKey: pk,
		SecretKey: sk,
	}, nil
}

// UnlockHash derives the unlockhash from the spendableKey
func (sk spendableKey) UnlockHash() (types.UnlockHash, error) {
	return types.NewEd25519PubKeyUnlockHash(sk.PublicKey)
}
