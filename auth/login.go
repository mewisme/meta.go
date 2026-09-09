package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/rs/zerolog"
	"go.mau.fi/mautrix-meta/pkg/messagix"
	"go.mau.fi/mautrix-meta/pkg/messagix/bloks"
	metaCookies "go.mau.fi/mautrix-meta/pkg/messagix/cookies"
	metaTypes "go.mau.fi/mautrix-meta/pkg/messagix/types"
	fberrors "go.mewis.me/fbgo/errors"
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
		var checkpoint bloks.CheckpointError
		if errors.As(err, &checkpoint) || strings.Contains(strings.ToLower(err.Error()), "checkpoint") {
			return nil, nil, fmt.Errorf("%w: %v", fberrors.ErrCheckpointRequired, err)
		}
		return nil, nil, err
	}
	if result != nil {
		cookies := Cookies{}
		for key, value := range result.GetAll() {
			cookies[string(key)] = value
		}
		if err := cookies.ValidateRegular(); err != nil {
			return nil, nil, err
		}
		return nil, cookies, nil
	}
	if step == nil {
		return nil, nil, errors.New("login flow returned neither challenge nor cookies")
	}
	challenge := &LoginChallenge{ID: step.StepID, Instructions: step.Instructions}
	if step.UserInputParams != nil {
		for _, field := range step.UserInputParams.Fields {
			typeName := strings.ToLower(string(field.Type))
			challenge.Fields = append(challenge.Fields, LoginField{ID: field.ID, Name: field.Name, Description: field.Description, Secret: strings.Contains(typeName, "password") || strings.Contains(typeName, "secret"), Options: append([]string(nil), field.Options...)})
		}
	}
	return challenge, nil, nil
}
