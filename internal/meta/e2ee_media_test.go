package meta

import (
	"testing"

	"go.mau.fi/whatsmeow/proto/waMediaTransport"
	"go.mewis.me/fbgo/model"
)

func TestE2EEMediaMessageBuildsAllSupportedKinds(t *testing.T) {
	for _, kind := range []model.E2EEMediaKind{model.E2EEMediaImage, model.E2EEMediaVideo, model.E2EEMediaAudio, model.E2EEMediaDocument, model.E2EEMediaSticker} {
		t.Run(string(kind), func(t *testing.T) {
			transport := &waMediaTransport.WAMediaTransport{Integral: &waMediaTransport.WAMediaTransport_Integral{}, Ancillary: &waMediaTransport.WAMediaTransport_Ancillary{}}
			message, err := e2eeMediaMessage(model.E2EEMediaInput{Kind: kind, Name: "file.bin", Caption: "caption", Width: 32, Height: 24, Duration: 7, Voice: true}, transport)
			if err != nil {
				t.Fatal(err)
			}
			content := message.GetPayload().GetContent()
			if content == nil || content.GetContent() == nil {
				t.Fatalf("missing application content for %s", kind)
			}
			switch kind {
			case model.E2EEMediaImage:
				decoded, err := content.GetImageMessage().Decode()
				if err != nil || decoded.GetAncillary().GetWidth() != 32 || decoded.GetAncillary().GetHeight() != 24 || content.GetImageMessage().GetCaption().GetText() != "caption" {
					t.Fatalf("unexpected image payload: %#v %v", decoded, err)
				}
			case model.E2EEMediaVideo:
				decoded, err := content.GetVideoMessage().Decode()
				if err != nil || decoded.GetAncillary().GetSeconds() != 7 || content.GetVideoMessage().GetCaption().GetText() != "caption" {
					t.Fatalf("unexpected video payload: %#v %v", decoded, err)
				}
			case model.E2EEMediaAudio:
				decoded, err := content.GetAudioMessage().Decode()
				if err != nil || decoded.GetAncillary().GetSeconds() != 7 || !content.GetAudioMessage().GetPTT() {
					t.Fatalf("unexpected audio payload: %#v %v", decoded, err)
				}
			case model.E2EEMediaDocument:
				if _, err := content.GetDocumentMessage().Decode(); err != nil || content.GetDocumentMessage().GetFileName() != "file.bin" {
					t.Fatalf("unexpected document payload: %v", err)
				}
			case model.E2EEMediaSticker:
				decoded, err := content.GetStickerMessage().Decode()
				if err != nil || decoded.GetAncillary().GetWidth() != 32 || decoded.GetAncillary().GetHeight() != 24 {
					t.Fatalf("unexpected sticker payload: %#v %v", decoded, err)
				}
			}
		})
	}
}

func TestE2EEMediaTypeRejectsUnknownKind(t *testing.T) {
	if _, err := e2eeMediaType(model.E2EEMediaKind("unknown")); err == nil {
		t.Fatal("expected unsupported media kind error")
	}
}
