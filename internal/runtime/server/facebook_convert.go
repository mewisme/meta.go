package server

import (
	"fmt"

	fberrors "go.mewis.me/meta.go/errors"
	metav1 "go.mewis.me/meta.go/gen/go/meta/v1"
	"go.mewis.me/meta.go/model"
)

func facebookUserToProto(user *model.FacebookUser) *metav1.FacebookUser {
	if user == nil {
		return nil
	}
	return &metav1.FacebookUser{Id: user.ID.String(), Name: user.Name, FirstName: user.FirstName, Username: user.Username, ProfileUrl: user.ProfileURL, AvatarUrl: user.AvatarURL, Gender: user.Gender, AlternateName: user.AlternateName, NonFriend: user.NonFriend}
}

func facebookSearchResultToProto(result model.SearchResult) *metav1.FacebookSearchResult {
	return &metav1.FacebookSearchResult{Id: result.ID.String(), Name: result.Name, Url: result.URL}
}

func facebookNotificationToProto(notification model.Notification) *metav1.FacebookNotification {
	return &metav1.FacebookNotification{Id: notification.ID.String(), Text: notification.Text, Url: notification.URL, Timestamp: timestampOrNil(notification.Timestamp)}
}

func facebookPostToProto(post *model.Post) *metav1.FacebookPost {
	if post == nil {
		return nil
	}
	return &metav1.FacebookPost{Id: post.ID.String(), Url: post.URL}
}

func postOwnershipFromProto(value metav1.FacebookPostOwnership) (model.PostOwnership, error) {
	switch value {
	case metav1.FacebookPostOwnership_FACEBOOK_POST_OWNERSHIP_UNSPECIFIED, metav1.FacebookPostOwnership_FACEBOOK_POST_OWNERSHIP_OWNED:
		return model.PostOwned, nil
	case metav1.FacebookPostOwnership_FACEBOOK_POST_OWNERSHIP_SHARED:
		return model.PostShared, nil
	default:
		return "", fmt.Errorf("%w: unsupported post ownership", fberrors.ErrInvalidInput)
	}
}

func marketplaceInputFromProto(input *metav1.MarketplaceListingInput) model.MarketplaceListingInput {
	if input == nil {
		return model.MarketplaceListingInput{}
	}
	result := model.MarketplaceListingInput{Title: input.GetTitle(), Brand: input.GetBrand(), Price: input.GetPrice(), Currency: input.GetCurrency(), Description: input.GetDescription(), Hashtags: append([]string(nil), input.GetHashtags()...), Category: input.GetCategory(), PhotoIDs: idsFromStrings(input.GetPhotoIds())}
	if location := input.GetLocation(); location != nil {
		result.Location = model.MarketplaceLocation{Latitude: location.GetLatitude(), Longitude: location.GetLongitude(), Name: location.GetName()}
	}
	return result
}

func marketplaceListingToProto(listing *model.MarketplaceListing) *metav1.MarketplaceListing {
	if listing == nil {
		return nil
	}
	return &metav1.MarketplaceListing{Id: listing.ID.String(), Title: listing.Title, Description: listing.Description, Price: listing.Price, Currency: listing.Currency, Seller: facebookUserToProto(&listing.Seller), Location: &metav1.MarketplaceLocation{Latitude: listing.Location.Latitude, Longitude: listing.Location.Longitude, Name: listing.Location.Name}, Url: listing.URL, CreatedAt: timestampOrNil(listing.CreatedAt)}
}
