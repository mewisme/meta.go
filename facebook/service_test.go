package facebook

import (
	"context"
	"errors"
	"testing"

	fberrors "go.mewis.me/fbgo/errors"
	"go.mewis.me/fbgo/model"
)

type fakeBackend struct{}

func (*fakeBackend) GetFacebookUser(context.Context, model.ID) (*model.FacebookUser, error) {
	return &model.FacebookUser{ID: "1"}, nil
}
func (*fakeBackend) SearchFacebook(context.Context, string, int) ([]model.SearchResult, error) {
	return []model.SearchResult{{ID: "1"}}, nil
}
func (*fakeBackend) ListNotifications(context.Context, int) ([]model.Notification, error) {
	return []model.Notification{{Text: "x"}}, nil
}
func (*fakeBackend) SetFacebookBio(context.Context, string, bool) error            { return nil }
func (*fakeBackend) CreateAdditionalProfile(context.Context, string, string) error { return nil }
func (*fakeBackend) UnfriendFacebookUser(context.Context, model.ID) error          { return nil }
func (*fakeBackend) SetFacebookBlocked(context.Context, model.ID, bool) error      { return nil }
func (*fakeBackend) CreateFacebookPost(context.Context, string) (*model.Post, error) {
	return &model.Post{URL: "https://example.invalid/post"}, nil
}
func (*fakeBackend) ArchiveFacebookPost(context.Context, model.ID, model.PostOwnership) error {
	return nil
}
func (*fakeBackend) DeleteFacebookPost(context.Context, model.ID, model.PostOwnership) error {
	return nil
}
func (*fakeBackend) CreateMarketplaceListing(context.Context, model.MarketplaceListingInput) (*model.MarketplaceListing, error) {
	return &model.MarketplaceListing{ID: "m1"}, nil
}
func (*fakeBackend) GetMarketplaceListing(context.Context, model.ID) (*model.MarketplaceListing, error) {
	return &model.MarketplaceListing{ID: "m1"}, nil
}
func (*fakeBackend) SetProfessionalMode(context.Context, bool) error { return nil }

func TestServiceReadOperations(t *testing.T) {
	service := NewService(new(fakeBackend))
	if user, err := service.User(context.Background(), "1"); err != nil || user.ID != "1" {
		t.Fatalf("unexpected user: %#v %v", user, err)
	}
	if _, err := service.Search(context.Background(), "", 5); !errors.Is(err, fberrors.ErrInvalidInput) {
		t.Fatalf("expected invalid search query, got %v", err)
	}
	if _, err := service.Search(context.Background(), "test", 21); !errors.Is(err, fberrors.ErrInvalidInput) {
		t.Fatalf("expected invalid search limit, got %v", err)
	}
	if _, err := service.Notifications(context.Background(), 51); !errors.Is(err, fberrors.ErrInvalidInput) {
		t.Fatalf("expected invalid notification limit, got %v", err)
	}
	if err := service.SetBio(context.Background(), string(make([]rune, 102)), false); !errors.Is(err, fberrors.ErrInvalidInput) {
		t.Fatalf("expected invalid bio, got %v", err)
	}
	if err := service.CreateAdditionalProfile(context.Background(), "", "user"); !errors.Is(err, fberrors.ErrInvalidInput) {
		t.Fatalf("expected invalid profile, got %v", err)
	}
	if err := service.SetBlocked(context.Background(), "", true); !errors.Is(err, fberrors.ErrInvalidInput) {
		t.Fatalf("expected invalid block target, got %v", err)
	}
	if _, err := service.CreatePost(context.Background(), " "); !errors.Is(err, fberrors.ErrInvalidInput) {
		t.Fatalf("expected invalid post, got %v", err)
	}
	if _, err := service.CreateMarketplaceListing(context.Background(), model.MarketplaceListingInput{}); !errors.Is(err, fberrors.ErrInvalidInput) {
		t.Fatalf("expected invalid Marketplace input, got %v", err)
	}
	if categories := MarketplaceCategories(); len(categories) != 25 || categories[0] == "" {
		t.Fatalf("unexpected Marketplace categories: %d", len(categories))
	}
}
