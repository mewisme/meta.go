package meta

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"

	"google.golang.org/protobuf/proto"
)

func TestDeviceStateRoundTrip(t *testing.T) {
	ctx := context.Background()
	store, err := NewMemoryDeviceStore()
	if err != nil {
		t.Fatal(err)
	}
	identity := [32]byte{1, 2, 3}
	if err := store.PutIdentity(ctx, "123:1", identity); err != nil {
		t.Fatal(err)
	}
	if err := store.PutSession(ctx, "123:1", []byte("session")); err != nil {
		t.Fatal(err)
	}
	if err := store.PutSenderKey(ctx, "group", "123", []byte("sender")); err != nil {
		t.Fatal(err)
	}
	preKeys, err := store.GetOrGenPreKeys(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.MarkPreKeysAsUploaded(ctx, preKeys[0].KeyID); err != nil {
		t.Fatal(err)
	}
	state, err := store.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadDeviceStore(state)
	if err != nil {
		t.Fatal(err)
	}
	trusted, err := loaded.IsTrustedIdentity(ctx, "123:1", identity)
	if err != nil || !trusted {
		t.Fatalf("identity did not round-trip: %v %v", trusted, err)
	}
	session, err := loaded.GetSession(ctx, "123:1")
	if err != nil || !bytes.Equal(session, []byte("session")) {
		t.Fatalf("session did not round-trip: %q %v", session, err)
	}
	sender, err := loaded.GetSenderKey(ctx, "group", "123")
	if err != nil || !bytes.Equal(sender, []byte("sender")) {
		t.Fatalf("sender key did not round-trip: %q %v", sender, err)
	}
	count, err := loaded.UploadedPreKeyCount(ctx)
	if err != nil || count != 1 {
		t.Fatalf("uploaded prekeys did not round-trip: %d %v", count, err)
	}
	if loaded.Device().FacebookUUID != store.Device().FacebookUUID || loaded.Device().RegistrationID != store.Device().RegistrationID {
		t.Fatal("device identity changed after round-trip")
	}
}

func TestDeviceStorePersistenceFailureRollsBack(t *testing.T) {
	ctx := context.Background()
	store, err := NewMemoryDeviceStore()
	if err != nil {
		t.Fatal(err)
	}
	persistErr := errors.New("persist failed")
	store.WithPersistence(func(context.Context, []byte) error { return persistErr })
	if err := store.PutSession(ctx, "123:1", []byte("new")); !errors.Is(err, persistErr) {
		t.Fatalf("expected persistence failure, got %v", err)
	}
	session, err := store.GetSession(ctx, "123:1")
	if err != nil {
		t.Fatal(err)
	}
	if len(session) != 0 {
		t.Fatalf("failed mutation was not rolled back: %q", session)
	}
}

func TestDeviceStorePersistenceReceivesCompleteSnapshots(t *testing.T) {
	ctx := context.Background()
	store, err := NewMemoryDeviceStore()
	if err != nil {
		t.Fatal(err)
	}
	var latest []byte
	store.WithPersistence(func(_ context.Context, data []byte) error {
		latest = append(latest[:0], data...)
		return nil
	})
	if err := store.PutSession(ctx, "1:1", []byte("a")); err != nil {
		t.Fatal(err)
	}
	if err := store.PutSession(ctx, "2:1", []byte("b")); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadDeviceStore(latest)
	if err != nil {
		t.Fatal(err)
	}
	for address, expected := range map[string]string{"1:1": "a", "2:1": "b"} {
		actual, err := loaded.GetSession(ctx, address)
		if err != nil || string(actual) != expected {
			t.Fatalf("snapshot missing %s: %q %v", address, actual, err)
		}
	}
}

func TestLegacyDeviceStateImport(t *testing.T) {
	store, err := NewMemoryDeviceStore()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	preKeys, err := store.GetOrGenPreKeys(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}
	account, err := proto.Marshal(store.Device().Account)
	if err != nil {
		t.Fatal(err)
	}
	legacy := legacyDeviceJSON{
		NoiseKeyPriv:     base64.StdEncoding.EncodeToString(store.Device().NoiseKey.Priv[:]),
		IdentityKeyPriv:  base64.StdEncoding.EncodeToString(store.Device().IdentityKey.Priv[:]),
		SignedPreKeyPriv: base64.StdEncoding.EncodeToString(store.Device().SignedPreKey.Priv[:]),
		SignedPreKeyID:   store.Device().SignedPreKey.KeyID,
		SignedPreKeySig:  base64.StdEncoding.EncodeToString(store.Device().SignedPreKey.Signature[:]),
		RegistrationID:   store.Device().RegistrationID,
		AdvSecretKey:     base64.StdEncoding.EncodeToString(store.Device().AdvSecretKey),
		FacebookUUID:     store.Device().FacebookUUID.String(),
		Account:          base64.StdEncoding.EncodeToString(account),
		PreKeys:          map[string]string{},
		Sessions:         map[string]string{"123:1": base64.StdEncoding.EncodeToString([]byte("session"))},
		Identities:       map[string]string{"123:1": base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32))},
		SenderKeys:       map[string]string{"g:u": base64.StdEncoding.EncodeToString([]byte("sender"))},
	}
	for _, preKey := range preKeys {
		legacy.PreKeys[string(rune('0'+preKey.KeyID))] = base64.StdEncoding.EncodeToString(preKey.Priv[:])
	}
	data, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadDeviceStore(data)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Device().RegistrationID != store.Device().RegistrationID || loaded.Device().FacebookUUID != store.Device().FacebookUUID {
		t.Fatal("legacy device identity changed during import")
	}
	if count, err := loaded.UploadedPreKeyCount(ctx); err != nil || count != 2 {
		t.Fatalf("legacy prekeys were not treated as uploaded: %d %v", count, err)
	}
	session, err := loaded.GetSession(ctx, "123:1")
	if err != nil || string(session) != "session" {
		t.Fatalf("legacy session did not import: %q %v", session, err)
	}
}
