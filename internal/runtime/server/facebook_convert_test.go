package server

import (
	"errors"
	"testing"
	"time"

	fberrors "go.mewis.me/meta.go/errors"
	metav1 "go.mewis.me/meta.go/gen/go/meta/v1"
	"go.mewis.me/meta.go/model"
)

func TestFacebookConversions(t *testing.T) {
	owned, err := postOwnershipFromProto(metav1.FacebookPostOwnership_FACEBOOK_POST_OWNERSHIP_UNSPECIFIED)
	if err != nil || owned != model.PostOwned {
		t.Fatalf("default ownership: %q %v", owned, err)
	}
	shared, err := postOwnershipFromProto(metav1.FacebookPostOwnership_FACEBOOK_POST_OWNERSHIP_SHARED)
	if err != nil || shared != model.PostShared {
		t.Fatalf("shared ownership: %q %v", shared, err)
	}
	if _, err := postOwnershipFromProto(99); !errors.Is(err, fberrors.ErrInvalidInput) {
		t.Fatalf("expected invalid ownership, got %v", err)
	}
	input := marketplaceInputFromProto(&metav1.MarketplaceListingInput{Title: "item", PhotoIds: []string{"12345678901234567890"}, Location: &metav1.MarketplaceLocation{Latitude: 1, Longitude: 2, Name: "place"}})
	if len(input.PhotoIDs) != 1 || input.PhotoIDs[0] != "12345678901234567890" || input.Location.Name != "place" {
		t.Fatalf("marketplace input lost data: %#v", input)
	}
	now := time.Unix(1_700_000_000, 0).UTC()
	listing := marketplaceListingToProto(&model.MarketplaceListing{ID: "98765432109876543210", Seller: model.FacebookUser{ID: "12345678901234567890"}, CreatedAt: now})
	if listing.GetId() != "98765432109876543210" || listing.GetSeller().GetId() != "12345678901234567890" || !listing.GetCreatedAt().AsTime().Equal(now) {
		t.Fatalf("marketplace output lost data: %#v", listing)
	}
}
