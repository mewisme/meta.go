package meta

import (
	"context"
	"crypto/rand"
	"errors"
	"io"
	"math/big"
	"sort"
	"strings"
	"sync"

	"github.com/google/uuid"
	"go.mau.fi/whatsmeow/proto/waAdv"
	"go.mau.fi/whatsmeow/store"
	waTypes "go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/util/keys"
)

const maxRegistrationID = 16380

type DeviceStore struct {
	*store.NoopStore

	mu sync.RWMutex

	device       *store.Device
	identities   map[string][32]byte
	sessions     map[string][]byte
	preKeys      map[uint32]*keys.PreKey
	uploaded     map[uint32]struct{}
	senderKeys   map[string][]byte
	nextPreKeyID uint32
}

var _ store.AllStores = (*DeviceStore)(nil)
var _ store.DeviceContainer = (*DeviceStore)(nil)

func NewMemoryDeviceStore() (*DeviceStore, error) { return newMemoryDeviceStore(rand.Reader) }

func newMemoryDeviceStore(random io.Reader) (*DeviceStore, error) {
	if random == nil {
		return nil, errors.New("nil cryptographic random source")
	}
	registration, err := randInt(random, maxRegistrationID)
	if err != nil {
		return nil, err
	}
	advSecret := make([]byte, 32)
	if _, err := io.ReadFull(random, advSecret); err != nil {
		return nil, err
	}
	identity := keys.NewKeyPair()
	device := &store.Device{
		NoiseKey:       keys.NewKeyPair(),
		IdentityKey:    identity,
		SignedPreKey:   identity.CreateSignedPreKey(1),
		RegistrationID: uint32(registration + 1),
		AdvSecretKey:   advSecret,
		FacebookUUID:   uuid.New(),
		Account: &waAdv.ADVSignedDeviceIdentity{
			Details: make([]byte, 0), AccountSignatureKey: make([]byte, 32), AccountSignature: make([]byte, 64), DeviceSignature: make([]byte, 64),
		},
	}
	ds := &DeviceStore{
		NoopStore:    &store.NoopStore{Error: errors.New("unsupported device store operation")},
		device:       device,
		identities:   make(map[string][32]byte),
		sessions:     make(map[string][]byte),
		preKeys:      make(map[uint32]*keys.PreKey),
		uploaded:     make(map[uint32]struct{}),
		senderKeys:   make(map[string][]byte),
		nextPreKeyID: 1,
	}
	device.SetAllStores(ds)
	device.LIDs = ds
	device.Container = ds
	device.Initialized = true
	return ds, nil
}

func randInt(reader io.Reader, max int64) (int64, error) {
	value, err := rand.Int(reader, big.NewInt(max))
	if err != nil {
		return 0, err
	}
	return value.Int64(), nil
}

func (s *DeviceStore) Device() *store.Device { return s.device }

func (s *DeviceStore) PutDevice(ctx context.Context, device *store.Device) error {
	if err := ctxErr(ctx); err != nil {
		return err
	}
	if device != s.device {
		return errors.New("device does not belong to this store")
	}
	return nil
}

func (s *DeviceStore) DeleteDevice(ctx context.Context, device *store.Device) error {
	if err := ctxErr(ctx); err != nil {
		return err
	}
	if device != s.device {
		return errors.New("device does not belong to this store")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	device.Deleted = true
	clear(s.identities)
	clear(s.sessions)
	clear(s.preKeys)
	clear(s.uploaded)
	clear(s.senderKeys)
	return nil
}

func (s *DeviceStore) PutIdentity(ctx context.Context, address string, key [32]byte) error {
	if err := ctxErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	s.identities[address] = key
	s.mu.Unlock()
	return nil
}

func (s *DeviceStore) DeleteIdentity(ctx context.Context, address string) error {
	if err := ctxErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	delete(s.identities, address)
	s.mu.Unlock()
	return nil
}

func (s *DeviceStore) DeleteAllIdentities(ctx context.Context, phone string) error {
	if err := ctxErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	for address := range s.identities {
		if addressForPhone(address, phone) {
			delete(s.identities, address)
		}
	}
	s.mu.Unlock()
	return nil
}

func (s *DeviceStore) IsTrustedIdentity(ctx context.Context, address string, key [32]byte) (bool, error) {
	if err := ctxErr(ctx); err != nil {
		return false, err
	}
	s.mu.RLock()
	stored, exists := s.identities[address]
	s.mu.RUnlock()
	return !exists || stored == key, nil
}

func (s *DeviceStore) GetSession(ctx context.Context, address string) ([]byte, error) {
	if err := ctxErr(ctx); err != nil {
		return nil, err
	}
	s.mu.RLock()
	result := clone(s.sessions[address])
	s.mu.RUnlock()
	return result, nil
}

func (s *DeviceStore) HasSession(ctx context.Context, address string) (bool, error) {
	if err := ctxErr(ctx); err != nil {
		return false, err
	}
	s.mu.RLock()
	_, exists := s.sessions[address]
	s.mu.RUnlock()
	return exists, nil
}

func (s *DeviceStore) GetManySessions(ctx context.Context, addresses []string) (map[string][]byte, error) {
	if err := ctxErr(ctx); err != nil {
		return nil, err
	}
	s.mu.RLock()
	result := make(map[string][]byte, len(addresses))
	for _, address := range addresses {
		result[address] = clone(s.sessions[address])
	}
	s.mu.RUnlock()
	return result, nil
}

func (s *DeviceStore) PutSession(ctx context.Context, address string, session []byte) error {
	return s.PutManySessions(ctx, map[string][]byte{address: session})
}

func (s *DeviceStore) PutManySessions(ctx context.Context, sessions map[string][]byte) error {
	if err := ctxErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	for address, session := range sessions {
		s.sessions[address] = clone(session)
	}
	s.mu.Unlock()
	return nil
}

func (s *DeviceStore) DeleteSession(ctx context.Context, address string) error {
	if err := ctxErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	delete(s.sessions, address)
	s.mu.Unlock()
	return nil
}

func (s *DeviceStore) DeleteAllSessions(ctx context.Context, phone string) error {
	if err := ctxErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	for address := range s.sessions {
		if addressForPhone(address, phone) {
			delete(s.sessions, address)
		}
	}
	s.mu.Unlock()
	return nil
}

func (s *DeviceStore) MigratePNToLID(ctx context.Context, pn, lid waTypes.JID) error {
	if err := ctxErr(ctx); err != nil {
		return err
	}
	oldPrefix, newPrefix := pn.SignalAddressUser(), lid.SignalAddressUser()
	if oldPrefix == "" || newPrefix == "" || oldPrefix == newPrefix {
		return nil
	}
	s.mu.Lock()
	for address, session := range s.sessions {
		if addressForPhone(address, oldPrefix) {
			s.sessions[newPrefix+strings.TrimPrefix(address, oldPrefix)] = session
			delete(s.sessions, address)
		}
	}
	for address, identity := range s.identities {
		if addressForPhone(address, oldPrefix) {
			s.identities[newPrefix+strings.TrimPrefix(address, oldPrefix)] = identity
			delete(s.identities, address)
		}
	}
	s.mu.Unlock()
	return nil
}

func (s *DeviceStore) GetOrGenPreKeys(ctx context.Context, count uint32) ([]*keys.PreKey, error) {
	if err := ctxErr(ctx); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := make([]uint32, 0, len(s.preKeys))
	for id := range s.preKeys {
		if _, uploaded := s.uploaded[id]; !uploaded {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	result := make([]*keys.PreKey, 0, count)
	for _, id := range ids {
		if uint32(len(result)) == count {
			break
		}
		result = append(result, s.preKeys[id])
	}
	for uint32(len(result)) < count {
		key := keys.NewPreKey(s.nextPreKeyID)
		s.preKeys[s.nextPreKeyID] = key
		s.nextPreKeyID++
		result = append(result, key)
	}
	return result, nil
}

func (s *DeviceStore) GenOnePreKey(ctx context.Context) (*keys.PreKey, error) {
	keys, err := s.GetOrGenPreKeys(ctx, 1)
	if err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return nil, errors.New("failed to generate pre-key")
	}
	if err := s.MarkPreKeysAsUploaded(ctx, keys[0].KeyID); err != nil {
		return nil, err
	}
	return keys[0], nil
}

func (s *DeviceStore) GetPreKey(ctx context.Context, id uint32) (*keys.PreKey, error) {
	if err := ctxErr(ctx); err != nil {
		return nil, err
	}
	s.mu.RLock()
	key := s.preKeys[id]
	s.mu.RUnlock()
	return key, nil
}

func (s *DeviceStore) RemovePreKey(ctx context.Context, id uint32) error {
	if err := ctxErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	delete(s.preKeys, id)
	delete(s.uploaded, id)
	s.mu.Unlock()
	return nil
}

func (s *DeviceStore) MarkPreKeysAsUploaded(ctx context.Context, upToID uint32) error {
	if err := ctxErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	for id := range s.preKeys {
		if id <= upToID {
			s.uploaded[id] = struct{}{}
		}
	}
	s.mu.Unlock()
	return nil
}

func (s *DeviceStore) UploadedPreKeyCount(ctx context.Context) (int, error) {
	if err := ctxErr(ctx); err != nil {
		return 0, err
	}
	s.mu.RLock()
	count := len(s.uploaded)
	s.mu.RUnlock()
	return count, nil
}

func (s *DeviceStore) PutSenderKey(ctx context.Context, group, user string, session []byte) error {
	if err := ctxErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	s.senderKeys[group+":"+user] = clone(session)
	s.mu.Unlock()
	return nil
}

func (s *DeviceStore) GetSenderKey(ctx context.Context, group, user string) ([]byte, error) {
	if err := ctxErr(ctx); err != nil {
		return nil, err
	}
	s.mu.RLock()
	result := clone(s.senderKeys[group+":"+user])
	s.mu.RUnlock()
	return result, nil
}

func clone(value []byte) []byte                  { return append([]byte(nil), value...) }
func addressForPhone(address, phone string) bool { return strings.HasPrefix(address, phone+":") }

func ctxErr(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}
