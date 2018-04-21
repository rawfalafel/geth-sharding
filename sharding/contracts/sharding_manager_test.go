package contracts

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/sharding"
)

type SMCConfig struct {
	notaryLockupLenght   *big.Int
	proposerLockupLength *big.Int
}

var (
	mainKey, _                   = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	accountBalance1001Eth, _     = new(big.Int).SetString("1001000000000000000000", 10)
	notaryDepositInsufficient, _ = new(big.Int).SetString("999000000000000000000", 10)
	notaryDeposit, _             = new(big.Int).SetString("1000000000000000000000", 10)
	smcConfig                    = SMCConfig{
		notaryLockupLenght:   new(big.Int).SetInt64(1),
		proposerLockupLength: new(big.Int).SetInt64(1),
	}
)

func deploySMCContract(backend *backends.SimulatedBackend, key *ecdsa.PrivateKey) (common.Address, *types.Transaction, *SMC, error) {
	transactOpts := bind.NewKeyedTransactor(key)
	defer backend.Commit()
	return DeploySMC(transactOpts, backend)
}

// Test creating the SMC contract
func TestContractCreation(t *testing.T) {
	addr := crypto.PubkeyToAddress(mainKey.PublicKey)
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance1001Eth}})
	_, _, _, err := deploySMCContract(backend, mainKey)
	backend.Commit()
	if err != nil {
		t.Fatalf("can't deploy SMC: %v", err)
	}
}

// Test register notary
func TestNotaryRegister(t *testing.T) {
	const notaryCount = 3
	var notaryPoolAddr [notaryCount]common.Address
	var notaryPoolPrivKeys [notaryCount]*ecdsa.PrivateKey
	var txOpts [notaryCount]*bind.TransactOpts
	genesis := make(core.GenesisAlloc)

	for i := 0; i < notaryCount; i++ {
		key, _ := crypto.GenerateKey()
		notaryPoolPrivKeys[i] = key
		notaryPoolAddr[i] = crypto.PubkeyToAddress(key.PublicKey)
		txOpts[i] = bind.NewKeyedTransactor(key)
		txOpts[i].Value = notaryDeposit
		genesis[notaryPoolAddr[i]] = core.GenesisAccount{
			Balance: accountBalance1001Eth,
		}
	}

	backend := backends.NewSimulatedBackend(genesis)
	_, _, smc, _ := deploySMCContract(backend, notaryPoolPrivKeys[0])

	// Test notary 0 has not registered
	notary, err := smc.NotaryRegistry(&bind.CallOpts{}, notaryPoolAddr[0])
	if err != nil {
		t.Fatalf("Can't get notary registry info: %v", err)
	}

	if notary.Deposited != false {
		t.Fatalf("Notary has not registered. Got deposited flag: %v", notary.Deposited)
	}

	// Test notary 0 has registered
	if _, err := smc.RegisterNotary(txOpts[0]); err != nil {
		t.Fatalf("Registering notary has failed: %v", err)
	}
	backend.Commit()

	notary, err = smc.NotaryRegistry(&bind.CallOpts{}, notaryPoolAddr[0])

	if notary.Deposited != true &&
		notary.PoolIndex.Cmp(big.NewInt(0)) != 0 &&
		notary.DeregisteredPeriod.Cmp(big.NewInt(0)) != 0 {
		t.Fatalf("Incorrect notary registry. Want - deposited:true, index:0, period:0"+
			"Got - deposited:%v, index:%v, period:%v ", notary.Deposited, notary.PoolIndex, notary.DeregisteredPeriod)
	}

	// Test notary 1 and 2 have registered
	if _, err := smc.RegisterNotary(txOpts[1]); err != nil {
		t.Fatalf("Registering notary has failed: %v", err)
	}
	backend.Commit()

	if _, err := smc.RegisterNotary(txOpts[2]); err != nil {
		t.Fatalf("Registering notary has failed: %v", err)
	}
	backend.Commit()

	notary, err = smc.NotaryRegistry(&bind.CallOpts{}, notaryPoolAddr[1])

	if notary.Deposited != true &&
		notary.PoolIndex.Cmp(big.NewInt(1)) != 0 &&
		notary.DeregisteredPeriod.Cmp(big.NewInt(0)) != 0 {
		t.Fatalf("Incorrect notary registry. Want - deposited:true, index:1, period:0"+
			"Got - deposited:%v, index:%v, period:%v ", notary.Deposited, notary.PoolIndex, notary.DeregisteredPeriod)
	}

	notary, err = smc.NotaryRegistry(&bind.CallOpts{}, notaryPoolAddr[2])

	if notary.Deposited != true &&
		notary.PoolIndex.Cmp(big.NewInt(2)) != 0 &&
		notary.DeregisteredPeriod.Cmp(big.NewInt(0)) != 0 {
		t.Fatalf("Incorrect notary registry. Want - deposited:true, index:2, period:0"+
			"Got - deposited:%v, index:%v, period:%v ", notary.Deposited, notary.PoolIndex, notary.DeregisteredPeriod)
	}

	// Check total numbers of notaries in pool
	numNotaries, err := smc.NotaryPoolLength(&bind.CallOpts{})
	if err != nil {
		t.Fatalf("Failed to get notary pool length: %v", err)
	}
	if numNotaries.Cmp(big.NewInt(3)) != 0 {
		t.Fatalf("Incorrect count from notary pool. Want: 3, Got: %v", numNotaries)
	}
}

func TestNotaryRegisterWithoutEnoughEther(t *testing.T) {
	addr := crypto.PubkeyToAddress(mainKey.PublicKey)
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance1001Eth}})
	txOpts := bind.NewKeyedTransactor(mainKey)
	txOpts.Value = notaryDepositInsufficient
	_, _, smc, _ := deploySMCContract(backend, mainKey)

	smc.RegisterNotary(txOpts)
	backend.Commit()

	notary, _ := smc.NotaryRegistry(&bind.CallOpts{}, addr)
	numNotaries, _ := smc.NotaryPoolLength(&bind.CallOpts{})

	if notary.Deposited != false {
		t.Fatalf("Notary deposited with insufficient fund")
	}

	if numNotaries.Cmp(big.NewInt(0)) != 0 {
		t.Fatalf("Incorrect count from notary pool. Want: 0, Got: %v", numNotaries)
	}
}

func TestNotaryDoubleRegisters(t *testing.T) {
	addr := crypto.PubkeyToAddress(mainKey.PublicKey)
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance1001Eth}})
	txOpts := bind.NewKeyedTransactor(mainKey)
	txOpts.Value = notaryDeposit
	_, _, smc, _ := deploySMCContract(backend, mainKey)

	// Notary 0 registers
	smc.RegisterNotary(txOpts)
	backend.Commit()

	notary, _ := smc.NotaryRegistry(&bind.CallOpts{}, addr)
	numNotaries, _ := smc.NotaryPoolLength(&bind.CallOpts{})

	if notary.Deposited != true {
		t.Fatalf("Notary has not registered. Got deposited flag: %v", notary.Deposited)
	}

	if numNotaries.Cmp(big.NewInt(1)) != 0 {
		t.Fatalf("Incorrect count from notary pool. Want: 1, Got: %v", numNotaries)
	}

	// Notary 0 registers again
	_, _ = smc.RegisterNotary(txOpts)
	backend.Commit()

	if numNotaries.Cmp(big.NewInt(1)) != 0 {
		t.Fatalf("Incorrect count from notary pool. Want: 1, Got: %v", numNotaries)
	}

	//ctx := context.Background()
	//_, _ = backend.TransactionReceipt(ctx, tx.Hash())
	////t.Log(r.CumulativeGasUsed)

}

func TestNotaryDeregisters(t *testing.T) {
	addr := crypto.PubkeyToAddress(mainKey.PublicKey)
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance1001Eth}})
	txOpts := bind.NewKeyedTransactor(mainKey)
	txOpts.Value = notaryDeposit
	_, _, smc, _ := deploySMCContract(backend, mainKey)

	// Notary 0 registers
	smc.RegisterNotary(txOpts)
	backend.Commit()

	notary, _ := smc.NotaryRegistry(&bind.CallOpts{}, addr)
	numNotaries, _ := smc.NotaryPoolLength(&bind.CallOpts{})

	if notary.Deposited != true {
		t.Fatalf("Notary has not registered. Got deposited flag: %v", notary.Deposited)
	}

	if numNotaries.Cmp(big.NewInt(1)) != 0 {
		t.Fatalf("Incorrect count from notary pool. Want: 1, Got: %v", numNotaries)
	}

	// Fast forward 100 blocks
	for i := 0; i < 100; i++ {
		backend.Commit()
	}

	txOpts = bind.NewKeyedTransactor(mainKey)
	_, err := smc.DeregisterNotary(txOpts)
	if err != nil {
		t.Fatalf("Failed to deregister notary: %v", err)
	}
	backend.Commit()

	// Verify notary has saved deregister period as (current block number / period length)
	notary, _ = smc.NotaryRegistry(&bind.CallOpts{}, addr)
	currentPeriod := big.NewInt(int64(100 / sharding.PeriodLength))
	if currentPeriod.Cmp(notary.DeregisteredPeriod) != 0 {
		t.Fatalf("Incorrect notary degister period. Want: %v, Got: %v ", currentPeriod, notary.DeregisteredPeriod)
	}

	numNotaries, _ = smc.NotaryPoolLength(&bind.CallOpts{})
	if numNotaries.Cmp(big.NewInt(0)) != 0 {
		t.Fatalf("Incorrect count from notary pool. Want: 0, Got: %v", numNotaries)
	}
}
