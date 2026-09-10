package auth

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"go.mewis.me/meta-extra/pkg/messagix"
	"go.mewis.me/meta-extra/pkg/messagix/bloks"
	metaCookies "go.mewis.me/meta-extra/pkg/messagix/cookies"
	metaHTTP "go.mewis.me/meta-extra/pkg/messagix/httpclient"
	metaTypes "go.mewis.me/meta-extra/pkg/messagix/types"
	fberrors "go.mewis.me/meta.go/errors"
	"go.mewis.me/meta.go/internal/protocol"
	"maunium.net/go/mautrix/bridgev2"
)

var ErrTwoFactorRequired = errors.New("two-factor code required")

const fb4aUserAgent = "Dalvik/2.1.0 (Linux; U; Android 7.1.2; SM-G988N Build/NRD90M) [FBAN/FB4A;FBAV/340.0.0.27.113;FBPN/com.facebook.katana;FBLC/vi_VN;FBBV/324485361;FBCR/Viettel Mobile;FBMF/samsung;FBBD/samsung;FBDV/SM-G988N;FBSV/7.1.2;FBCA/x86:armeabi-v7a;FBDM/{density=1.0,width=540,height=960};FB_FW/1;FBRV/0;]"

var fb4aTwoFactorSubcodes = map[int]struct{}{1348162: {}, 1348023: {}}

type Credentials struct {
	Identifier string
	Password   string
	TOTP       string
	OTP        string
}

type loginField struct {
	ID          string
	Name        string
	Description string
	Secret      bool
	Options     []string
}

type loginChallenge struct {
	ID           string
	Instructions string
	Fields       []loginField
}

type CredentialLogin struct {
	client *messagix.Client
}

func NewCredentialLogin() *CredentialLogin {
	jar := &metaCookies.Cookies{Platform: metaTypes.MessengerLiteIOS}
	client := messagix.NewClient(jar, zerolog.Nop(), &messagix.Config{})
	return &CredentialLogin{client: client}
}

func (l *CredentialLogin) Login(ctx context.Context, credentials Credentials) (Cookies, error) {
	return loginCredentials(ctx, credentials, l.step, func(ctx context.Context, credentials Credentials) (Cookies, error) {
		return loginFB4A(ctx, credentials, fb4aConfig{})
	})
}

type credentialStep func(context.Context, map[string]string) (*loginChallenge, Cookies, error)
type credentialFallback func(context.Context, Credentials) (Cookies, error)

func loginCredentials(ctx context.Context, credentials Credentials, step credentialStep, fallback credentialFallback) (Cookies, error) {
	if err := validateCredentials(credentials); err != nil {
		return nil, err
	}
	responses := map[string]string{}
	for attempts := 0; attempts < 12; attempts++ {
		challenge, cookies, err := step(ctx, responses)
		if err != nil {
			if errors.Is(err, fberrors.ErrProtocolChanged) && fallback != nil {
				return fallback(ctx, credentials)
			}
			return nil, err
		}
		if cookies != nil {
			return cookies, nil
		}
		if challenge == nil {
			if fallback != nil {
				return fallback(ctx, credentials)
			}
			return nil, &fberrors.ProtocolError{Operation: "credential login challenge", Cause: fberrors.ErrProtocolChanged}
		}
		responses, err = resolveCredentialFields(challenge.Fields, credentials)
		if err != nil {
			if errors.Is(err, fberrors.ErrProtocolChanged) && fallback != nil {
				return fallback(ctx, credentials)
			}
			return nil, err
		}
	}
	if fallback != nil {
		return fallback(ctx, credentials)
	}
	return nil, &fberrors.ProtocolError{Operation: "credential login challenge flow", Cause: fberrors.ErrProtocolChanged}
}

func validateCredentials(credentials Credentials) error {
	if strings.TrimSpace(credentials.Identifier) == "" {
		return fmt.Errorf("%w: identifier is required", fberrors.ErrInvalidInput)
	}
	if credentials.Password == "" {
		return fmt.Errorf("%w: password is required", fberrors.ErrInvalidInput)
	}
	if credentials.TOTP != "" && credentials.OTP != "" {
		return fmt.Errorf("%w: totp and otp are mutually exclusive", fberrors.ErrInvalidInput)
	}
	if credentials.TOTP != "" {
		if _, err := TOTP(credentials.TOTP, time.Now()); err != nil {
			return fmt.Errorf("%w: invalid TOTP secret", fberrors.ErrInvalidInput)
		}
	}
	if credentials.OTP != "" {
		value := strings.ReplaceAll(strings.TrimSpace(credentials.OTP), " ", "")
		if len(value) < 6 || len(value) > 8 {
			return fmt.Errorf("%w: otp must contain 6 to 8 digits", fberrors.ErrInvalidInput)
		}
		if _, err := strconv.ParseUint(value, 10, 32); err != nil {
			return fmt.Errorf("%w: otp must contain 6 to 8 digits", fberrors.ErrInvalidInput)
		}
	}
	return nil
}

func resolveCredentialFields(fields []loginField, credentials Credentials) (map[string]string, error) {
	responses := make(map[string]string, len(fields))
	for _, field := range fields {
		label := strings.ToLower(field.ID + " " + field.Name + " " + field.Description)
		switch {
		case strings.Contains(label, "password"):
			responses[field.ID] = credentials.Password
		case isCredentialOTPField(label):
			otp, err := credentialOTP(credentials, time.Now())
			if err != nil {
				return nil, err
			}
			responses[field.ID] = otp
		case isCredentialIdentifierField(field, label):
			responses[field.ID] = credentials.Identifier
		case len(field.Options) == 1:
			responses[field.ID] = field.Options[0]
		case len(field.Options) > 1 && (credentials.TOTP != "" || credentials.OTP != ""):
			selected := ""
			for _, option := range field.Options {
				if strings.EqualFold(strings.TrimSpace(option), "Authentication app") {
					selected = option
					break
				}
			}
			if selected == "" {
				return nil, &fberrors.ProtocolError{Operation: "credential login MFA method", Cause: fberrors.ErrProtocolChanged}
			}
			responses[field.ID] = selected
		default:
			return nil, &fberrors.ProtocolError{Operation: "credential login field", Code: field.ID, Cause: fberrors.ErrProtocolChanged}
		}
	}
	return responses, nil
}

func isCredentialIdentifierField(field loginField, label string) bool {
	id := strings.ToLower(strings.TrimSpace(field.ID))
	return id == "login" || id == "identifier" || strings.Contains(label, "email") || strings.Contains(label, "username") || strings.Contains(label, "phone") || strings.Contains(label, "identifier")
}

func isCredentialOTPField(label string) bool {
	return strings.Contains(label, "totp") || strings.Contains(label, "authenticator") || strings.Contains(label, "two_factor") || strings.Contains(label, "two factor") || strings.Contains(label, "2fa") || strings.Contains(label, "otp")
}

func credentialOTP(credentials Credentials, now time.Time) (string, error) {
	if credentials.OTP != "" {
		return ResolveOTP(credentials.OTP, now)
	}
	if credentials.TOTP != "" {
		return TOTP(credentials.TOTP, now)
	}
	return "", ErrTwoFactorRequired
}

func (l *CredentialLogin) step(ctx context.Context, input map[string]string) (*loginChallenge, Cookies, error) {
	if l == nil || l.client == nil {
		return nil, nil, errors.New("nil credential login")
	}
	step, result, err := l.client.MessengerLite.DoLoginSteps(ctx, input)
	if err != nil {
		return nil, nil, normalizeCredentialLoginError(err)
	}
	if result != nil {
		cookies := Cookies{}
		for key, value := range result.GetAll() {
			cookies[string(key)] = value
		}
		if err := cookies.ValidateRegular(); err != nil {
			return nil, nil, &fberrors.ProtocolError{Operation: "credential login result", Cause: fberrors.ErrProtocolChanged}
		}
		return nil, cookies, nil
	}
	if step == nil {
		return nil, nil, &fberrors.ProtocolError{Operation: "credential login flow", Cause: fberrors.ErrProtocolChanged}
	}
	challenge := &loginChallenge{ID: step.StepID, Instructions: step.Instructions}
	if step.UserInputParams != nil {
		for _, field := range step.UserInputParams.Fields {
			typeName := strings.ToLower(string(field.Type))
			secret := strings.Contains(typeName, "password") || strings.Contains(typeName, "secret") || strings.Contains(typeName, "code") || strings.Contains(typeName, "captcha")
			challenge.Fields = append(challenge.Fields, loginField{ID: field.ID, Name: field.Name, Description: field.Description, Secret: secret, Options: append([]string(nil), field.Options...)})
		}
	}
	return challenge, nil, nil
}

func normalizeCredentialLoginError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return err
	}
	var checkpoint bloks.CheckpointError
	if errors.As(err, &checkpoint) || errors.Is(err, metaHTTP.ErrCheckpointRequired) || strings.Contains(strings.ToLower(err.Error()), "checkpoint") {
		return fmt.Errorf("%w: credential login requires checkpoint", fberrors.ErrCheckpointRequired)
	}
	if resp := loginResponseError(err); resp != nil {
		if resp.ErrCode == "FI.MAU.META_PHONE_NUMBER" {
			return fmt.Errorf("%w: %s", fberrors.ErrInvalidInput, resp.Err)
		}
		return fmt.Errorf("%w: %s", fberrors.ErrUnauthorized, resp.Err)
	}
	lower := strings.ToLower(err.Error())
	for _, marker := range []string{"invalid username or password", "incorrect password", "not connected to a messenger account", "not connected to an account", "login rejected"} {
		if strings.Contains(lower, marker) {
			return fmt.Errorf("%w: credential login rejected", fberrors.ErrUnauthorized)
		}
	}
	return &fberrors.ProtocolError{Operation: "credential login", Cause: fberrors.ErrProtocolChanged}
}

func loginResponseError(err error) *bridgev2.RespError {
	var value bridgev2.RespError
	if errors.As(err, &value) {
		return &value
	}
	var pointer *bridgev2.RespError
	if errors.As(err, &pointer) {
		return pointer
	}
	return nil
}

type fb4aConfig struct {
	HTTP           *http.Client
	URL            string
	APIKey         string
	AppAccessToken string
}

type fb4aResponse struct {
	AccessToken    string        `json:"access_token"`
	SessionCookies []fb4aCookie  `json:"session_cookies"`
	Error          *fb4aAPIError `json:"error"`
}

type fb4aCookie struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type fb4aAPIError struct {
	Code        int             `json:"code"`
	Subcode     int             `json:"error_subcode"`
	UserTitle   string          `json:"error_user_title"`
	UserMessage string          `json:"error_user_msg"`
	FBTraceID   string          `json:"fbtrace_id"`
	ErrorData   json.RawMessage `json:"error_data"`
}

func loginFB4A(ctx context.Context, credentials Credentials, cfg fb4aConfig) (Cookies, error) {
	if err := validateCredentials(credentials); err != nil {
		return nil, err
	}
	state, err := newFB4AState(credentials, cfg)
	if err != nil {
		return nil, err
	}
	response, status, err := state.login(ctx, state.form(credentials.Password, "password", 1))
	if err != nil {
		return nil, err
	}
	if response.Error == nil {
		return fb4aCookies(response)
	}
	if _, ok := fb4aTwoFactorSubcodes[response.Error.Subcode]; !ok {
		return nil, fb4aResponseError(response.Error, status, state.url)
	}
	otp, err := credentialOTP(credentials, time.Now())
	if err != nil {
		return nil, err
	}
	userID, firstFactor := fb4aTwoFactorMetadata(response.Error.ErrorData)
	if userID == "" || firstFactor == "" {
		return nil, &fberrors.ProtocolError{Operation: "FB4A two-factor metadata", Endpoint: state.url, Subcode: strconv.Itoa(response.Error.Subcode), FBTraceID: response.Error.FBTraceID, StatusCode: status, Cause: fberrors.ErrProtocolChanged}
	}
	response, status, err = state.login(ctx, state.twoFactorForm(otp, otp, userID, firstFactor, 2))
	if err != nil {
		return nil, err
	}
	if response.Error == nil {
		return fb4aCookies(response)
	}
	response, status, err = state.login(ctx, state.twoFactorForm(credentials.Password, otp, userID, firstFactor, 3))
	if err != nil {
		return nil, err
	}
	if response.Error == nil {
		return fb4aCookies(response)
	}
	if credentials.TOTP != "" {
		retryOTP, retryErr := TOTP(credentials.TOTP, time.Now())
		if retryErr != nil {
			return nil, retryErr
		}
		if retryOTP != otp {
			response, status, err = state.login(ctx, state.twoFactorForm(retryOTP, retryOTP, userID, firstFactor, 4))
			if err != nil {
				return nil, err
			}
			if response.Error == nil {
				return fb4aCookies(response)
			}
		}
	}
	return nil, fb4aResponseError(response.Error, status, state.url)
}

type fb4aState struct {
	credentials Credentials
	httpClient  *http.Client
	url         string
	apiKey      string
	appToken    string
	deviceID    string
	adID        string
	secureID    string
	machineID   string
}

func newFB4AState(credentials Credentials, cfg fb4aConfig) (*fb4aState, error) {
	deviceID, err := randomFB4AID()
	if err != nil {
		return nil, err
	}
	machineID, err := randomLowerAlphaNum(24)
	if err != nil {
		return nil, err
	}
	client := cfg.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	endpoint := strings.TrimSpace(cfg.URL)
	if endpoint == "" {
		endpoint = protocol.FB4AAuthURL
	}
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" {
		apiKey = protocol.FB4AAPIKey
	}
	appToken := strings.TrimSpace(cfg.AppAccessToken)
	if appToken == "" {
		appToken = protocol.FB4AAppAccessToken
	}
	return &fb4aState{credentials: credentials, httpClient: client, url: endpoint, apiKey: apiKey, appToken: appToken, deviceID: deviceID, adID: deviceID, secureID: deviceID, machineID: machineID}, nil
}

func (s *fb4aState) form(password, credentialType string, tryNum int) url.Values {
	values := url.Values{
		"adid": {s.adID}, "format": {"json"}, "device_id": {s.deviceID}, "email": {s.credentials.Identifier}, "password": {password},
		"generate_analytics_claim": {"1"}, "community_id": {""}, "cpl": {"true"}, "try_num": {strconv.Itoa(tryNum)}, "family_device_id": {s.deviceID},
		"secure_family_device_id": {s.secureID}, "credentials_type": {credentialType}, "fb4a_shared_phone_cpl_experiment": {"fb4a_shared_phone_nonce_cpl_at_risk_v3"},
		"fb4a_shared_phone_cpl_group": {"enable_v3_at_risk"}, "enroll_misauth": {"false"}, "generate_session_cookies": {"1"}, "error_detail_type": {"button_with_disabled"},
		"source": {"login"}, "machine_id": {s.machineID}, "meta_inf_fbmeta": {""}, "advertiser_id": {s.adID}, "encrypted_msisdn": {""}, "currently_logged_in_userid": {"0"},
		"locale": {"vi_VN"}, "client_country_code": {"VN"}, "fb_api_req_friendly_name": {"authenticate"}, "fb_api_caller_class": {"Fb4aAuthHandler"},
		"api_key": {s.apiKey}, "access_token": {s.appToken},
	}
	if credentialType == "two_factor" {
		values.Set("jazoest", "22327")
		values.Set("sim_serials", "[]")
	} else {
		values.Set("jazoest", "22421")
	}
	return values
}

func (s *fb4aState) twoFactorForm(password, otp, userID, firstFactor string, tryNum int) url.Values {
	values := s.form(password, "two_factor", tryNum)
	values.Set("twofactor_code", otp)
	values.Set("userid", userID)
	values.Set("first_factor", firstFactor)
	return values
}

func (s *fb4aState) login(ctx context.Context, form url.Values) (fb4aResponse, int, error) {
	requestCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, s.url, strings.NewReader(form.Encode()))
	if err != nil {
		return fb4aResponse{}, 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Fb-Connection-Type", "unknown")
	req.Header.Set("User-Agent", fb4aUserAgent)
	req.Header.Set("X-Fb-Connection-Quality", "EXCELLENT")
	req.Header.Set("Authorization", "OAuth null")
	req.Header.Set("X-Fb-Friendly-Name", "authenticate")
	req.Header.Set("X-Fb-Server-Cluster", "True")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fb4aResponse{}, 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fb4aResponse{}, resp.StatusCode, err
	}
	var value fb4aResponse
	if err := json.Unmarshal(data, &value); err != nil {
		return fb4aResponse{}, resp.StatusCode, &fberrors.ProtocolError{Operation: "FB4A login response", Endpoint: s.url, StatusCode: resp.StatusCode, Cause: fberrors.ErrProtocolChanged}
	}
	if resp.StatusCode >= http.StatusBadRequest && value.Error == nil {
		return fb4aResponse{}, resp.StatusCode, &fberrors.ProtocolError{Operation: "FB4A login", Endpoint: s.url, StatusCode: resp.StatusCode, Cause: fberrors.ErrProtocolChanged}
	}
	return value, resp.StatusCode, nil
}

func fb4aCookies(response fb4aResponse) (Cookies, error) {
	cookies := Cookies{}
	for _, cookie := range response.SessionCookies {
		if strings.TrimSpace(cookie.Name) != "" {
			cookies[cookie.Name] = cookie.Value
		}
	}
	if err := cookies.ValidateRegular(); err != nil {
		return nil, &fberrors.ProtocolError{Operation: "FB4A login cookies", Cause: fberrors.ErrProtocolChanged}
	}
	return cookies, nil
}

func fb4aResponseError(value *fb4aAPIError, status int, endpoint string) error {
	if value == nil {
		return &fberrors.ProtocolError{Operation: "FB4A login", Endpoint: endpoint, StatusCode: status, Cause: fberrors.ErrProtocolChanged}
	}
	cause := error(fberrors.ErrUnauthorized)
	lower := strings.ToLower(value.UserTitle + " " + value.UserMessage)
	switch {
	case strings.Contains(lower, "checkpoint"):
		cause = fberrors.ErrCheckpointRequired
	case value.Code == 4 || value.Code == 17 || value.Code == 32 || value.Code == 613:
		cause = fberrors.ErrRateLimited
	}
	return &fberrors.ProtocolError{Operation: "FB4A login", Endpoint: endpoint, Code: strconv.Itoa(value.Code), Subcode: strconv.Itoa(value.Subcode), FBTraceID: value.FBTraceID, StatusCode: status, Cause: cause}
}

func fb4aTwoFactorMetadata(raw json.RawMessage) (string, string) {
	if len(raw) == 0 {
		return "", ""
	}
	if raw[0] == '"' {
		var text string
		if json.Unmarshal(raw, &text) != nil {
			return "", ""
		}
		raw = []byte(text)
	}
	var data map[string]any
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()
	if decoder.Decode(&data) != nil {
		return "", ""
	}
	return firstFB4AValue(data, "uid", "userid", "user_id"), firstFB4AValue(data, "login_first_factor", "first_factor", "first_factor_id")
}

func firstFB4AValue(data map[string]any, keys ...string) string {
	for _, key := range keys {
		switch value := data[key].(type) {
		case string:
			if strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
		case json.Number:
			return value.String()
		}
	}
	return ""
}

func randomFB4AID() (string, error) {
	parts := []int{8, 4, 4, 4, 12}
	values := make([]string, len(parts))
	for index, size := range parts {
		value, err := randomLowerAlphaNum(size)
		if err != nil {
			return "", err
		}
		values[index] = value
	}
	return strings.Join(values, "-"), nil
}

func randomLowerAlphaNum(length int) (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	buffer := make([]byte, length)
	max := big.NewInt(int64(len(alphabet)))
	for index := range buffer {
		value, err := cryptorand.Int(cryptorand.Reader, max)
		if err != nil {
			return "", err
		}
		buffer[index] = alphabet[value.Int64()]
	}
	return string(buffer), nil
}
