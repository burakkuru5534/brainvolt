package firebase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	fb "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

// ErrTokenInvalid reports a registration token FCM no longer accepts
// (app uninstalled, token rotated): the caller forgets it.
var ErrTokenInvalid = errors.New("firebase: registration token invalid")

// Messenger sends push messages through FCM HTTP v1 (schema-platform §1).
type Messenger struct {
	client *messaging.Client
}

// NewMessenger returns an FCM client for projectID authenticated with:
// - a raw JSON service-account key (if credentials starts with '{')
// - a service-account key at credentialsFile (if non-empty)
// - Application Default Credentials (when credentials is empty)
func NewMessenger(ctx context.Context, projectID, credentials string) (*Messenger, error) {
	var opts []option.ClientOption
	cred := strings.TrimSpace(credentials)
	if cred != "" {
		if strings.HasPrefix(cred, "{") {
			opts = append(opts, option.WithCredentialsJSON([]byte(cred)))
		} else {
			opts = append(opts, option.WithAuthCredentialsFile(option.ServiceAccount, cred))
		}
	}
	app, err := fb.NewApp(ctx, &fb.Config{ProjectID: projectID}, opts...)
	if err != nil {
		return nil, fmt.Errorf("firebase: %w", err)
	}
	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("firebase messaging: %w", err)
	}
	return &Messenger{client: client}, nil
}

// Send delivers one notification with data to one device (its FCM
// installation id) and returns
// FCM's message id.
func (m *Messenger) Send(ctx context.Context, token, title, body string, data map[string]string) (string, error) {
	id, err := m.client.Send(ctx, &messaging.Message{
		// The app reports FCM registration tokens (firebase_messaging
		// getToken), which the Token field still addresses.
		Token: token, //nolint:staticcheck // SA1019: Fid needs installation ids the app does not send

		Notification: &messaging.Notification{Title: title, Body: body},
		Data:         data,
		Android:      &messaging.AndroidConfig{Priority: "high"},
		APNS:         &messaging.APNSConfig{Payload: &messaging.APNSPayload{Aps: &messaging.Aps{Sound: "default"}}},
	})
	if err != nil {
		if messaging.IsUnregistered(err) || messaging.IsInvalidArgument(err) {
			return "", fmt.Errorf("%w: %w", ErrTokenInvalid, err)
		}
		return "", err
	}
	return id, nil
}
