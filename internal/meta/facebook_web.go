package meta

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go.mewis.me/fbgo/internal/protocol"
	"go.mewis.me/fbgo/internal/webapi"
	"go.mewis.me/fbgo/model"
	metaTypes "go.mewis.me/meta-extra/pkg/messagix/types"
)

func (b *messagixBackend) GetFacebookUser(ctx context.Context, userID model.ID) (*model.FacebookUser, error) {
	form, _, err := b.baseForm(ctx)
	if err != nil {
		return nil, err
	}
	form.Set("ids[0]", userID.String())
	result, err := b.postFacebookForm(ctx, protocol.UserInfoURL, form)
	if err != nil {
		return nil, err
	}
	profile := mapAt(mapAt(result, "payload"), "profiles", userID.String())
	if len(profile) == 0 {
		return nil, fmt.Errorf("Facebook user %s not found", userID)
	}
	gender := "unknown"
	switch int64Value(profile["gender"]) {
	case 1:
		gender = "female"
	case 2:
		gender = "male"
	}
	return &model.FacebookUser{ID: model.ID(firstString(profile, "id", "uid")), Name: stringValue(profile["name"]), FirstName: firstString(profile, "firstName", "first_name"), Username: firstString(profile, "vanity", "username"), ProfileURL: firstString(profile, "uri", "url"), AvatarURL: firstString(profile, "thumbSrc", "thumb_src", "thumnSrc"), Gender: gender, AlternateName: firstString(profile, "alternateName", "alternate_name"), NonFriend: boolValue(profile["is_nonfriend_messenger_contact"])}, nil
}

func cloneStringAnyMap(input map[string]any) map[string]any {
	result := make(map[string]any, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func (b *messagixBackend) SearchFacebook(ctx context.Context, query string, limit int) ([]model.SearchResult, error) {
	if limit <= 0 {
		limit = 5
	}
	sessionID, err := webapi.NewSessionID()
	if err != nil {
		return nil, err
	}
	query = strings.TrimSpace(query)
	baseArgs := map[string]any{
		"callsite": "COMET_GLOBAL_SEARCH",
		"config":   map[string]any{"exact_match": false, "high_confidence_config": nil, "intercept_config": nil, "sts_disambiguation": nil, "watch_config": nil},
		"filters":  []any{}, "text": strings.ToLower(query),
	}
	primaryArgs := cloneStringAnyMap(baseArgs)
	primaryArgs["context"] = map[string]any{"bsid": strconv.FormatUint(sessionID, 10), "tsid": nil}
	primaryArgs["experience"] = map[string]any{"encoded_server_defined_params": nil, "fbid": nil, "type": "PEOPLE_TAB"}
	legacyArgs := cloneStringAnyMap(baseArgs)
	legacyArgs["context"] = map[string]any{"bsid": strconv.FormatUint(sessionID, 10), "tsid": fmt.Sprintf("0.%016d", sessionID)}
	legacyArgs["experience"] = map[string]any{"encoded_server_defined_params": nil, "fbid": nil, "type": "GLOBAL_SEARCH"}
	primaryVariables := map[string]any{"count": limit, "allow_streaming": false, "args": primaryArgs, "cursor": nil, "feedbackSource": 23, "fetch_filters": true, "renderLocation": "search_results_page", "scale": 1, "stream_initial_count": 0, "useDefaultActor": false, "__relay_internal__pv__IsWorkUserrelayprovider": false, "__relay_internal__pv__IsMergQAPollsrelayprovider": false, "__relay_internal__pv__StoriesArmadilloReplyEnabledrelayprovider": false, "__relay_internal__pv__StoriesRingrelayprovider": false}
	legacyVariables := map[string]any{"count": limit, "allow_streaming": false, "args": legacyArgs, "cursor": nil, "feedbackSource": 23, "fetch_filters": true, "renderLocation": "search_results_page", "scale": 3, "stream_initial_count": 0, "useDefaultActor": false, "__relay_internal__pv__SearchCometResultsShowUserAvailabilityrelayprovider": true, "__relay_internal__pv__IsWorkUserrelayprovider": false, "__relay_internal__pv__IsMergQAPollsrelayprovider": false, "__relay_internal__pv__StoriesArmadilloReplyEnabledrelayprovider": false, "__relay_internal__pv__StoriesRingrelayprovider": false}
	attempts := []struct {
		docID     string
		variables map[string]any
	}{{protocol.SearchDocID, primaryVariables}, {protocol.SearchLegacyDocID, legacyVariables}}
	var result map[string]any
	var searchErr error
	for _, attempt := range attempts {
		result, searchErr = b.graphQL(ctx, protocol.SearchFriendlyName, attempt.docID, attempt.variables)
		if searchErr == nil {
			break
		}
	}
	if searchErr != nil {
		return nil, fmt.Errorf("Facebook search persisted queries failed: %w", searchErr)
	}
	edges, _ := mapAt(result, "data", "serpResponse", "results")["edges"].([]any)
	return parseSearchResults(edges, limit), nil
}

func parseSearchResults(edges []any, limit int) []model.SearchResult {
	results := make([]model.SearchResult, 0, min(limit, len(edges)))
	seen := make(map[model.ID]struct{})
	appendProfile := func(profile map[string]any) bool {
		id := model.ID(stringValue(profile["id"]))
		if id.Empty() {
			return false
		}
		if _, exists := seen[id]; exists {
			return false
		}
		seen[id] = struct{}{}
		results = append(results, model.SearchResult{ID: id, Name: stringValue(profile["name"]), URL: stringValue(profile["url"])})
		return len(results) >= limit
	}
	for _, rawEdge := range edges {
		edge, ok := rawEdge.(map[string]any)
		if !ok {
			continue
		}
		strategy := mapAt(edge, "relay_rendering_strategy")
		if profile := mapAt(strategy, "view_model", "profile"); len(profile) > 0 && appendProfile(profile) {
			break
		}
		strategies, _ := strategy["result_rendering_strategies"].([]any)
		for _, rawStrategy := range strategies {
			item, ok := rawStrategy.(map[string]any)
			if !ok {
				continue
			}
			if profile := mapAt(item, "view_model", "profile"); len(profile) > 0 && appendProfile(profile) {
				return results
			}
		}
	}
	return results
}

func (b *messagixBackend) ListNotifications(ctx context.Context, limit int) ([]model.Notification, error) {
	if limit <= 0 {
		limit = 15
	}
	result, err := b.graphQL(ctx, protocol.NotificationsFriendlyName, protocol.NotificationsDocID, map[string]any{"count": limit, "environment": "MAIN_SURFACE", "scale": 3})
	if err != nil {
		return nil, err
	}
	edges, ok := mapAt(result, "data", "viewer", "notifications_page")["edges"].([]any)
	if !ok {
		return nil, errors.New("notification response did not contain edges")
	}
	items := make([]model.Notification, 0, min(limit, len(edges)))
	for _, raw := range edges {
		edge, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		notif := mapAt(edge, "node", "notif")
		text := stringValue(mapAt(notif, "body")["text"])
		if text == "" {
			continue
		}
		items = append(items, model.Notification{ID: model.ID(firstString(notif, "id", "notif_id")), Text: text, URL: firstString(notif, "url", "href"), Timestamp: parseFacebookTimestamp(firstString(notif, "creation_time", "timestamp"))})
		if len(items) == limit {
			break
		}
	}
	return items, nil
}

func (b *messagixBackend) SetFacebookBio(ctx context.Context, bio string, publish bool) error {
	actorID, mutationID, err := b.facebookMutationIDs()
	if err != nil {
		return err
	}
	result, err := b.graphQL(ctx, protocol.SetBioFriendlyName, protocol.SetBioDocID, map[string]any{"input": map[string]any{"bio": bio, "publish_bio_feed_story": publish, "actor_id": actorID, "client_mutation_id": mutationID}, "hasProfileTileViewID": false, "profileTileViewID": nil, "scale": 1})
	if err != nil {
		return err
	}
	returned := stringValue(mapAt(result, "data", "profile_intro_card_set", "profile_intro_card", "bio")["text"])
	if returned != bio {
		return errors.New("Facebook did not confirm the requested bio")
	}
	return nil
}

func (b *messagixBackend) CreateAdditionalProfile(ctx context.Context, name, username string) error {
	actorID, mutationID, err := b.facebookMutationIDs()
	if err != nil {
		return err
	}
	result, err := b.graphQL(ctx, protocol.AdditionalProfileFriendlyName, protocol.AdditionalProfileDocID, map[string]any{"input": map[string]any{"name": name, "source": "PROFILE_SWITCHER", "user_name": username, "actor_id": actorID, "client_mutation_id": mutationID}})
	if err != nil {
		return err
	}
	created := mapAt(result, "data", "additional_profile_create")
	if message := stringValue(created["error_message"]); message != "" {
		return errors.New(message)
	}
	if len(created) == 0 {
		return errors.New("Facebook did not confirm additional profile creation")
	}
	return nil
}

func (b *messagixBackend) UnfriendFacebookUser(ctx context.Context, userID model.ID) error {
	actorID, _, err := b.facebookMutationIDs()
	if err != nil {
		return err
	}
	restricted := base64.StdEncoding.EncodeToString([]byte("restrictedUserNode" + userID.String()))
	result, err := b.graphQL(ctx, protocol.UnfriendFriendlyName, protocol.UnfriendDocID, map[string]any{"input": map[string]any{"source": "friending_jewel", "unfriended_user_id": restricted, "actor_id": actorID, "client_mutation_id": "1"}, "scale": 3})
	if err != nil {
		return err
	}
	data := mapAt(result, "data")
	if len(data) == 0 || containsFailedSuccess(data) {
		return errors.New("Facebook did not confirm unfriend mutation")
	}
	return nil
}

func (b *messagixBackend) SetFacebookBlocked(ctx context.Context, userID model.ID, blocked bool) error {
	actorID, mutationID, err := b.facebookMutationIDs()
	if err != nil {
		return err
	}
	friendlyName, docID := protocol.BlockFriendlyName, protocol.BlockDocID
	variables := map[string]any{"collectionID": nil, "hasCollectionAndSectionID": false, "input": map[string]any{"blocksource": "PROFILE", "should_apply_to_later_created_profiles": false, "user_id": int64Value(userID.String()), "actor_id": actorID, "client_mutation_id": mutationID}, "scale": 3, "sectionID": nil, "isPrivacyCheckupContext": false}
	if !blocked {
		friendlyName, docID = protocol.UnblockFriendlyName, protocol.UnblockDocID
		variables = map[string]any{"input": map[string]any{"block_action": "UNBLOCK", "setting": "USER", "target_id": userID.String(), "actor_id": actorID, "client_mutation_id": mutationID}, "profile_picture_size": 36}
	}
	result, err := b.graphQL(ctx, friendlyName, docID, variables)
	if err != nil {
		return err
	}
	if len(mapAt(result, "data")) == 0 {
		return errors.New("Facebook did not confirm block mutation")
	}
	return nil
}

func (b *messagixBackend) CreateFacebookPost(ctx context.Context, text string) (*model.Post, error) {
	actorID, mutationID, err := b.facebookMutationIDs()
	if err != nil {
		return nil, err
	}
	sessionID, err := webapi.NewSessionID()
	if err != nil {
		return nil, err
	}
	now := time.Now().UnixMilli()
	variables := map[string]any{"input": map[string]any{"composer_entry_point": "inline_composer", "composer_source_surface": "timeline", "source": "WWW", "attachments": []any{}, "audience": map[string]any{"privacy": map[string]any{"allow": []any{}, "base_state": "EVERYONE", "deny": []any{}, "tag_expansion_state": "UNSPECIFIED"}}, "message": map[string]any{"ranges": []any{}, "text": text}, "with_tags_ids": []any{}, "inline_activities": []any{}, "explicit_place_id": "0", "text_format_preset_id": "0", "logging": map[string]any{"composer_session_id": strconv.FormatUint(sessionID, 10)}, "navigation_data": map[string]any{"attribution_id_v2": fmt.Sprintf("ProfileCometTimelineListViewRoot.react,comet.profile.timeline.list,tap_bookmark,%d,0,%s", now, actorID)}, "tracking": "[null]", "actor_id": actorID, "client_mutation_id": mutationID}, "displayCommentsFeedbackContext": nil, "displayCommentsContextEnableComment": nil, "displayCommentsContextIsAdPreview": nil, "displayCommentsContextIsAggregatedShare": nil, "displayCommentsContextIsStorySet": nil, "feedLocation": "TIMELINE", "focusCommentID": nil, "scale": "1", "privacySelectorRenderLocation": "COMET_STREAM", "renderLocation": "timeline", "useDefaultActor": false, "inviteShortLinkKey": nil, "isFeed": false, "isFundraiser": false, "isFunFactPost": false, "isGroup": false, "isEvent": false, "isTimeline": true, "isSocialLearning": false, "isPageNewsFeed": false, "isProfileReviews": false, "isWorkSharedDraft": false, "UFI2CommentsProvider_commentsKey": "ProfileCometTimelineRoute", "hashtag": nil, "canUserManageOffers": false, "__relay_internal__pv__IsWorkUserrelayprovider": false, "__relay_internal__pv__IsMergQAPollsrelayprovider": false, "__relay_internal__pv__StoriesArmadilloReplyEnabledrelayprovider": false, "__relay_internal__pv__StoriesRingrelayprovider": false}
	result, err := b.graphQL(ctx, protocol.CreatePostFriendlyName, protocol.CreatePostDocID, variables)
	if err != nil {
		return nil, err
	}
	story := mapAt(result, "data", "story_create", "story")
	url := stringValue(story["url"])
	if url == "" {
		return nil, errors.New("Facebook did not return the created post")
	}
	return &model.Post{ID: model.ID(firstString(story, "id", "post_id")), URL: url}, nil
}

func (b *messagixBackend) ArchiveFacebookPost(ctx context.Context, postID model.ID, ownership model.PostOwnership) error {
	return b.setPostDisposition(ctx, postID, ownership, true)
}

func (b *messagixBackend) DeleteFacebookPost(ctx context.Context, postID model.ID, ownership model.PostOwnership) error {
	return b.setPostDisposition(ctx, postID, ownership, false)
}

func (b *messagixBackend) setPostDisposition(ctx context.Context, postID model.ID, ownership model.PostOwnership, archive bool) error {
	actorID, _, err := b.facebookMutationIDs()
	if err != nil {
		return err
	}
	storyID := "S:_I1054626957723036:" + postID.String()
	if ownership == model.PostShared {
		storyID = "S:_I" + actorID + ":" + postID.String() + ":" + postID.String()
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(storyID))
	friendlyName, docID, resultPath := protocol.DeletePostFriendlyName, protocol.DeletePostDocID, "move_to_trash_story"
	input := map[string]any{"story_id": encoded, "story_location": "TIMELINE", "actor_id": actorID, "client_mutation_id": "1"}
	if archive {
		friendlyName, docID, resultPath = protocol.ArchivePostFriendlyName, protocol.ArchivePostDocID, "archive_story"
		input["surface"] = "POST_CHEVRON_MENU_TIMELINE"
	}
	result, err := b.graphQL(ctx, friendlyName, docID, map[string]any{"input": input})
	if err != nil {
		return err
	}
	if !boolValue(mapAt(result, "data", resultPath)["success"]) {
		return errors.New("Facebook did not confirm post mutation")
	}
	return nil
}

func (b *messagixBackend) facebookMutationIDs() (string, string, error) {
	account, err := b.client.GetCurrentAccount()
	if err != nil {
		return "", "", err
	}
	actorID := strconv.FormatInt(account.GetFBID(), 10)
	if actorID == "0" {
		return "", "", errors.New("Facebook actor ID is unavailable")
	}
	mutationID, err := webapi.NewSessionID()
	if err != nil {
		return "", "", err
	}
	return actorID, strconv.FormatUint(mutationID, 10), nil
}

func containsFailedSuccess(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		if success, exists := typed["success"]; exists && !boolValue(success) {
			return true
		}
		for _, child := range typed {
			if containsFailedSuccess(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if containsFailedSuccess(child) {
				return true
			}
		}
	}
	return false
}

func (b *messagixBackend) CreateMarketplaceListing(ctx context.Context, input model.MarketplaceListingInput) (*model.MarketplaceListing, error) {
	categoryID, ok := marketplaceCategoryID[input.Category]
	if !ok {
		return nil, fmt.Errorf("unsupported Marketplace category %q", input.Category)
	}
	actorID, mutationID, err := b.facebookMutationIDs()
	if err != nil {
		return nil, err
	}
	now := time.Now().UnixMilli()
	photoIDs := make([]string, 0, len(input.PhotoIDs))
	for _, id := range input.PhotoIDs {
		if id.Empty() {
			return nil, errors.New("Marketplace photo ID is empty")
		}
		photoIDs = append(photoIDs, id.String())
	}
	attributeData, err := json.Marshal(map[string]any{"vt_attributes_free_form": map[string]string{"372885700169792": input.Brand}, "vt_attributes_normalized": map[string]any{}, "condition": "new", "brand": input.Brand})
	if err != nil {
		return nil, err
	}
	common := map[string]any{
		"attribute_data_json": string(attributeData), "category_id": categoryID, "commerce_shipping_carrier": nil, "commerce_shipping_carriers": []any{}, "comparable_price": "null", "cost_per_additional_item": nil,
		"delivery_types": []string{"IN_PERSON"}, "description": map[string]string{"text": input.Description}, "draft_type": nil, "hidden_from_friends_visibility": "VISIBLE_TO_EVERYONE", "is_personalization_required": nil,
		"is_preview": false, "item_price": map[string]string{"currency": strings.ToUpper(input.Currency), "price": input.Price}, "latitude": input.Location.Latitude, "longitude": input.Location.Longitude,
		"min_acceptable_checkout_offer_price": "null", "personalization_info": nil, "product_hashtag_names": input.Hashtags, "quantity": -1, "shipping_calculation_logic_version": nil, "shipping_cost_option": "BUYER_PAID_SHIPPING",
		"shipping_cost_range_lower_cost": nil, "shipping_cost_range_upper_cost": nil, "shipping_label_price": "0", "shipping_label_rate_code": nil, "shipping_label_rate_type": nil, "shipping_offered": false,
		"shipping_options_data": []any{}, "shipping_package_weight": nil, "shipping_price": "null", "shipping_service_type": nil, "sku": "", "source_type": "marketplace_unknown", "suggested_hashtag_names": []any{},
		"surface": "composer", "title": input.Title, "variants": []any{}, "video_ids": []any{}, "xpost_target_ids": []any{}, "photo_ids": photoIDs,
	}
	variables := map[string]any{"input": map[string]any{"client_mutation_id": mutationID, "actor_id": actorID, "attribution_id_v2": fmt.Sprintf("CometMarketplaceComposerRoot.react,comet.marketplace.composer,via_cold_start,%d,0,1606854132932955,", now), "audience": map[string]any{"marketplace": map[string]string{"marketplace_id": strconv.FormatInt(categoryID, 10)}}, "data": map[string]any{"common": common}}}
	result, err := b.graphQL(ctx, protocol.MarketplaceCreateFriendlyName, protocol.MarketplaceCreateDocID, variables)
	if err != nil {
		return nil, err
	}
	story := mapAt(result, "data", "marketplace_listing_create", "listing", "story")
	id, listingURL := model.ID(stringValue(story["id"])), stringValue(story["url"])
	if id.Empty() || listingURL == "" {
		return nil, errors.New("Facebook did not return the created Marketplace listing")
	}
	return &model.MarketplaceListing{ID: id, Title: input.Title, Description: input.Description, Price: input.Price, Currency: strings.ToUpper(input.Currency), Location: input.Location, URL: listingURL}, nil
}

func (b *messagixBackend) GetMarketplaceListing(ctx context.Context, listingID model.ID) (*model.MarketplaceListing, error) {
	variables := map[string]any{"UFI2CommentsProvider_commentsKey": "MarketplacePDP", "feedbackSource": 56, "feedLocation": "MARKETPLACE_MEGAMALL", "referralCode": "marketplace_top_picks", "scale": 3, "should_show_new_pdp": false, "targetId": listingID.String(), "useDefaultActor": false, "__relay_internal__pv__CometUFIIsRTAEnabledrelayprovider": false}
	result, err := b.graphQL(ctx, protocol.MarketplaceDetailFriendlyName, protocol.MarketplaceDetailDocID, variables)
	if err != nil {
		return nil, err
	}
	page := mapAt(result, "data", "viewer", "marketplace_product_details_page")
	renderable, target := mapAt(page, "marketplace_listing_renderable_target"), mapAt(page, "target")
	if len(renderable) == 0 || len(target) == 0 {
		return nil, errors.New("Marketplace response did not contain listing details")
	}
	actors, _ := mapAt(target, "story")["actors"].([]any)
	var seller model.FacebookUser
	if len(actors) > 0 {
		if actor, ok := actors[0].(map[string]any); ok {
			seller = model.FacebookUser{ID: model.ID(stringValue(actor["id"])), Name: stringValue(actor["name"]), ProfileURL: firstString(actor, "url", "profile_url")}
		}
	}
	locationMap := mapAt(renderable, "location")
	location := model.MarketplaceLocation{Name: firstString(locationMap, "reverse_geocode", "name")}
	location.Latitude, _ = strconv.ParseFloat(stringValue(locationMap["latitude"]), 64)
	location.Longitude, _ = strconv.ParseFloat(stringValue(locationMap["longitude"]), 64)
	price := mapAt(target, "listing_price")
	return &model.MarketplaceListing{ID: listingID, Title: stringValue(renderable["marketplace_listing_title"]), Description: stringValue(mapAt(target, "redacted_description")["text"]), Price: stringValue(price["amount"]), Currency: stringValue(price["currency"]), Seller: seller, Location: location, URL: stringValue(mapAt(target, "story")["url"]), CreatedAt: parseFacebookTimestamp(stringValue(target["creation_time"]))}, nil
}

func (b *messagixBackend) SetProfessionalMode(ctx context.Context, enabled bool) error {
	friendlyName, docID := protocol.ProfessionalDisableFriendlyName, protocol.ProfessionalDisableDocID
	variables := map[string]any{}
	if enabled {
		friendlyName, docID = protocol.ProfessionalEnableFriendlyName, protocol.ProfessionalEnableDocID
		value, err := webapi.NewSessionID()
		if err != nil {
			return err
		}
		const maxCategoryID uint64 = 1738263827237839
		variables = map[string]any{"category_id": int64(value%maxCategoryID + 1), "surface": nil}
	}
	result, err := b.graphQL(ctx, friendlyName, docID, variables)
	if err != nil {
		return err
	}
	if len(mapAt(result, "data")) == 0 {
		return errors.New("Facebook did not confirm Professional Mode mutation")
	}
	return nil
}

var marketplaceCategoryID = map[string]int64{
	"Tools": 2171028376552553, "Furniture": 1583634935226685, "Household": 1569171756675761, "Garden": 800089866739547, "Appliances": 678754142233400,
	"Video Games": 686977074745292, "Books Movies&Music": 613858625416355, "Bags & Luggage": 1567543000236608, "Women's clothing & shoes": 1266429133383966,
	"Men's clothing & shoes": 931157863635831, "Jewelry & Accessories": 214968118845643, "Health & beauty": 1555452698044988, "Pet Supplies": 1550246318620997,
	"Baby & kids": 624859874282116, "Toys & Games": 606456512821491, "Electronics & computers": 1792291877663080, "Mobile phones": 1557869527812749,
	"Bicycles": 1658310421102081, "Arts & Crafts": 1534799543476160, "Sports & Outdoors": 1383948661922113, "Auto parts": 757715671026531,
	"Musical Instruments": 676772489112490, "Antiques & Collectibles": 393860164117441, "Garage Sale": 1834536343472201, "Miscellaneous": 895487550471874,
}

func (b *messagixBackend) postFacebookForm(ctx context.Context, endpoint string, form url.Values) (map[string]any, error) {
	headers := b.graphQLHeaders("")
	_, data, err := b.client.GetHTTP().MakeRequest(ctx, endpoint, http.MethodPost, headers, []byte(form.Encode()), metaTypes.FORM)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if err := webapi.DecodeJSONObject(data, true, &result); err != nil {
		return nil, err
	}
	if err := facebookErrorEnvelope(result); err != nil {
		return nil, err
	}
	return result, nil
}
