package web

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/SherClockHolmes/webpush-go"
)

// registerPushHandler handles requests for registering a push subscription
// into the subscription manager.
func (h *Handler) registerPushHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "registering for push notifications is not implemented yet", http.StatusNotImplemented)
	// TODO: here, add a new subscriber and invoke Save.
}

type Subscriber struct {
	UserEmail string // without mailto:
	Data      webpush.Subscription
	Create    time.Time
	Access    time.Time
}

func (s *Subscriber) SendNotification_prototype(ctx context.Context, body []byte, sm *SubscriptionManager) error {
	// might be good to do it in submanager.
	// might be good to add a wrapper that does this in a goroutine and reports status on a channel.
	resp, err := webpush.SendNotificationWithContext(ctx, body, &s.Data, &webpush.Options{
		Subscriber:      s.UserEmail,
		TTL:             30,
		VAPIDPublicKey:  sm.vapidPubKey,
		VAPIDPrivateKey: sm.vapidPrivKey,
	})
	if err != nil {
		return err
	}
	// TODO: any reason to read resp.Body?
	defer resp.Body.Close()

	return nil
}

type SubscriptionManager struct {
	config struct {
		Subscribers []Subscriber `json:"subscribers"`
	}
	vapidPubKey  string // base64 encoded
	vapidPrivKey string // base64 encoded
}

// NewSubscriptionManager creates a subscription manager using configuration
// from the passed io.Reader. If nil, the JSON is not decoded.
//
// The bundle can be created by invoking the subscription manager's Save method.
//
// The io.Reader must be closed by the caller even if this function returns an
// error.
func NewSubscriptionManager(bundle io.Reader) (*SubscriptionManager, error) {
	sm := &SubscriptionManager{}
	if bundle != nil {
		dec := json.NewDecoder(bundle)
		if err := dec.Decode(sm.config); err != nil {
			return nil, err
		}
	}
	return sm, nil
}

// Save saves the configuration to the passed io.Writer. This includes
// subscriptions known to the subscription manager.
//
// The io.Writer must be closed by the caller even if this function returns an
// error.
func (sm *SubscriptionManager) Save(bundle io.Writer) error {
	enc := json.NewEncoder(bundle)
	return enc.Encode(sm.config)
}
