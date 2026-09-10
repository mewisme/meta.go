package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"slices"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	metav1 "go.mewis.me/meta.go/gen/go/meta/v1"
	"go.mewis.me/meta.go/internal/runtime/session"
	"go.mewis.me/meta.go/model"
)

type phase6Messenger struct {
	session.Messenger
	upload model.UploadInput
	data   []byte
	send   model.SendRequest
}

func (f *phase6Messenger) Upload(_ context.Context, input model.UploadInput) (model.UploadResult, error) {
	data, err := io.ReadAll(input.Reader)
	if err != nil {
		return model.UploadResult{}, err
	}
	f.upload = input
	f.upload.Reader = nil
	f.data = data
	return model.UploadResult{ID: "a1", Name: input.Name, ContentType: input.ContentType, Type: "file"}, nil
}

func (f *phase6Messenger) Send(_ context.Context, req model.SendRequest) (model.SendResult, error) {
	f.send = req
	return model.SendResult{MessageID: "m1"}, nil
}

type phase6Threads struct {
	session.Threads
	threadID model.ID
	photo    model.AttachmentInput
	data     []byte
}

func (f *phase6Threads) SetPhoto(_ context.Context, threadID model.ID, input model.AttachmentInput) error {
	data, err := io.ReadAll(input.Reader)
	if err != nil {
		return err
	}
	f.threadID = threadID
	f.photo = input
	f.photo.Reader = nil
	f.data = data
	return nil
}

type phase6E2EE struct {
	session.E2EE
	media     model.E2EEMediaInput
	mediaData []byte
	download  model.E2EEMediaReference
}

func (f *phase6E2EE) SendE2EEMedia(_ context.Context, input model.E2EEMediaInput) (model.SendResult, error) {
	data, err := io.ReadAll(input.Reader)
	if err != nil {
		return model.SendResult{}, err
	}
	f.media = input
	f.media.Reader = nil
	f.mediaData = data
	return model.SendResult{MessageID: "e1", Timestamp: time.Unix(1_700_000_000, 0).UTC()}, nil
}

func (f *phase6E2EE) DownloadE2EE(_ context.Context, input model.E2EEMediaDownload) ([]byte, error) {
	f.download = input.Reference
	return bytes.Repeat([]byte("z"), maxMediaChunkBytes+123), nil
}

func TestPhase6StreamingRPCContract(t *testing.T) {
	messenger := &phase6Messenger{}
	threads := &phase6Threads{}
	e2ee := &phase6E2EE{}
	fake := &runtimeFakeClient{messenger: messenger, threads: threads, e2ee: e2ee}
	manager := session.NewManager(func(session.Config) (session.Client, error) { return fake, nil })
	created, err := manager.Create(session.Config{})
	if err != nil {
		t.Fatal(err)
	}
	listener, err := Listen("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	runtimeServer, err := New(Config{Token: "test-token", Sessions: manager})
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = runtimeServer.Serve(listener) }()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := runtimeServer.Shutdown(ctx); err != nil && !errors.Is(err, context.Canceled) {
			t.Errorf("shutdown: %v", err)
		}
	})
	conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	ctx := metadata.AppendToOutgoingContext(t.Context(), authorizationKey, "Bearer test-token")
	messengerClient := metav1.NewMessengerServiceClient(conn)

	uploadData := []byte("regular-upload-data")
	uploadHash := sha256.Sum256(uploadData)
	upload, err := messengerClient.Upload(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := upload.Send(&metav1.UploadRequest{Payload: &metav1.UploadRequest_Metadata{Metadata: &metav1.UploadMetadata{SessionId: created.ID(), ThreadId: "t1", Name: "file.bin", ContentType: "application/octet-stream", Size: int64(len(uploadData)), Voice: true, Sha256: uploadHash[:]}}}); err != nil {
		t.Fatal(err)
	}
	if err := upload.Send(&metav1.UploadRequest{Payload: &metav1.UploadRequest_Chunk{Chunk: &metav1.MediaChunk{Data: uploadData[:7]}}}); err != nil {
		t.Fatal(err)
	}
	if err := upload.Send(&metav1.UploadRequest{Payload: &metav1.UploadRequest_Chunk{Chunk: &metav1.MediaChunk{Data: uploadData[7:]}}}); err != nil {
		t.Fatal(err)
	}
	uploaded, err := upload.CloseAndRecv()
	if err != nil || uploaded.GetId() != "a1" || !bytes.Equal(uploaded.GetSha256(), uploadHash[:]) || messenger.upload.ThreadID != "t1" || !messenger.upload.Voice || !bytes.Equal(messenger.data, uploadData) {
		t.Fatalf("regular upload mismatch: response=%#v input=%#v data=%q err=%v", uploaded, messenger.upload, messenger.data, err)
	}

	sent, err := messengerClient.Send(ctx, &metav1.SendRequest{SessionId: created.ID(), ThreadId: "t1", AttachmentIds: []string{uploaded.GetId()}})
	if err != nil || sent.GetMessageId() != "m1" || !slices.Equal(messenger.send.AttachmentIDs, []model.ID{"a1"}) {
		t.Fatalf("uploaded attachment send mismatch: response=%#v request=%#v err=%v", sent, messenger.send, err)
	}

	photoData := []byte("thread-photo")
	photoHash := sha256.Sum256(photoData)
	photo, err := messengerClient.SetThreadPhoto(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := photo.Send(&metav1.SetThreadPhotoRequest{Payload: &metav1.SetThreadPhotoRequest_Metadata{Metadata: &metav1.SetThreadPhotoMetadata{SessionId: created.ID(), ThreadId: "t1", Name: "photo.jpg", ContentType: "image/jpeg", Size: int64(len(photoData)), Sha256: photoHash[:]}}}); err != nil {
		t.Fatal(err)
	}
	if err := photo.Send(&metav1.SetThreadPhotoRequest{Payload: &metav1.SetThreadPhotoRequest_Chunk{Chunk: &metav1.MediaChunk{Data: photoData}}}); err != nil {
		t.Fatal(err)
	}
	photoResult, err := photo.CloseAndRecv()
	if err != nil || !bytes.Equal(photoResult.GetSha256(), photoHash[:]) || threads.threadID != "t1" || threads.photo.Name != "photo.jpg" || !bytes.Equal(threads.data, photoData) {
		t.Fatalf("thread photo mismatch: response=%#v input=%#v data=%q err=%v", photoResult, threads.photo, threads.data, err)
	}

	e2eeClient := metav1.NewE2EEServiceClient(conn)
	mediaData := []byte("encrypted-media-input")
	mediaHash := sha256.Sum256(mediaData)
	media, err := e2eeClient.SendMedia(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := media.Send(&metav1.E2EEServiceSendMediaRequest{Payload: &metav1.E2EEServiceSendMediaRequest_Metadata{Metadata: &metav1.E2EEServiceSendMediaMetadata{SessionId: created.ID(), ChatJid: "chat", Kind: metav1.E2EEMediaKind_E2EE_MEDIA_KIND_DOCUMENT, Name: "doc.bin", ContentType: "application/octet-stream", Size: int64(len(mediaData)), Caption: "caption", Width: 10, Height: 20, Duration: 30, Sha256: mediaHash[:]}}}); err != nil {
		t.Fatal(err)
	}
	if err := media.Send(&metav1.E2EEServiceSendMediaRequest{Payload: &metav1.E2EEServiceSendMediaRequest_Chunk{Chunk: &metav1.MediaChunk{Data: mediaData}}}); err != nil {
		t.Fatal(err)
	}
	mediaResult, err := media.CloseAndRecv()
	if err != nil || mediaResult.GetMessageId() != "e1" || !bytes.Equal(mediaResult.GetSha256(), mediaHash[:]) || e2ee.media.Kind != model.E2EEMediaDocument || e2ee.media.Caption != "caption" || !bytes.Equal(e2ee.mediaData, mediaData) {
		t.Fatalf("E2EE media send mismatch: response=%#v input=%#v data=%q err=%v", mediaResult, e2ee.media, e2ee.mediaData, err)
	}

	reference := &metav1.E2EEMediaReference{Kind: string(model.E2EEMediaDocument), DirectPath: "/media", MediaKey: []byte{1}, FileSha256: []byte{2}, ContentType: "application/octet-stream", Size: maxMediaChunkBytes + 123}
	download, err := e2eeClient.DownloadMedia(ctx, &metav1.E2EEServiceDownloadMediaRequest{SessionId: created.ID(), Reference: reference})
	if err != nil {
		t.Fatal(err)
	}
	first, err := download.Recv()
	if err != nil || first.GetMetadata() == nil || first.GetMetadata().GetSize() != maxMediaChunkBytes+123 {
		t.Fatalf("download metadata mismatch: %#v err=%v", first, err)
	}
	var downloaded []byte
	for {
		frame, err := download.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		chunk := frame.GetChunk().GetData()
		if len(chunk) > maxMediaChunkBytes {
			t.Fatalf("download chunk too large: %d", len(chunk))
		}
		downloaded = append(downloaded, chunk...)
	}
	expectedDownload := bytes.Repeat([]byte("z"), maxMediaChunkBytes+123)
	expectedDownloadHash := sha256.Sum256(expectedDownload)
	if !bytes.Equal(downloaded, expectedDownload) || !bytes.Equal(first.GetMetadata().GetSha256(), expectedDownloadHash[:]) || e2ee.download.DirectPath != "/media" || e2ee.download.Kind != model.E2EEMediaDocument {
		t.Fatalf("E2EE download mismatch: metadata=%#v reference=%#v len=%d", first.GetMetadata(), e2ee.download, len(downloaded))
	}
}

func TestPhase6StreamSurfacesComplete(t *testing.T) {
	messengerExpected := []string{"Upload", "SetThreadPhoto"}
	messengerActual := make([]string, 0, len(metav1.MessengerService_ServiceDesc.Streams))
	for _, stream := range metav1.MessengerService_ServiceDesc.Streams {
		messengerActual = append(messengerActual, stream.StreamName)
	}
	if !slices.Equal(messengerActual, messengerExpected) {
		t.Fatalf("Messenger stream surface mismatch: actual=%v expected=%v", messengerActual, messengerExpected)
	}
	e2eeExpected := []string{"SendMedia", "DownloadMedia"}
	e2eeActual := make([]string, 0, len(metav1.E2EEService_ServiceDesc.Streams))
	for _, stream := range metav1.E2EEService_ServiceDesc.Streams {
		e2eeActual = append(e2eeActual, stream.StreamName)
	}
	if !slices.Equal(e2eeActual, e2eeExpected) {
		t.Fatalf("E2EE stream surface mismatch: actual=%v expected=%v", e2eeActual, e2eeExpected)
	}
}
