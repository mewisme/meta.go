package meta

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"go.mau.fi/whatsmeow/proto/waAdv"
	"go.mau.fi/whatsmeow/store"
	waTypes "go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/util/keys"
	"google.golang.org/protobuf/proto"
)

const deviceStateVersion = 1

type persistedPreKey struct {
	ID        uint32    `json:"id"`
	Private   [32]byte  `json:"private"`
	Signature *[64]byte `json:"signature,omitempty"`
}

type persistedDeviceState struct {
	Version               int                 `json:"version"`
	NoisePrivate          [32]byte            `json:"noise_private"`
	IdentityPrivate       [32]byte            `json:"identity_private"`
	SignedPreKey          persistedPreKey     `json:"signed_pre_key"`
	RegistrationID        uint32              `json:"registration_id"`
	AdvSecretKey          []byte              `json:"adv_secret_key"`
	ID                    string              `json:"id,omitempty"`
	LID                   string              `json:"lid,omitempty"`
	Account               []byte              `json:"account,omitempty"`
	Platform              string              `json:"platform,omitempty"`
	BusinessName          string              `json:"business_name,omitempty"`
	PushName              string              `json:"push_name,omitempty"`
	LIDMigrationTimestamp int64               `json:"lid_migration_timestamp,omitempty"`
	FacebookUUID          string              `json:"facebook_uuid"`
	Initialized           bool                `json:"initialized"`
	Deleted               bool                `json:"deleted"`
	Identities            map[string][32]byte `json:"identities,omitempty"`
	Sessions              map[string][]byte   `json:"sessions,omitempty"`
	PreKeys               []persistedPreKey   `json:"pre_keys,omitempty"`
	Uploaded              []uint32            `json:"uploaded,omitempty"`
	SenderKeys            map[string][]byte   `json:"sender_keys,omitempty"`
	NextPreKeyID          uint32              `json:"next_pre_key_id"`
}

func (s *DeviceStore) Snapshot() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snapshotLocked()
}

func LoadDeviceStore(data []byte) (*DeviceStore, error) {
	var header struct {
		Version int `json:"version"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return nil, fmt.Errorf("decode device state: %w", err)
	}
	var state persistedDeviceState
	if header.Version == 0 {
		legacy, err := legacyDeviceState(data)
		if err != nil {
			return nil, err
		}
		state = legacy
	} else if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("decode device state: %w", err)
	}
	if state.Version != deviceStateVersion {
		return nil, fmt.Errorf("unsupported device state version %d", state.Version)
	}
	ds := &DeviceStore{NoopStore: &store.NoopStore{Error: errors.New("unsupported device store operation")}}
	if err := ds.restoreLocked(state); err != nil {
		return nil, err
	}
	return ds, nil
}

type legacyDeviceJSON struct {
	NoiseKeyPriv     string            `json:"noise_key_priv"`
	IdentityKeyPriv  string            `json:"identity_key_priv"`
	SignedPreKeyPriv string            `json:"signed_pre_key_priv"`
	SignedPreKeyID   uint32            `json:"signed_pre_key_id"`
	SignedPreKeySig  string            `json:"signed_pre_key_sig"`
	RegistrationID   uint32            `json:"registration_id"`
	AdvSecretKey     string            `json:"adv_secret_key"`
	FacebookUUID     string            `json:"facebook_uuid"`
	JIDUser          string            `json:"jid_user,omitempty"`
	JIDDevice        uint16            `json:"jid_device,omitempty"`
	LID              string            `json:"lid,omitempty"`
	LIDMigrationTS   int64             `json:"lid_migration_timestamp,omitempty"`
	PushName         string            `json:"push_name,omitempty"`
	BusinessName     string            `json:"business_name,omitempty"`
	Platform         string            `json:"platform,omitempty"`
	Account          string            `json:"account,omitempty"`
	Identities       map[string]string `json:"identities,omitempty"`
	Sessions         map[string]string `json:"sessions,omitempty"`
	PreKeys          map[string]string `json:"pre_keys,omitempty"`
	UploadedPreKeys  []uint32          `json:"uploaded_pre_keys"`
	SenderKeys       map[string]string `json:"sender_keys,omitempty"`
	NextPreKeyID     uint32            `json:"next_pre_key_id"`
}

func legacyDeviceState(data []byte) (persistedDeviceState, error) {
	var legacy legacyDeviceJSON
	if err := json.Unmarshal(data, &legacy); err != nil {
		return persistedDeviceState{}, fmt.Errorf("decode legacy device state: %w", err)
	}
	decode32 := func(name, value string) ([32]byte, error) {
		var result [32]byte
		decoded, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			return result, fmt.Errorf("decode legacy %s: %w", name, err)
		}
		if len(decoded) != len(result) {
			return result, fmt.Errorf("invalid legacy %s length: got %d, want %d", name, len(decoded), len(result))
		}
		copy(result[:], decoded)
		return result, nil
	}
	noise, err := decode32("noise private key", legacy.NoiseKeyPriv)
	if err != nil {
		return persistedDeviceState{}, err
	}
	identity, err := decode32("identity private key", legacy.IdentityKeyPriv)
	if err != nil {
		return persistedDeviceState{}, err
	}
	signed, err := decode32("signed pre-key private key", legacy.SignedPreKeyPriv)
	if err != nil {
		return persistedDeviceState{}, err
	}
	sigBytes, err := base64.StdEncoding.DecodeString(legacy.SignedPreKeySig)
	if err != nil || len(sigBytes) != 64 {
		if err != nil {
			return persistedDeviceState{}, fmt.Errorf("decode legacy signed pre-key signature: %w", err)
		}
		return persistedDeviceState{}, fmt.Errorf("invalid legacy signed pre-key signature length: got %d, want 64", len(sigBytes))
	}
	var signature [64]byte
	copy(signature[:], sigBytes)
	adv, err := base64.StdEncoding.DecodeString(legacy.AdvSecretKey)
	if err != nil || len(adv) != 32 {
		if err != nil {
			return persistedDeviceState{}, fmt.Errorf("decode legacy ADV secret key: %w", err)
		}
		return persistedDeviceState{}, fmt.Errorf("invalid legacy ADV secret key length: got %d, want 32", len(adv))
	}
	if legacy.RegistrationID < 1 || legacy.RegistrationID > maxRegistrationID {
		return persistedDeviceState{}, fmt.Errorf("invalid legacy registration ID %d", legacy.RegistrationID)
	}
	state := persistedDeviceState{Version: deviceStateVersion, NoisePrivate: noise, IdentityPrivate: identity, SignedPreKey: persistedPreKey{ID: legacy.SignedPreKeyID, Private: signed, Signature: &signature}, RegistrationID: legacy.RegistrationID, AdvSecretKey: clone(adv), LID: legacy.LID, Platform: legacy.Platform, BusinessName: legacy.BusinessName, PushName: legacy.PushName, LIDMigrationTimestamp: legacy.LIDMigrationTS, FacebookUUID: legacy.FacebookUUID, Initialized: true, Identities: map[string][32]byte{}, Sessions: map[string][]byte{}, SenderKeys: map[string][]byte{}, NextPreKeyID: legacy.NextPreKeyID}
	if legacy.JIDUser != "" {
		state.ID = waTypes.JID{User: legacy.JIDUser, Device: legacy.JIDDevice, Server: waTypes.MessengerServer}.String()
	}
	if legacy.Account != "" {
		state.Account, err = base64.StdEncoding.DecodeString(legacy.Account)
		if err != nil {
			return persistedDeviceState{}, fmt.Errorf("decode legacy account: %w", err)
		}
	}
	for address, encoded := range legacy.Identities {
		value, decodeErr := decode32("identity", encoded)
		if decodeErr != nil {
			return persistedDeviceState{}, fmt.Errorf("legacy identity %q: %w", address, decodeErr)
		}
		state.Identities[address] = value
	}
	for address, encoded := range legacy.Sessions {
		value, decodeErr := base64.StdEncoding.DecodeString(encoded)
		if decodeErr != nil {
			return persistedDeviceState{}, fmt.Errorf("decode legacy session %q: %w", address, decodeErr)
		}
		state.Sessions[address] = value
	}
	state.PreKeys = make([]persistedPreKey, 0, len(legacy.PreKeys))
	for idText, encoded := range legacy.PreKeys {
		id, parseErr := strconv.ParseUint(idText, 10, 32)
		if parseErr != nil {
			return persistedDeviceState{}, fmt.Errorf("parse legacy pre-key ID %q: %w", idText, parseErr)
		}
		private, decodeErr := decode32("pre-key private key", encoded)
		if decodeErr != nil {
			return persistedDeviceState{}, fmt.Errorf("legacy pre-key %d: %w", id, decodeErr)
		}
		state.PreKeys = append(state.PreKeys, persistedPreKey{ID: uint32(id), Private: private})
	}
	if legacy.UploadedPreKeys == nil {
		state.Uploaded = make([]uint32, 0, len(state.PreKeys))
		for _, preKey := range state.PreKeys {
			state.Uploaded = append(state.Uploaded, preKey.ID)
		}
	} else {
		state.Uploaded = append([]uint32(nil), legacy.UploadedPreKeys...)
	}
	for key, encoded := range legacy.SenderKeys {
		value, decodeErr := base64.StdEncoding.DecodeString(encoded)
		if decodeErr != nil {
			return persistedDeviceState{}, fmt.Errorf("decode legacy sender key %q: %w", key, decodeErr)
		}
		state.SenderKeys[key] = value
	}
	if state.NextPreKeyID == 0 {
		state.NextPreKeyID = 1
	}
	for _, preKey := range state.PreKeys {
		if preKey.ID >= state.NextPreKeyID {
			state.NextPreKeyID = preKey.ID + 1
		}
	}
	return state, nil
}

func (s *DeviceStore) WithPersistence(persist func(context.Context, []byte) error) *DeviceStore {
	s.mu.Lock()
	s.persist = persist
	s.mu.Unlock()
	return s
}

func (s *DeviceStore) persistLocked(ctx context.Context, before persistedDeviceState) error {
	if s.persist == nil {
		return nil
	}
	data, err := s.snapshotLocked()
	if err != nil {
		_ = s.restoreLocked(before)
		return err
	}
	if err := s.persist(ctx, data); err != nil {
		_ = s.restoreLocked(before)
		return err
	}
	return nil
}

func (s *DeviceStore) mutate(ctx context.Context, fn func()) error {
	if err := ctxErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	before, err := s.stateLocked()
	if err != nil {
		return err
	}
	fn()
	return s.persistLocked(ctx, before)
}

func (s *DeviceStore) stateLocked() (persistedDeviceState, error) {
	data, err := s.snapshotLocked()
	if err != nil {
		return persistedDeviceState{}, err
	}
	var state persistedDeviceState
	if err := json.Unmarshal(data, &state); err != nil {
		return persistedDeviceState{}, err
	}
	return state, nil
}

func (s *DeviceStore) snapshotLocked() ([]byte, error) {
	if s.device == nil || s.device.NoiseKey == nil || s.device.NoiseKey.Priv == nil || s.device.IdentityKey == nil || s.device.IdentityKey.Priv == nil || s.device.SignedPreKey == nil || s.device.SignedPreKey.Priv == nil {
		return nil, errors.New("device state is incomplete")
	}
	state := persistedDeviceState{
		Version:               deviceStateVersion,
		NoisePrivate:          *s.device.NoiseKey.Priv,
		IdentityPrivate:       *s.device.IdentityKey.Priv,
		SignedPreKey:          persistedPreKey{ID: s.device.SignedPreKey.KeyID, Private: *s.device.SignedPreKey.Priv, Signature: s.device.SignedPreKey.Signature},
		RegistrationID:        s.device.RegistrationID,
		AdvSecretKey:          clone(s.device.AdvSecretKey),
		LID:                   s.device.LID.String(),
		Platform:              s.device.Platform,
		BusinessName:          s.device.BusinessName,
		PushName:              s.device.PushName,
		LIDMigrationTimestamp: s.device.LIDMigrationTimestamp,
		FacebookUUID:          s.device.FacebookUUID.String(),
		Initialized:           s.device.Initialized,
		Deleted:               s.device.Deleted,
		Identities:            cloneIdentities(s.identities),
		Sessions:              cloneBytesMap(s.sessions),
		SenderKeys:            cloneBytesMap(s.senderKeys),
		NextPreKeyID:          s.nextPreKeyID,
	}
	if s.device.ID != nil {
		state.ID = s.device.ID.String()
	}
	if s.device.Account != nil {
		account, err := proto.Marshal(s.device.Account)
		if err != nil {
			return nil, err
		}
		state.Account = account
	}
	state.PreKeys = make([]persistedPreKey, 0, len(s.preKeys))
	for _, preKey := range s.preKeys {
		if preKey == nil || preKey.Priv == nil {
			continue
		}
		state.PreKeys = append(state.PreKeys, persistedPreKey{ID: preKey.KeyID, Private: *preKey.Priv, Signature: preKey.Signature})
	}
	state.Uploaded = make([]uint32, 0, len(s.uploaded))
	for id := range s.uploaded {
		state.Uploaded = append(state.Uploaded, id)
	}
	return json.Marshal(state)
}

func (s *DeviceStore) restoreLocked(state persistedDeviceState) error {
	if state.RegistrationID < 1 || state.RegistrationID > maxRegistrationID {
		return fmt.Errorf("invalid device registration ID %d", state.RegistrationID)
	}
	if zeroKey(state.NoisePrivate) || zeroKey(state.IdentityPrivate) || zeroKey(state.SignedPreKey.Private) {
		return errors.New("device state contains empty key material")
	}
	if len(state.AdvSecretKey) != 32 {
		return fmt.Errorf("invalid device ADV secret key length %d", len(state.AdvSecretKey))
	}
	if state.FacebookUUID == "" {
		return errors.New("device state is missing Facebook UUID")
	}
	noise := keys.NewKeyPairFromPrivateKey(state.NoisePrivate)
	identity := keys.NewKeyPairFromPrivateKey(state.IdentityPrivate)
	signed := &keys.PreKey{KeyPair: *keys.NewKeyPairFromPrivateKey(state.SignedPreKey.Private), KeyID: state.SignedPreKey.ID, Signature: state.SignedPreKey.Signature}
	device := &store.Device{
		NoiseKey: noise, IdentityKey: identity, SignedPreKey: signed, RegistrationID: state.RegistrationID, AdvSecretKey: clone(state.AdvSecretKey),
		Platform: state.Platform, BusinessName: state.BusinessName, PushName: state.PushName, LIDMigrationTimestamp: state.LIDMigrationTimestamp,
		Initialized: state.Initialized, Deleted: state.Deleted,
	}
	if state.FacebookUUID != "" {
		parsed, err := uuid.Parse(state.FacebookUUID)
		if err != nil {
			return fmt.Errorf("parse device UUID: %w", err)
		}
		device.FacebookUUID = parsed
	}
	if state.ID != "" {
		jid, err := waTypes.ParseJID(state.ID)
		if err != nil {
			return fmt.Errorf("parse device JID: %w", err)
		}
		device.ID = &jid
	}
	if state.LID != "" {
		jid, err := waTypes.ParseJID(state.LID)
		if err != nil {
			return fmt.Errorf("parse device LID: %w", err)
		}
		device.LID = jid
	}
	if len(state.Account) > 0 {
		account := new(waAdv.ADVSignedDeviceIdentity)
		if err := proto.Unmarshal(state.Account, account); err != nil {
			return fmt.Errorf("decode device account: %w", err)
		}
		device.Account = account
	} else {
		device.Account = &waAdv.ADVSignedDeviceIdentity{}
	}
	s.device = device
	s.identities = cloneIdentities(state.Identities)
	s.sessions = cloneBytesMap(state.Sessions)
	s.preKeys = make(map[uint32]*keys.PreKey, len(state.PreKeys))
	for _, item := range state.PreKeys {
		s.preKeys[item.ID] = &keys.PreKey{KeyPair: *keys.NewKeyPairFromPrivateKey(item.Private), KeyID: item.ID, Signature: item.Signature}
	}
	s.uploaded = make(map[uint32]struct{}, len(state.Uploaded))
	for _, id := range state.Uploaded {
		s.uploaded[id] = struct{}{}
	}
	s.senderKeys = cloneBytesMap(state.SenderKeys)
	s.nextPreKeyID = state.NextPreKeyID
	if s.nextPreKeyID == 0 {
		s.nextPreKeyID = 1
	}
	device.SetAllStores(s)
	device.LIDs = s
	device.Container = s
	return nil
}

func zeroKey(key [32]byte) bool {
	for _, value := range key {
		if value != 0 {
			return false
		}
	}
	return true
}

func cloneBytesMap(values map[string][]byte) map[string][]byte {
	result := make(map[string][]byte, len(values))
	for key, value := range values {
		result[key] = clone(value)
	}
	return result
}

func cloneIdentities(values map[string][32]byte) map[string][32]byte {
	result := make(map[string][32]byte, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}
