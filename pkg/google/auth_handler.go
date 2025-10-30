package google

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/klokku/klokku/internal/config"
	"github.com/klokku/klokku/internal/rest"
	"github.com/klokku/klokku/pkg/user"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

type googleAuthRedirect struct {
	RedirectUrl string `json:"redirectUrl"`
}

type GoogleAuth struct {
	db          *sql.DB
	userService user.Service
	oauthConfig *oauth2.Config
}

func NewGoogleAuth(db *sql.DB, userService user.Service, cfg config.Application) *GoogleAuth {
	oauthConfig := &oauth2.Config{
		ClientID:     cfg.Google.ClientId,
		ClientSecret: cfg.Google.ClientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  cfg.Host + "/api/integrations/google/auth/callback",
		Scopes:       []string{calendar.CalendarEventsScope, calendar.CalendarReadonlyScope},
	}

	return &GoogleAuth{db: db, userService: userService, oauthConfig: oauthConfig}
}

func (g *GoogleAuth) OAuthLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	currentUser, err := g.userService.GetCurrentUser(r.Context())
	if err != nil {
		log.Error("unable to retrieve current user: ", err)
		http.Error(w, "unable to retrieve current user", http.StatusInternalServerError)
		return
	}
	userId := currentUser.Id

	_, err = g.db.Exec("DELETE FROM google_calendar_auth WHERE user_id = ?", userId)
	if err != nil {
		log.Errorf("failed to delete old Google auth row for user %d: %v", userId, err)
		w.WriteHeader(http.StatusInternalServerError)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error: "Failed to handle Google authentication",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
		}
		return
	}

	stateNonce := uuid.New().String()
	finalUrl := r.URL.Query().Get("finalUrl")

	// store nonce for the use in the DB
	_, err = g.db.Exec("INSERT INTO google_calendar_auth (user_id, nonce) VALUES (?, ?)", userId, stateNonce)
	if err != nil {
		log.Errorf("failed to store Google auth nonce for user %d: %v", userId, err)
		w.WriteHeader(http.StatusInternalServerError)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error: "Failed to handle Google authentication",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
		}
		return
	}

	log.Tracef("Redirecting to Google auth URL with nonce: %s", stateNonce)
	u := g.oauthConfig.AuthCodeURL(finalUrl+"|"+stateNonce, oauth2.AccessTypeOffline, oauth2.ApprovalForce)

	w.WriteHeader(http.StatusOK)
	encodeErr := json.NewEncoder(w).Encode(googleAuthRedirect{
		RedirectUrl: u,
	})
	if encodeErr != nil {
		http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
	}
}

func (g *GoogleAuth) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	code := r.FormValue("code")
	state := r.FormValue("state")

	parts := strings.SplitN(state, "|", 2)
	finalUrl := parts[0]
	nonce := parts[1]

	token, err := g.oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		err := fmt.Errorf("unable to exchange code for token: %v", err)
		log.Error(err)
		http.Redirect(w, r, finalUrl+"?success=false", http.StatusFound)
		return
	}

	_, err = g.db.Exec("UPDATE google_calendar_auth SET access_token = ?, refresh_token = ?, expiry = ? WHERE nonce = ?",
		token.AccessToken, token.RefreshToken, token.Expiry.Unix(), nonce)
	if err != nil {
		err := fmt.Errorf("unable to store Google auth token for nonce: %v", err)
		log.Error(err)
		http.Redirect(w, r, finalUrl+"?success=false", http.StatusFound)
		return
	}
	log.Debug("Successfully stored Google auth token for nonce: ", nonce)
	http.Redirect(w, r, finalUrl+"?success=true", http.StatusFound)
}

func (g *GoogleAuth) getToken(ctx context.Context, userId int) (*oauth2.Token, error) {
	var token oauth2.Token
	var expiryTimestamp int64
	err := g.db.QueryRowContext(ctx, "SELECT access_token, refresh_token, expiry FROM google_calendar_auth WHERE user_id = ?", userId).
		Scan(&token.AccessToken, &token.RefreshToken, &expiryTimestamp)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("unable to retrieve Google auth token: %v", err)
	}

	token.Expiry = time.Unix(expiryTimestamp, 0)
	return &token, nil
}

func (g *GoogleAuth) getClient(ctx context.Context, userId int) (*http.Client, error) {
	token, err := g.getToken(ctx, userId)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	if token == nil {
		return nil, nil
	}
	return g.oauthConfig.Client(context.Background(), token), nil
}

func (g *GoogleAuth) OAuthLogout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userId, err := user.CurrentId(r.Context())
	if err != nil {
		log.Error("unable to retrieve current user: ", err)
		http.Error(w, "unable to retrieve current user", http.StatusInternalServerError)
		return
	}
	_, err = g.db.Exec("DELETE FROM google_calendar_auth WHERE user_id = ?", userId)

	if err != nil {
		log.Errorf("failed to delete Google auth row for user %d: %v", userId, err)
		w.WriteHeader(http.StatusInternalServerError)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error: "Failed to handle Google authentication",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
