package server

import (
	"context"

	fberrors "go.mewis.me/meta.go/errors"
	metav1 "go.mewis.me/meta.go/gen/go/meta/v1"
	"go.mewis.me/meta.go/internal/runtime/session"
	"go.mewis.me/meta.go/model"
)

type facebookService struct {
	metav1.UnimplementedFacebookServiceServer
	server *Server
}

func (s *facebookService) service(sessionID string) (session.Facebook, error) {
	sess, err := (&messengerService{server: s.server}).runtimeSession(sessionID)
	if err != nil {
		return nil, err
	}
	service, err := sess.FacebookService()
	if err != nil {
		return nil, err
	}
	if service == nil {
		return nil, fberrors.ErrNotConnected
	}
	return service, nil
}

func (s *facebookService) GetUser(ctx context.Context, req *metav1.FacebookServiceGetUserRequest) (*metav1.FacebookServiceGetUserResponse, error) {
	service, err := s.service(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	user, err := service.User(ctx, model.ID(req.GetUserId()))
	if err != nil {
		return nil, grpcError(err)
	}
	return &metav1.FacebookServiceGetUserResponse{User: facebookUserToProto(user)}, nil
}

func (s *facebookService) Search(ctx context.Context, req *metav1.FacebookServiceSearchRequest) (*metav1.FacebookServiceSearchResponse, error) {
	service, err := s.service(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	items, err := service.Search(ctx, req.GetQuery(), int(req.GetLimit()))
	if err != nil {
		return nil, grpcError(err)
	}
	result := &metav1.FacebookServiceSearchResponse{Results: make([]*metav1.FacebookSearchResult, 0, len(items))}
	for _, item := range items {
		result.Results = append(result.Results, facebookSearchResultToProto(item))
	}
	return result, nil
}

func (s *facebookService) ListNotifications(ctx context.Context, req *metav1.FacebookServiceListNotificationsRequest) (*metav1.FacebookServiceListNotificationsResponse, error) {
	service, err := s.service(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	items, err := service.Notifications(ctx, int(req.GetLimit()))
	if err != nil {
		return nil, grpcError(err)
	}
	result := &metav1.FacebookServiceListNotificationsResponse{Notifications: make([]*metav1.FacebookNotification, 0, len(items))}
	for _, item := range items {
		result.Notifications = append(result.Notifications, facebookNotificationToProto(item))
	}
	return result, nil
}

func (s *facebookService) SetBio(ctx context.Context, req *metav1.FacebookServiceSetBioRequest) (*metav1.FacebookServiceSetBioResponse, error) {
	service, err := s.service(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	if err := service.SetBio(ctx, req.GetBio(), req.GetPublishFeedStory()); err != nil {
		return nil, grpcError(err)
	}
	return &metav1.FacebookServiceSetBioResponse{}, nil
}

func (s *facebookService) CreateAdditionalProfile(ctx context.Context, req *metav1.FacebookServiceCreateAdditionalProfileRequest) (*metav1.FacebookServiceCreateAdditionalProfileResponse, error) {
	service, err := s.service(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	if err := service.CreateAdditionalProfile(ctx, req.GetName(), req.GetUsername()); err != nil {
		return nil, grpcError(err)
	}
	return &metav1.FacebookServiceCreateAdditionalProfileResponse{}, nil
}

func (s *facebookService) Unfriend(ctx context.Context, req *metav1.FacebookServiceUnfriendRequest) (*metav1.FacebookServiceUnfriendResponse, error) {
	service, err := s.service(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	if err := service.Unfriend(ctx, model.ID(req.GetUserId())); err != nil {
		return nil, grpcError(err)
	}
	return &metav1.FacebookServiceUnfriendResponse{}, nil
}

func (s *facebookService) SetBlocked(ctx context.Context, req *metav1.FacebookServiceSetBlockedRequest) (*metav1.FacebookServiceSetBlockedResponse, error) {
	service, err := s.service(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	if err := service.SetBlocked(ctx, model.ID(req.GetUserId()), req.GetBlocked()); err != nil {
		return nil, grpcError(err)
	}
	return &metav1.FacebookServiceSetBlockedResponse{}, nil
}

func (s *facebookService) CreatePost(ctx context.Context, req *metav1.FacebookServiceCreatePostRequest) (*metav1.FacebookServiceCreatePostResponse, error) {
	service, err := s.service(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	post, err := service.CreatePost(ctx, req.GetText())
	if err != nil {
		return nil, grpcError(err)
	}
	return &metav1.FacebookServiceCreatePostResponse{Post: facebookPostToProto(post)}, nil
}

func (s *facebookService) ArchivePost(ctx context.Context, req *metav1.FacebookServiceArchivePostRequest) (*metav1.FacebookServiceArchivePostResponse, error) {
	service, err := s.service(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	ownership, err := postOwnershipFromProto(req.GetOwnership())
	if err != nil {
		return nil, grpcError(err)
	}
	if err := service.ArchivePost(ctx, model.ID(req.GetPostId()), ownership); err != nil {
		return nil, grpcError(err)
	}
	return &metav1.FacebookServiceArchivePostResponse{}, nil
}

func (s *facebookService) DeletePost(ctx context.Context, req *metav1.FacebookServiceDeletePostRequest) (*metav1.FacebookServiceDeletePostResponse, error) {
	service, err := s.service(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	ownership, err := postOwnershipFromProto(req.GetOwnership())
	if err != nil {
		return nil, grpcError(err)
	}
	if err := service.DeletePost(ctx, model.ID(req.GetPostId()), ownership); err != nil {
		return nil, grpcError(err)
	}
	return &metav1.FacebookServiceDeletePostResponse{}, nil
}

func (s *facebookService) CreateMarketplaceListing(ctx context.Context, req *metav1.FacebookServiceCreateMarketplaceListingRequest) (*metav1.FacebookServiceCreateMarketplaceListingResponse, error) {
	service, err := s.service(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	listing, err := service.CreateMarketplaceListing(ctx, marketplaceInputFromProto(req.GetListing()))
	if err != nil {
		return nil, grpcError(err)
	}
	return &metav1.FacebookServiceCreateMarketplaceListingResponse{Listing: marketplaceListingToProto(listing)}, nil
}

func (s *facebookService) GetMarketplaceListing(ctx context.Context, req *metav1.FacebookServiceGetMarketplaceListingRequest) (*metav1.FacebookServiceGetMarketplaceListingResponse, error) {
	service, err := s.service(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	listing, err := service.MarketplaceListing(ctx, model.ID(req.GetListingId()))
	if err != nil {
		return nil, grpcError(err)
	}
	return &metav1.FacebookServiceGetMarketplaceListingResponse{Listing: marketplaceListingToProto(listing)}, nil
}

func (s *facebookService) SetProfessionalMode(ctx context.Context, req *metav1.FacebookServiceSetProfessionalModeRequest) (*metav1.FacebookServiceSetProfessionalModeResponse, error) {
	service, err := s.service(req.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	if err := service.SetProfessionalMode(ctx, req.GetEnabled()); err != nil {
		return nil, grpcError(err)
	}
	return &metav1.FacebookServiceSetProfessionalModeResponse{}, nil
}
