// Copyright (c) 2014-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package waddrmgr

import (
	"crypto/rand"
	"crypto/sha512"
	"fmt"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/crypto/bip32"
	ecc "github.com/Qitmeer/qng/crypto/ecc/secp256k1"
	chaincfg "github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/services/hotwallet/internal/zero"
	"github.com/Qitmeer/qng/services/hotwallet/snacl"
	"github.com/Qitmeer/qng/services/hotwallet/walletdb"
	"sync"
	"time"
)

const (
	// MaxAccountNum is the maximum allowed account number.  This value was
	// chosen because accounts are hardened children and therefore must not
	// exceed the hardened child range of extended keys and it provides a
	// reserved account at the top of the range for supporting imported
	// addresses.
	MaxAccountNum = HardenedKeyStart - 2 // 2^31 - 2
	// MaxAddressesPerAccount is the maximum allowed number of addresses
	// per account number.  This value is based on the limitation of the
	// underlying hierarchical deterministic key derivation.
	MaxAddressesPerAccount = HardenedKeyStart - 1
	// ImportedAddrAccount is the account number to use for all imported
	// addresses.  This is useful since normal accounts are derived from
	// the root hierarchical deterministic key and imported addresses do
	// not fit into that model.
	ImportedAddrAccount = MaxAccountNum + 1 // 2^31 - 1
	// ImportedAddrAccountName is the name of the imported account.
	ImportedAddrAccountName = "imported"

	// HardenedKeyStart is the index at which a hardended key starts.  Each
	// extended key has 2^31 normal child keys and 2^31 hardned child keys.
	// Thus the range for normal child keys is [0, 2^31 - 1] and the range
	// for hardened child keys is [2^31, 2^32 - 1].
	HardenedKeyStart = 0x80000000 // 2^31

	// DefaultAccountNum is the number of the default account.
	DefaultAccountNum = 0

	// Account combination payment mark
	AccountMergePayNum = -1

	// defaultAccountName is the initial name of the default account.  Note
	// that the default account may be renamed and is not a reserved name,
	// so the default account might not be named "default" and non-default
	// accounts may be named "default".
	//
	// Account numbers never change, so the DefaultAccountNum should be
	// used to refer to (and only to) the default account.
	defaultAccountName = "default"

	// The hierarchy described by BIP0043 is:
	//  m/<purpose>'/*
	// This is further extended by BIP0044 to:
	//  m/44'/<coin type>'/<account>'/<branch>/<address index>
	//
	// The branch is 0 for external addresses and 1 for internal addresses.

	// maxCoinType is the maximum allowed coin type used when structuring
	// the BIP0044 multi-account hierarchy.  This value is based on the
	// limitation of the underlying hierarchical deterministic key
	// derivation.

	// ExternalBranch is the child number to use when performing BIP0044
	// style hierarchical deterministic key derivation for the external
	// branch.
	ExternalBranch uint32 = 0

	// InternalBranch is the child number to use when performing BIP0044
	// style hierarchical deterministic key derivation for the internal
	// branch.
	InternalBranch uint32 = 1

	// saltSize is the number of bytes of the salt used when hashing
	// private passphrases.
	saltSize = 32
)

// isReservedAccountName returns true if the account name is reserved.
// Reserved accounts may never be renamed, and other accounts may not be
// renamed to a reserved name.
func isReservedAccountName(name string) bool {
	return name == ImportedAddrAccountName
}

// ScryptOptions is used to hold the scrypt parameters needed when deriving new
// passphrase keys.
type ScryptOptions struct {
	N, R, P int
}

// AccountProperties contains properties associated with each account, such as
// the account name, number, and the nubmer of derived and imported keys.
type AccountProperties struct {
	AccountNumber    uint32
	AccountName      string
	ExternalKeyCount uint32
	InternalKeyCount uint32
	ImportedKeyCount uint32
}

// unlockDeriveInfo houses the information needed to derive a private key for a
// managed address when the address manager is unlocked.  See the
// deriveOnUnlock field in the Manager struct for more details on how this is
// used.
type unlockDeriveInfo struct {
	managedAddr ManagedAddress
	branch      uint32
	index       uint32
}

// OpenCallbacks houses caller-provided callbacks that may be called when
// opening an existing manager.  The open blocks on the execution of these
// functions.
type OpenCallbacks struct {
	// ObtainSeed is a callback function that is potentially invoked during
	// upgrades.  It is intended to be used to request the wallet seed
	// from the user (or any other mechanism the caller deems fit).
	ObtainSeed ObtainUserInputFunc

	// ObtainPrivatePass is a callback function that is potentially invoked
	// during upgrades.  It is intended to be used to request the wallet
	// private passphrase from the user (or any other mechanism the caller
	// deems fit).
	ObtainPrivatePass ObtainUserInputFunc
}

type accountInfo struct {
	acctName string

	// The account key is used to derive the branches which in turn derive
	// the internal and external addresses.  The accountKeyPriv will be nil
	// when the address manager is locked.
	acctKeyEncrypted []byte
	acctKeyPriv      *bip32.Key
	acctKeyPub       *bip32.Key

	// The external branch is used for all addresses which are intended for
	// external use.
	nextExternalIndex uint32
	lastExternalAddr  ManagedAddress

	// The internal branch is used for all adddresses which are only
	// intended for internal wallet use such as change addresses.
	nextInternalIndex uint32
	lastInternalAddr  ManagedAddress
}

// DefaultScryptOptions is the default options used with scrypt.
var DefaultScryptOptions = ScryptOptions{
	N: 262144, // 2^18
	R: 8,
	P: 1,
}

// addrKey is used to uniquely identify an address even when those addresses
// would end up being the same bitcoin address (as is the case for
// pay-to-pubkey and pay-to-pubkey-hash style of addresses).
type addrKey string

// accountInfo houses the current state of the internal and external branches
// of an account along with the extended keys needed to derive new keys.  It
// also handles locking by keeping an encrypted version of the serialized
// private extended key so the unencrypted versions can be cleared from memory
// when the address manager is locked.

// unlockDeriveInfo houses the information needed to derive a private key for a
// managed address when the address manager is unlocked.  See the
// deriveOnUnlock field in the Manager struct for more details on how this is
// used.

// SecretKeyGenerator is the function signature of a method that can generate
// secret keys for the address manager.

// EncryptorDecryptor provides an abstraction on top of snacl.CryptoKey so that
// our tests can use dependency injection to force the behaviour they need.
type EncryptorDecryptor interface {
	Encrypt(in []byte) ([]byte, error)
	Decrypt(in []byte) ([]byte, error)
	Bytes() []byte
	CopyBytes([]byte)
	Zero()
}

// CryptoKeyType is used to differentiate between different kinds of
// crypto keys.
type CryptoKeyType byte

// newCryptoKey is used as a way to replace the new crypto key generation
// function used so tests can provide a version that fails for testing error
// paths.

var newCryptoKey = defaultNewCryptoKey

// defaultNewCryptoKey returns a new CryptoKey.  See newCryptoKey.
func defaultNewCryptoKey() (EncryptorDecryptor, error) {
	key, err := snacl.GenerateCryptoKey()
	if err != nil {
		return nil, err
	}
	return &cryptoKey{*key}, nil
}

// Manager represents a concurrency safe crypto currency address manager and
// key store.
type Manager struct {
	mtx sync.RWMutex
	// scopedManager is a mapping of scope of scoped manager, the manager
	// itself loaded into memory.
	scopedManagers map[KeyScope]*ScopedKeyManager

	externalAddrSchemas map[AddressType][]KeyScope
	internalAddrSchemas map[AddressType][]KeyScope
	syncState           syncState
	chainHeight         uint32
	watchingOnly        bool
	birthday            time.Time
	locked              bool
	closed              bool
	chainParams         *chaincfg.Params

	// masterKeyPub is the secret key used to secure the cryptoKeyPub key
	// and masterKeyPriv is the secret key used to secure the cryptoKeyPriv
	// key.  This approach is used because it makes changing the passwords
	// much simpler as it then becomes just changing these keys.  It also
	// provides future flexibility.
	//
	// NOTE: This is not the same thing as BIP0032 master node extended
	// key.
	//
	// The underlying master private key will be zeroed when the address
	// manager is locked.
	masterKeyPub  *snacl.SecretKey
	masterKeyPriv *snacl.SecretKey

	// cryptoKeyPub is the key used to encrypt public extended keys and
	// addresses.
	cryptoKeyPub EncryptorDecryptor

	// cryptoKeyPriv is the key used to encrypt private data such as the
	// master hierarchical deterministic extended key.
	//
	// This key will be zeroed when the address manager is locked.
	cryptoKeyPrivEncrypted []byte
	cryptoKeyPriv          EncryptorDecryptor

	// cryptoKeyScript is the key used to encrypt script data.
	//
	// This key will be zeroed when the address manager is locked.
	cryptoKeyScriptEncrypted []byte
	cryptoKeyScript          EncryptorDecryptor

	// privPassphraseSalt and hashedPrivPassphrase allow for the secure
	// detection of a correct passphrase on manager unlock when the
	// manager is already unlocked.  The hash is zeroed each lock.
	privPassphraseSalt   [saltSize]byte
	hashedPrivPassphrase [sha512.Size]byte
}

// cryptoKey extends snacl.CryptoKey to implement EncryptorDecryptor.
type cryptoKey struct {
	snacl.CryptoKey
}

// Bytes returns a copy of this crypto key's byte slice.
func (ck *cryptoKey) Bytes() []byte {
	return ck.CryptoKey[:]
}

// CopyBytes copies the bytes from the given slice into this CryptoKey.
func (ck *cryptoKey) CopyBytes(from []byte) {
	copy(ck.CryptoKey[:], from)
}

// WatchOnly returns true if the root manager is in watch only mode, and false
// otherwise.
func (m *Manager) WatchOnly() bool {
	m.mtx.RLock()
	defer m.mtx.RUnlock()

	return m.watchOnly()
}

// watchOnly returns true if the root manager is in watch only mode, and false
// otherwise.
//
// NOTE: This method requires the Manager's lock to be held.
func (m *Manager) watchOnly() bool {
	return m.watchingOnly
}

// Lock performs a best try effort to remove and zero all secret keys associated
// with the address manager.
//
// This function will return an error if invoked on a watching-only address
// manager.
func (m *Manager) Lock() error {
	// A watching-only address manager can't be locked.
	if m.watchingOnly {
		return managerError(ErrWatchingOnly, errWatchingOnly, nil)
	}

	m.mtx.Lock()
	defer m.mtx.Unlock()

	// Error on attempt to lock an already locked manager.
	if m.locked {
		return managerError(ErrLocked, errLocked, nil)
	}

	m.lock()
	return nil
}

// lock performs a best try effort to remove and zero all secret keys associated
// with the address manager.
//
// This function MUST be called with the manager lock held for writes.
func (m *Manager) lock() {

	// Remove clear text private master and crypto keys from memory.
	m.cryptoKeyScript.Zero()
	m.cryptoKeyPriv.Zero()

	// NOTE: m.cryptoKeyPub is intentionally not cleared here as the address
	// manager needs to be able to continue to read and decrypt public data
	// which uses a separate derived key from the database even when it is
	// locked.

	m.locked = true
}

// Close cleanly shuts down the manager.  It makes a best try effort to remove
// and zero all private key and sensitive public key material associated with
// the address manager from memory.
func (m *Manager) Close() {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	if m.closed {
		return
	}

	// Attempt to clear private key material from memory.
	if !m.watchingOnly && !m.locked {
		m.lock()
	}

	// Remove clear text public master and crypto keys from memory.
	//m.cryptoKeyPub.Zero()

	m.closed = true
	return
}

// NewScopedKeyManager creates a newgo bu scoped key manager from the root manager. A
// scoped key manager is a sub-manager that only has the coin type key of a
// particular coin type and BIP0043 purpose. This is useful as it enables
// callers to create an arbitrary BIP0043 like schema with a stand alone
// manager. Note that a new scoped manager cannot be created if: the wallet is
// watch only, the manager hasn't been unlocked, or the root key has been.
// neutered from the database.
//
// TODO(roasbeef): addrtype of raw key means it'll look in scripts to possibly
// mark as gucci?
func Create(ns walletdb.ReadWriteBucket, seed, pubPassphrase, privPassphrase []byte,
	chainParams *chaincfg.Params, config *ScryptOptions,
	birthday time.Time) error {

	// Return an error if the manager has already been created in
	// the given database namespace.
	exists := managerExists(ns)
	if exists {
		return managerError(ErrAlreadyExists, errAlreadyExists, nil)
	}

	// Ensure the private passphrase is not empty.
	if len(privPassphrase) == 0 {
		str := "private passphrase may not be empty"
		return managerError(ErrEmptyPassphrase, str, nil)
	}

	// Perform the initial bucket creation and database namespace setup.
	if err := CreateManagerNS(ns, ScopeAddrMap); err != nil {
		return maybeConvertDbError(err)
	}

	if config == nil {
		config = &DefaultScryptOptions
	}

	// Generate new master keys.  These master keys are used to protect the
	// crypto keys that will be generated next.
	masterKeyPub, err := newSecretKey(&pubPassphrase, config)
	if err != nil {
		str := "failed to master public key"
		return managerError(ErrCrypto, str, err)
	}
	masterKeyPriv, err := newSecretKey(&privPassphrase, config)
	if err != nil {
		str := "failed to master private key"
		return managerError(ErrCrypto, str, err)
	}
	defer masterKeyPriv.Zero()

	// Generate the private passphrase salt.  This is used when hashing
	// passwords to detect whether an unlock can be avoided when the manager
	// is already unlocked.
	var privPassphraseSalt [saltSize]byte
	_, err = rand.Read(privPassphraseSalt[:])
	if err != nil {
		str := "failed to read random source for passphrase salt"
		return managerError(ErrCrypto, str, err)
	}

	// Generate new crypto public, private, and script keys.  These keys are
	// used to protect the actual public and private data such as addresses,
	// extended keys, and scripts.
	cryptoKeyPub, err := newCryptoKey()
	if err != nil {
		str := "failed to generate crypto public key"
		return managerError(ErrCrypto, str, err)
	}
	cryptoKeyPriv, err := newCryptoKey()
	if err != nil {
		str := "failed to generate crypto private key"
		return managerError(ErrCrypto, str, err)
	}
	defer cryptoKeyPriv.Zero()
	cryptoKeyScript, err := newCryptoKey()
	if err != nil {
		str := "failed to generate crypto script key"
		return managerError(ErrCrypto, str, err)
	}
	defer cryptoKeyScript.Zero()

	// Encrypt the crypto keys with the associated master keys.
	cryptoKeyPubEnc, err := masterKeyPub.Encrypt(cryptoKeyPub.Bytes())
	if err != nil {
		str := "failed to encrypt crypto public key"
		return managerError(ErrCrypto, str, err)
	}
	cryptoKeyPrivEnc, err := masterKeyPriv.Encrypt(cryptoKeyPriv.Bytes())
	if err != nil {
		str := "failed to encrypt crypto private key"
		return managerError(ErrCrypto, str, err)
	}
	cryptoKeyScriptEnc, err := masterKeyPriv.Encrypt(cryptoKeyScript.Bytes())
	if err != nil {
		str := "failed to encrypt crypto script key"
		return managerError(ErrCrypto, str, err)
	}
	createdAt := &BlockStamp{Hash: *chainParams.GenesisHash, Order: 0}
	// Create the initial sync state.
	syncInfo := newSyncState(createdAt, createdAt)

	// Save the master key params to the database.
	pubParams := masterKeyPub.Marshal()
	privParams := masterKeyPriv.Marshal()
	err = putMasterKeyParams(ns, pubParams, privParams)
	if err != nil {
		return maybeConvertDbError(err)
	}
	// Derive the master extended key from the seed.
	rootKey, err := bip32.NewMasterKey(seed)
	if err != nil {
		str := "failed to derive master extended key"
		return managerError(ErrKeyChain, str, err)
	}
	rootPubKey := rootKey.PublicKey()

	// Next, for each registers default manager scope, we'll create the
	// hardened cointype key for it, as well as the first default account.
	for _, defaultScope := range DefaultKeyScopes {
		err := createManagerKeyScope(
			ns, defaultScope, rootKey, cryptoKeyPub, cryptoKeyPriv,
		)
		if err != nil {
			return maybeConvertDbError(err)
		}
	}

	// Before we proceed, we'll also store the root master private key
	// within the database in an encrypted format. This is required as in
	// the future, we may need to create additional scoped key managers.
	masterHDPrivKeyEnc, err := cryptoKeyPriv.Encrypt([]byte(rootKey.String()))
	if err != nil {
		return maybeConvertDbError(err)
	}
	masterHDPubKeyEnc, err := cryptoKeyPub.Encrypt([]byte(rootPubKey.String()))
	if err != nil {
		return maybeConvertDbError(err)
	}
	err = putMasterHDKeys(ns, masterHDPrivKeyEnc, masterHDPubKeyEnc)
	if err != nil {
		return maybeConvertDbError(err)
	}

	// Save the encrypted crypto keys to the database.
	err = putCryptoKeys(ns, cryptoKeyPubEnc, cryptoKeyPrivEnc,
		cryptoKeyScriptEnc)
	if err != nil {
		return maybeConvertDbError(err)
	}

	// Save the fact this is not a watching-only address manager to the
	// database.
	err = putWatchingOnly(ns, false)
	if err != nil {
		return maybeConvertDbError(err)
	}

	// Save the initial synced to state.
	//log.Info("&syncInfo.syncedTo ：",&syncInfo.syncedTo)
	err = PutSyncedTo(ns, &syncInfo.syncedTo)
	if err != nil {
		return maybeConvertDbError(err)
	}
	err = putChainHeight(ns, 0)
	if err != nil {
		return maybeConvertDbError(err)
	}
	err = putStartBlock(ns, &syncInfo.startBlock)
	if err != nil {
		return maybeConvertDbError(err)
	}
	// Use 48 hours as margin of safety for wallet birthday.
	return putBirthday(ns, birthday.Add(-48*time.Hour))
}

func Open(ns walletdb.ReadBucket, pubPassphrase []byte,
	chainParams *chaincfg.Params) (*Manager, error) {

	// Return an error if the manager has NOT already been created in the
	// given database namespace.
	exists := managerExists(ns)
	if !exists {
		str := "the specified address manager does not exist"
		return nil, managerError(ErrNoExist, str, nil)
	}

	return loadManager(ns, pubPassphrase, chainParams)
}

// newSecretKey generates a new secret key using the active secretKeyGen.
func newSecretKey(passphrase *[]byte,
	config *ScryptOptions) (*snacl.SecretKey, error) {

	secretKeyGenMtx.RLock()
	defer secretKeyGenMtx.RUnlock()
	return secretKeyGen(passphrase, config)
}

// defaultNewSecretKey returns a new secret key.  See newSecretKey.
func defaultNewSecretKey(passphrase *[]byte,
	config *ScryptOptions) (*snacl.SecretKey, error) {
	return snacl.NewSecretKey(passphrase, config.N, config.R, config.P)
}

var (
	// secretKeyGen is the inner method that is executed when calling
	// newSecretKey.
	secretKeyGen = defaultNewSecretKey

	// secretKeyGenMtx protects access to secretKeyGen, so that it can be
	// replaced in testing.
	secretKeyGenMtx sync.RWMutex
)

func loadManager(ns walletdb.ReadBucket, pubPassphrase []byte,
	chainParams *chaincfg.Params) (*Manager, error) {

	// Verify the version is neither too old or too new.
	version, err := fetchManagerVersion(ns)
	if err != nil {
		str := "failed to fetch version for update"
		return nil, managerError(ErrDatabase, str, err)
	}
	if version < latestMgrVersion {
		str := "database upgrade required"
		return nil, managerError(ErrUpgrade, str, nil)
	} else if version > latestMgrVersion {
		str := "database version is greater than latest understood version"
		return nil, managerError(ErrUpgrade, str, nil)
	}

	// Load whether or not the manager is watching-only from the db.
	watchingOnly, err := fetchWatchingOnly(ns)
	if err != nil {
		return nil, maybeConvertDbError(err)
	}

	// Load the master key params from the db.
	masterKeyPubParams, masterKeyPrivParams, err := fetchMasterKeyParams(ns)
	if err != nil {
		return nil, maybeConvertDbError(err)
	}
	// Load the crypto keys from the db.
	cryptoKeyPubEnc, cryptoKeyPrivEnc, cryptoKeyScriptEnc, err :=
		fetchCryptoKeys(ns)
	if err != nil {
		return nil, maybeConvertDbError(err)
	}

	// Load the sync state from the db.
	syncedTo, err := fetchSyncedTo(ns)
	if err != nil {
		return nil, maybeConvertDbError(err)
	}
	chainHeight, err := fetchChainHeight(ns)
	if err != nil {
		return nil, maybeConvertDbError(err)
	}
	startBlock, err := FetchStartBlock(ns)
	if err != nil {
		return nil, maybeConvertDbError(err)
	}
	birthday, err := fetchBirthday(ns)
	if err != nil {
		return nil, maybeConvertDbError(err)
	}

	// When not a watching-only manager, set the master private key params,
	// but don't derive it now since the manager starts off locked.
	var masterKeyPriv snacl.SecretKey
	if !watchingOnly {
		err := masterKeyPriv.Unmarshal(masterKeyPrivParams)
		if err != nil {
			str := "failed to unmarshal master private key"
			return nil, managerError(ErrCrypto, str, err)
		}
	}
	// Derive the master public key using the serialized params and provided
	// passphrase.
	var masterKeyPub snacl.SecretKey
	if err := masterKeyPub.Unmarshal(masterKeyPubParams); err != nil {
		str := "failed to unmarshal master public key"
		return nil, managerError(ErrCrypto, str, err)
	}
	if err := masterKeyPub.DeriveKey(&pubPassphrase); err != nil {
		str := "invalid passphrase for master public key"
		return nil, managerError(ErrWrongPassphrase, str, nil)
	}

	// Use the master public key to decrypt the crypto public key.
	cryptoKeyPub := &cryptoKey{snacl.CryptoKey{}}
	cryptoKeyPubCT, err := masterKeyPub.Decrypt(cryptoKeyPubEnc)
	if err != nil {
		str := "failed to decrypt crypto public key"
		return nil, managerError(ErrCrypto, str, err)
	}
	cryptoKeyPub.CopyBytes(cryptoKeyPubCT)

	// Create the sync state struct.
	syncInfo := newSyncState(startBlock, syncedTo)

	// Generate private passphrase salt.
	var privPassphraseSalt [saltSize]byte
	_, err = rand.Read(privPassphraseSalt[:])
	if err != nil {
		str := "failed to read random source for passphrase salt"
		return nil, managerError(ErrCrypto, str, err)
	}

	// Next, we'll need to load all known manager scopes from disk. Each
	// scope is on a distinct top-level path within our HD key chain.
	scopedManagers := make(map[KeyScope]*ScopedKeyManager)
	err = forEachKeyScope(ns, func(scope KeyScope) error {
		scopeSchema, err := fetchScopeAddrSchema(ns, &scope)
		if err != nil {
			return err
		}

		scopedManagers[scope] = &ScopedKeyManager{
			scope:      scope,
			addrSchema: *scopeSchema,
			addrs:      make(map[addrKey]ManagedAddress),
			acctInfo:   make(map[uint32]*accountInfo),
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Create new address manager with the given parameters.  Also,
	// override the defaults for the additional fields which are not
	// specified in the call to new with the values loaded from the
	// database.
	mgr := newManager(
		chainParams, &masterKeyPub, &masterKeyPriv,
		cryptoKeyPub, cryptoKeyPrivEnc, cryptoKeyScriptEnc, syncInfo,
		chainHeight, birthday, privPassphraseSalt, scopedManagers,
	)
	mgr.watchingOnly = watchingOnly

	for _, scopedManager := range scopedManagers {
		scopedManager.rootManager = mgr
	}
	return mgr, nil
}

// newManager returns a new locked address manager with the given parameters.
//func newManager(chainParams *chaincfg.Params, syncInfo *syncState,
//	birthday time.Time) *Manager {
//
//	m := &Manager{
//		chainParams:              chainParams,
//		syncState:                *syncInfo,
//		locked:                   true,
//		birthday:                 birthday,
//	}
//
//	return m
//}
// newManager returns a new locked address manager with the given parameters.
func newManager(chainParams *chaincfg.Params, masterKeyPub *snacl.SecretKey,
	masterKeyPriv *snacl.SecretKey, cryptoKeyPub EncryptorDecryptor,
	cryptoKeyPrivEncrypted, cryptoKeyScriptEncrypted []byte, syncInfo *syncState,
	chainHeight uint32, birthday time.Time, privPassphraseSalt [saltSize]byte,
	scopedManagers map[KeyScope]*ScopedKeyManager) *Manager {

	m := &Manager{
		chainParams:              chainParams,
		syncState:                *syncInfo,
		chainHeight:              chainHeight,
		locked:                   true,
		birthday:                 birthday,
		masterKeyPub:             masterKeyPub,
		masterKeyPriv:            masterKeyPriv,
		cryptoKeyPub:             cryptoKeyPub,
		cryptoKeyPrivEncrypted:   cryptoKeyPrivEncrypted,
		cryptoKeyPriv:            &cryptoKey{},
		cryptoKeyScriptEncrypted: cryptoKeyScriptEncrypted,
		cryptoKeyScript:          &cryptoKey{},
		privPassphraseSalt:       privPassphraseSalt,
		scopedManagers:           scopedManagers,
		externalAddrSchemas:      make(map[AddressType][]KeyScope),
		internalAddrSchemas:      make(map[AddressType][]KeyScope),
	}

	for _, sMgr := range m.scopedManagers {
		externalType := sMgr.AddrSchema().ExternalAddrType
		internalType := sMgr.AddrSchema().InternalAddrType
		scope := sMgr.Scope()

		m.externalAddrSchemas[externalType] = append(
			m.externalAddrSchemas[externalType], scope,
		)
		m.internalAddrSchemas[internalType] = append(
			m.internalAddrSchemas[internalType], scope,
		)
	}

	return m
}

// IsLocked returns whether or not the address managed is locked.  When it is
// unlocked, the decryption key needed to decrypt private keys used for signing
// is in memory.
func (m *Manager) IsLocked() bool {
	m.mtx.RLock()
	defer m.mtx.RUnlock()

	return m.isLocked()
}

// isLocked is an internal method returning whether or not the address manager
// is locked via an unprotected read.
//
// NOTE: The caller *MUST* acquire the Manager's mutex before invocation to
// avoid data races.
func (m *Manager) isLocked() bool {
	return m.locked
}

// deriveAccountKey derives the extended key for an account according to the
// hierarchy described by BIP0044 given the master node.
//
// In particular this is the hierarchical deterministic extended key path:
//   m/purpose'/<coin type>'/<account>'
func deriveAccountKey(coinTypeKey *bip32.Key,
	account uint32) (*bip32.Key, error) {
	return coinTypeKey.NewChildKey(account)
}

// ValidateAccountName validates the given account name and returns an error, if any.
func ValidateAccountName(name string) error {
	if name == "" {
		str := "accounts may not be named the empty string"
		return managerError(ErrInvalidAccount, str, nil)
	}
	if isReservedAccountName(name) {
		str := "reserved account name"
		return managerError(ErrInvalidAccount, str, nil)
	}
	return nil
}

// Address returns a managed address given the passed address if it is known to
// the address manager. A managed address differs from the passed address in
// that it also potentially contains extra information needed to sign
// transactions such as the associated private key for pay-to-pubkey and
// pay-to-pubkey-hash addresses and the script associated with
// pay-to-script-hash addresses.
func (m *Manager) Address(ns walletdb.ReadBucket,
	address types.Address) (ManagedAddress, error) {

	m.mtx.RLock()
	defer m.mtx.RUnlock()

	// We'll iterate through each of the known scoped managers, and see if
	// any of them now of the target address.
	for _, scopedMgr := range m.scopedManagers {
		addr, err := scopedMgr.Address(ns, address)
		if err != nil {
			continue
		}

		return addr, nil
	}

	// If the address wasn't known to any of the scoped managers, then
	// we'll return an error.
	str := fmt.Sprintf("unable to find key for addr %v", address)
	return nil, managerError(ErrAddressNotFound, str, nil)
}

// ForEachAccountAddress calls the given function with each address of
// the given account stored in the manager, breaking early on error.
func (m *Manager) ForEachAccountAddress(ns walletdb.ReadBucket, account uint32,
	fn func(maddr ManagedAddress) error) error {

	m.mtx.RLock()
	defer m.mtx.RUnlock()

	for _, scopedMgr := range m.scopedManagers {
		err := scopedMgr.ForEachAccountAddress(ns, account, fn)
		if err != nil {
			return err
		}
	}

	return nil
}

// createManagerKeyScope creates a new key scoped for a target manager's scope.
// This partitions key derivation for a particular purpose+coin tuple, allowing
// multiple address derivation schems to be maintained concurrently.
func createManagerKeyScope(ns walletdb.ReadWriteBucket,
	scope KeyScope, root *bip32.Key,
	cryptoKeyPub, cryptoKeyPriv EncryptorDecryptor) error {

	// Derive the cointype key according to the passed scope.
	coinTypeKeyPriv, err := deriveCoinTypeKey(root, scope)
	if err != nil {
		str := "failed to derive cointype extended key"
		return managerError(ErrKeyChain, str, err)
	}
	// Derive the account key for the first account according our
	// BIP0044-like derivation.
	acctKeyPriv, err := deriveAccountKey(coinTypeKeyPriv, 0)
	if err != nil {
		return err
	}

	// The address manager needs the public extended key for the account.
	acctKeyPub := acctKeyPriv.PublicKey()

	// Encrypt the cointype keys with the associated crypto keys.
	coinTypeKeyPub := coinTypeKeyPriv.PublicKey()

	coinTypePubEnc, err := cryptoKeyPub.Encrypt([]byte(coinTypeKeyPub.String()))
	if err != nil {
		str := "failed to encrypt cointype public key"
		return managerError(ErrCrypto, str, err)
	}
	coinTypePrivEnc, err := cryptoKeyPriv.Encrypt([]byte(coinTypeKeyPriv.String()))
	if err != nil {
		str := "failed to encrypt cointype private key"
		return managerError(ErrCrypto, str, err)
	}

	// Encrypt the default account keys with the associated crypto keys.
	acctPubEnc, err := cryptoKeyPub.Encrypt([]byte(acctKeyPub.String()))
	if err != nil {
		str := "failed to  encrypt public key for account 0"
		return managerError(ErrCrypto, str, err)
	}
	acctPrivEnc, err := cryptoKeyPriv.Encrypt([]byte(acctKeyPriv.String()))
	if err != nil {
		str := "failed to encrypt private key for account 0"
		return managerError(ErrCrypto, str, err)
	}

	// Save the encrypted cointype keys to the database.
	err = putCoinTypeKeys(ns, &scope, coinTypePubEnc, coinTypePrivEnc)
	if err != nil {
		return err
	}

	// Save the information for the default account to the database.
	err = putAccountInfo(
		ns, &scope, DefaultAccountNum, acctPubEnc, acctPrivEnc, 0, 0,
		defaultAccountName,
	)
	if err != nil {
		return err
	}

	return putAccountInfo(
		ns, &scope, ImportedAddrAccount, nil, nil, 0, 0,
		ImportedAddrAccountName,
	)
}

// AddrAccount returns the account to which the given address belongs. We also
// return the scoped manager that owns the addr+account combo.
func (m *Manager) AddrAccount(ns walletdb.ReadBucket,
	address types.Address) (*ScopedKeyManager, uint32, error) {

	m.mtx.RLock()
	defer m.mtx.RUnlock()

	for _, scopedMgr := range m.scopedManagers {
		if _, err := scopedMgr.Address(ns, address); err != nil {
			continue
		}

		// We've found the manager that this address belongs to, so we
		// can retrieve the address' account along with the manager
		// that the addr belongs to.
		accNo, err := scopedMgr.AddrAccount(ns, address)
		if err != nil {
			return nil, 0, err
		}

		return scopedMgr, accNo, err
	}

	// If we get to this point, then we weren't able to find the address in
	// any of the managers, so we'll exit with an error.
	str := fmt.Sprintf("unable to find key for addr %v", address)
	return nil, 0, managerError(ErrAddressNotFound, str, nil)
}

// deriveCoinTypeKey derives the cointype key which can be used to derive the
// extended key for an account according to the hierarchy described by BIP0044
// given the coin type key.
//
// In particular this is the hierarchical deterministic extended key path:
// m/purpose'/<coin type>'
func deriveCoinTypeKey(masterNode *bip32.Key,
	scope KeyScope) (*bip32.Key, error) {

	// The hierarchy described by BIP0043 is:
	//  m/<purpose>'/*
	//
	// This is further extended by BIP0044 to:
	//  m/44'/<coin type>'/<account>'/<branch>/<address index>
	//
	// However, as this is a generic key store for any family for BIP0044
	// standards, we'll use the custom scope to govern our key derivation.
	//
	// The branch is 0 for external addresses and 1 for internal addresses.

	// Derive the purpose key as a child of the master node.
	purpose, err := masterNode.NewChildKey(scope.Purpose + HardenedKeyStart)
	if err != nil {
		return nil, err
	}

	// Derive the coin type key as a child of the purpose key.
	coinTypeKey, err := purpose.NewChildKey(scope.Coin + HardenedKeyStart)
	if err != nil {
		return nil, err
	}

	return coinTypeKey, nil
}

// Unlock derives the master private key from the specified passphrase.  An
// invalid passphrase will return an error.  Otherwise, the derived secret key
// is stored in memory until the address manager is locked.  Any failures that
// occur during this function will result in the address manager being locked,
// even if it was already unlocked prior to calling this function.
//
// This function will return an error if invoked on a watching-only address
// manager.
func (m *Manager) Unlock(ns walletdb.ReadBucket, passphrase []byte) error {
	// A watching-only address manager can't be unlocked.
	if m.watchingOnly {
		return managerError(ErrWatchingOnly, errWatchingOnly, nil)
	}

	m.mtx.Lock()
	defer m.mtx.Unlock()

	// Avoid actually unlocking if the manager is already unlocked
	// and the passphrases match.
	if !m.locked {
		saltedPassphrase := append(m.privPassphraseSalt[:],
			passphrase...)
		hashedPassphrase := sha512.Sum512(saltedPassphrase)
		zero.Bytes(saltedPassphrase)
		if hashedPassphrase != m.hashedPrivPassphrase {
			m.lock()
			str := "invalid passphrase for master private key"
			return managerError(ErrWrongPassphrase, str, nil)
		}
		return nil
	}

	// Derive the master private key using the provided passphrase.
	if err := m.masterKeyPriv.DeriveKey(&passphrase); err != nil {
		m.lock()
		if err == snacl.ErrInvalidPassword {
			str := "invalid passphrase for master private key"
			return managerError(ErrWrongPassphrase, str, nil)
		}

		str := "failed to derive master private key"
		return managerError(ErrCrypto, str, err)
	}

	// Use the master private key to decrypt the crypto private key.
	decryptedKey, err := m.masterKeyPriv.Decrypt(m.cryptoKeyPrivEncrypted)
	if err != nil {
		m.lock()
		str := "failed to decrypt crypto private key"
		return managerError(ErrCrypto, str, err)
	}
	m.cryptoKeyPriv.CopyBytes(decryptedKey)
	zero.Bytes(decryptedKey)

	// Use the crypto private key to decrypt all of the account private
	// extended keys.
	for _, manager := range m.scopedManagers {
		for account, acctInfo := range manager.acctInfo {
			decrypted, err := m.cryptoKeyPriv.Decrypt(acctInfo.acctKeyEncrypted)
			if err != nil {
				m.lock()
				str := fmt.Sprintf("failed to decrypt account %d "+
					"private key", account)
				return managerError(ErrCrypto, str, err)
			}

			acctKeyPriv, err := bip32.B58Deserialize(string(decrypted), bip32.DefaultBip32Version)
			zero.Bytes(decrypted)
			if err != nil {
				m.lock()
				str := fmt.Sprintf("failed to regenerate account %d "+
					"extended key", account)
				return managerError(ErrKeyChain, str, err)
			}
			acctInfo.acctKeyPriv = acctKeyPriv
		}

		// We'll also derive any private keys that are pending due to
		// them being created while the address manager was locked.
		for _, info := range manager.deriveOnUnlock {
			addressKey, err := manager.deriveKeyFromPath(
				ns, info.managedAddr.Account(), info.branch,
				info.index, true,
			)
			if err != nil {
				m.lock()
				return err
			}

			// It's ok to ignore the error here since it can only
			// fail if the extended key is not private, however it
			// was just derived as a private key.
			privKey, _ := ecc.PrivKeyFromBytes(addressKey.Key)

			privKeyBytes := privKey.Serialize()
			privKeyEncrypted, err := m.cryptoKeyPriv.Encrypt(privKeyBytes)
			zero.BigInt(privKey.D)
			if err != nil {
				m.lock()
				str := fmt.Sprintf("failed to encrypt private key for "+
					"address %s", info.managedAddr.Address())
				return managerError(ErrCrypto, str, err)
			}

			switch a := info.managedAddr.(type) {
			case *managedAddress:
				a.privKeyEncrypted = privKeyEncrypted
				a.privKeyCT = privKeyBytes
			case *scriptAddress:
			}

			// Avoid re-deriving this key on subsequent unlocks.
			manager.deriveOnUnlock[0] = nil
			manager.deriveOnUnlock = manager.deriveOnUnlock[1:]
		}
	}

	m.locked = false
	saltedPassphrase := append(m.privPassphraseSalt[:], passphrase...)
	m.hashedPrivPassphrase = sha512.Sum512(saltedPassphrase)
	zero.Bytes(saltedPassphrase)
	return nil
}

// FetchScopedKeyManager attempts to fetch an active scoped manager according to
// its registered scope. If the manger is found, then a nil error is returned
// along with the active scoped manager. Otherwise, a nil manager and a non-nil
// error will be returned.
func (m *Manager) FetchScopedKeyManager(scope KeyScope) (*ScopedKeyManager, error) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()

	sm, ok := m.scopedManagers[scope]
	if !ok {
		str := fmt.Sprintf("scope %v not found", scope)
		return nil, managerError(ErrScopeNotFound, str, nil)
	}

	return sm, nil
}
