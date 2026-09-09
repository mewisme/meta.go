package auth

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/rs/zerolog"
	"go.mau.fi/mautrix-meta/pkg/messagix"
	"go.mau.fi/mautrix-meta/pkg/messagix/bloks"
	metaCookies "go.mau.fi/mautrix-meta/pkg/messagix/cookies"
	metaTypes "go.mau.fi/mautrix-meta/pkg/messagix/types"
	fberrors "go.mewis.me/fbgo/errors"
	"maunium.net/go/mautrix/bridgev2"
)

type LoginField struct {
	ID          string
	Name        string
	Description string
	Secret      bool
	Options     []string
}

type LoginChallenge struct {
	ID           string
	Instructions string
	Fields       []LoginField
}

type CredentialLogin struct {
	client *messagix.Client
}

func NewCredentialLogin(logger zerolog.Logger) *CredentialLogin {
	jar := &metaCookies.Cookies{Platform: metaTypes.MessengerLite}
	client := messagix.NewClient(jar, logger, &messagix.Config{})
	return &CredentialLogin{client: client}
}

func (l *CredentialLogin) Step(ctx context.Context, input map[string]string) (*LoginChallenge, Cookies, error) {
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
	challenge := &LoginChallenge{ID: step.StepID, Instructions: step.Instructions}
	if step.UserInputParams != nil {
		for _, field := range step.UserInputParams.Fields {
			typeName := strings.ToLower(string(field.Type))
			secret := strings.Contains(typeName, "password") || strings.Contains(typeName, "secret") || strings.Contains(typeName, "code") || strings.Contains(typeName, "captcha")
			challenge.Fields = append(challenge.Fields, LoginField{ID: field.ID, Name: field.Name, Description: field.Description, Secret: secret, Options: append([]string(nil), field.Options...)})
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
	if errors.As(err, &checkpoint) || errors.Is(err, messagix.ErrCheckpointRequired) || strings.Contains(strings.ToLower(err.Error()), "checkpoint") {
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
