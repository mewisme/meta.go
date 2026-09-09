// Package facebook exposes Facebook profile, search, post, social, notification, Marketplace and Professional Mode operations.
package facebook

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	fberrors "go.mewis.me/fbgo/errors"
	"go.mewis.me/fbgo/model"
)

var ErrUnavailable = errors.New("facebook service unavailable")

type Backend interface {
	GetFacebookUser(context.Context, model.ID) (*model.FacebookUser, error)
	SearchFacebook(context.Context, string, int) ([]model.SearchResult, error)
	ListNotifications(context.Context, int) ([]model.Notification, error)
	SetFacebookBio(context.Context, string, bool) error
	CreateAdditionalProfile(context.Context, string, string) error
	UnfriendFacebookUser(context.Context, model.ID) error
	SetFacebookBlocked(context.Context, model.ID, bool) error
	CreateFacebookPost(context.Context, string) (*model.Post, error)
	ArchiveFacebookPost(context.Context, model.ID, model.PostOwnership) error
	DeleteFacebookPost(context.Context, model.ID, model.PostOwnership) error
	CreateMarketplaceListing(context.Context, model.MarketplaceListingInput) (*model.MarketplaceListing, error)
	GetMarketplaceListing(context.Context, model.ID) (*model.MarketplaceListing, error)
	SetProfessionalMode(context.Context, bool) error
}

type Service struct{ backend Backend }

func NewService(backend Backend) *Service { return &Service{backend: backend} }

func (s *Service) User(ctx context.Context, userID model.ID) (*model.FacebookUser, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	if userID.Empty() {
		return nil, invalid("user ID is required")
	}
	return s.backend.GetFacebookUser(ctx, userID)
}

func (s *Service) Search(ctx context.Context, query string, limit int) ([]model.SearchResult, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, invalid("search query is required")
	}
	if limit <= 0 {
		limit = 5
	}
	if limit > 20 {
		return nil, invalid("search limit cannot exceed 20")
	}
	return s.backend.SearchFacebook(ctx, query, limit)
}

func (s *Service) Notifications(ctx context.Context, limit int) ([]model.Notification, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 15
	}
	if limit > 50 {
		return nil, invalid("notification limit cannot exceed 50")
	}
	return s.backend.ListNotifications(ctx, limit)
}

func (s *Service) SetBio(ctx context.Context, bio string, publishFeedStory bool) error {
	if err := s.ready(); err != nil {
		return err
	}
	if len([]rune(bio)) > 101 {
		return invalid("bio cannot exceed 101 characters")
	}
	return s.backend.SetFacebookBio(ctx, bio, publishFeedStory)
}

func (s *Service) CreateAdditionalProfile(ctx context.Context, name, username string) error {
	if err := s.ready(); err != nil {
		return err
	}
	name, username = strings.TrimSpace(name), strings.TrimSpace(username)
	if name == "" || username == "" {
		return invalid("additional profile name and username are required")
	}
	return s.backend.CreateAdditionalProfile(ctx, name, username)
}

func (s *Service) Unfriend(ctx context.Context, userID model.ID) error {
	if err := s.ready(); err != nil {
		return err
	}
	if userID.Empty() {
		return invalid("user ID is required")
	}
	return s.backend.UnfriendFacebookUser(ctx, userID)
}

func (s *Service) SetBlocked(ctx context.Context, userID model.ID, blocked bool) error {
	if err := s.ready(); err != nil {
		return err
	}
	if userID.Empty() {
		return invalid("user ID is required")
	}
	return s.backend.SetFacebookBlocked(ctx, userID, blocked)
}

func (s *Service) CreatePost(ctx context.Context, text string) (*model.Post, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, invalid("post text is required")
	}
	return s.backend.CreateFacebookPost(ctx, text)
}

func (s *Service) ArchivePost(ctx context.Context, postID model.ID, ownership model.PostOwnership) error {
	if err := s.ready(); err != nil {
		return err
	}
	if err := validatePost(postID, ownership); err != nil {
		return err
	}
	return s.backend.ArchiveFacebookPost(ctx, postID, ownership)
}

func (s *Service) DeletePost(ctx context.Context, postID model.ID, ownership model.PostOwnership) error {
	if err := s.ready(); err != nil {
		return err
	}
	if err := validatePost(postID, ownership); err != nil {
		return err
	}
	return s.backend.DeleteFacebookPost(ctx, postID, ownership)
}

func validatePost(postID model.ID, ownership model.PostOwnership) error {
	if postID.Empty() {
		return invalid("post ID is required")
	}
	if ownership == "" {
		ownership = model.PostOwned
	}
	if ownership != model.PostOwned && ownership != model.PostShared {
		return invalid("post ownership must be owned or shared")
	}
	return nil
}

func (s *Service) CreateMarketplaceListing(ctx context.Context, input model.MarketplaceListingInput) (*model.MarketplaceListing, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	input.Title, input.Brand, input.Price, input.Currency, input.Category = strings.TrimSpace(input.Title), strings.TrimSpace(input.Brand), strings.TrimSpace(input.Price), strings.ToUpper(strings.TrimSpace(input.Currency)), strings.TrimSpace(input.Category)
	if input.Title == "" || input.Price == "" || input.Currency == "" || input.Category == "" {
		return nil, invalid("Marketplace title, price, currency and category are required")
	}
	price, err := strconv.ParseFloat(input.Price, 64)
	if err != nil || price < 0 {
		return nil, invalid("Marketplace price must be a non-negative number")
	}
	if len(input.PhotoIDs) == 0 {
		return nil, invalid("Marketplace requires at least one photo ID")
	}
	if input.Location.Latitude < -90 || input.Location.Latitude > 90 || input.Location.Longitude < -180 || input.Location.Longitude > 180 {
		return nil, invalid("Marketplace seller coordinates are invalid")
	}
	if _, ok := marketplaceCategoryIDs[input.Category]; !ok {
		return nil, invalid("unsupported Marketplace category")
	}
	return s.backend.CreateMarketplaceListing(ctx, input)
}

func (s *Service) MarketplaceListing(ctx context.Context, listingID model.ID) (*model.MarketplaceListing, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	if listingID.Empty() {
		return nil, invalid("Marketplace listing ID is required")
	}
	return s.backend.GetMarketplaceListing(ctx, listingID)
}

func (s *Service) SetProfessionalMode(ctx context.Context, enabled bool) error {
	if err := s.ready(); err != nil {
		return err
	}
	return s.backend.SetProfessionalMode(ctx, enabled)
}

func MarketplaceCategories() []string {
	result := make([]string, 0, len(marketplaceCategoryIDs))
	for name := range marketplaceCategoryIDs {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

var marketplaceCategoryIDs = map[string]int64{
	"Tools": 2171028376552553, "Furniture": 1583634935226685, "Household": 1569171756675761, "Garden": 800089866739547, "Appliances": 678754142233400,
	"Video Games": 686977074745292, "Books Movies&Music": 613858625416355, "Bags & Luggage": 1567543000236608, "Women's clothing & shoes": 1266429133383966,
	"Men's clothing & shoes": 931157863635831, "Jewelry & Accessories": 214968118845643, "Health & beauty": 1555452698044988, "Pet Supplies": 1550246318620997,
	"Baby & kids": 624859874282116, "Toys & Games": 606456512821491, "Electronics & computers": 1792291877663080, "Mobile phones": 1557869527812749,
	"Bicycles": 1658310421102081, "Arts & Crafts": 1534799543476160, "Sports & Outdoors": 1383948661922113, "Auto parts": 757715671026531,
	"Musical Instruments": 676772489112490, "Antiques & Collectibles": 393860164117441, "Garage Sale": 1834536343472201, "Miscellaneous": 895487550471874,
}

func (s *Service) ready() error {
	if s == nil || s.backend == nil {
		return ErrUnavailable
	}
	return nil
}

func invalid(message string) error { return fmt.Errorf("%w: %s", fberrors.ErrInvalidInput, message) }
