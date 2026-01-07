package clickup

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
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klokku/klokku/internal/config"
	"github.com/klokku/klokku/internal/rest"
	"github.com/klokku/klokku/pkg/user"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

var endpoint = oauth2.Endpoint{
	AuthURL:       "https://app.clickup.com/api",
	TokenURL:      "https://api.clickup.com/api/v2/oauth/token",
	DeviceAuthURL: "https://app.clickup.com/api",
	AuthStyle:     oauth2.AuthStyleInParams,
}

type clickUpAuthRedirect struct {
	RedirectUrl string `json:"redirectUrl"`
}

type ClickUpAuth struct {
	db          *pgxpool.Pool
	userService user.Service
	oauthConfig *oauth2.Config
}

func NewClickUpAuth(db *pgxpool.Pool, userService user.Service, cfg config.Application) *ClickUpAuth {
	oauthConfig := &oauth2.Config{
		ClientID:     cfg.ClickUp.ClientId,
		ClientSecret: cfg.ClickUp.ClientSecret,
		Endpoint:     endpoint,
		RedirectURL:  cfg.Host + "/api/integrations/clickup/auth/callback",
	}

	return &ClickUpAuth{db: db, userService: userService, oauthConfig: oauthConfig}
}

// OAuthLogin godoc
// @Summary Initiate ClickUp OAuth login
// @Description Start the OAuth flow to connect ClickUp account
// @Tags ClickUp
// @Produce json
// @Param finalUrl query string false "URL to redirect to after authentication"
// @Success 200 {object} object{redirectUrl=string} "OAuth redirect URL"
// @Failure 403 {string} string "User not found"
// @Router /api/integrations/clickup/auth/login [get]
// @Security XUserId
func (g *ClickUpAuth) OAuthLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	currentUser, err := g.userService.GetCurrentUser(r.Context())
	if err != nil {
		log.Error("unable to retrieve current user: ", err)
		http.Error(w, "unable to retrieve current user", http.StatusInternalServerError)
		return
	}
	userId := currentUser.Id

	_, err = g.db.Exec(r.Context(), "DELETE FROM clickup_auth WHERE user_id = $1", userId)
	if err != nil {
		log.Errorf("failed to delete old ClickUp auth row for user %d: %v", userId, err)
		w.WriteHeader(http.StatusInternalServerError)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error: "Failed to handle ClickUp authentication",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
		}
		return
	}

	stateNonce := uuid.New().String()
	finalUrl := r.URL.Query().Get("finalUrl")

	// store nonce for the use in the DB
	_, err = g.db.Exec(r.Context(), "INSERT INTO clickup_auth (user_id, nonce) VALUES ($1, $2)", userId, stateNonce)
	if err != nil {
		log.Errorf("failed to store ClickUp auth nonce for user %d: %v", userId, err)
		w.WriteHeader(http.StatusInternalServerError)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error: "Failed to handle ClickUp authentication",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
		}
		return
	}

	log.Tracef("Redirecting to ClickUp auth URL with nonce: %s", stateNonce)
	u := g.oauthConfig.AuthCodeURL(finalUrl+"|"+stateNonce, oauth2.AccessTypeOffline, oauth2.ApprovalForce)

	w.WriteHeader(http.StatusOK)
	encodeErr := json.NewEncoder(w).Encode(clickUpAuthRedirect{
		RedirectUrl: u,
	})
	if encodeErr != nil {
		http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
	}
}

// OAuthCallback godoc
// @Summary ClickUp OAuth callback
// @Description Handle the OAuth callback from ClickUp
// @Tags ClickUp
// @Param code query string true "Authorization code"
// @Param state query string true "State parameter"
// @Success 302 "Redirect to finalUrl with success=true/false"
// @Router /api/integrations/clickup/auth/callback [get]
func (g *ClickUpAuth) OAuthCallback(w http.ResponseWriter, r *http.Request) {
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

	// Handle zero expiry time properly
	var expiryTimestamp *int64
	if !token.Expiry.IsZero() {
		timestamp := token.Expiry.Unix()
		expiryTimestamp = &timestamp
	}

	_, err = g.db.Exec(r.Context(), "UPDATE clickup_auth SET access_token = $1, refresh_token = $2, expiry = $3 WHERE nonce = $4",
		token.AccessToken, token.RefreshToken, expiryTimestamp, nonce)
	if err != nil {
		err := fmt.Errorf("unable to store ClickUp auth token for nonce: %v", err)
		log.Error(err)
		http.Redirect(w, r, finalUrl+"?success=false", http.StatusFound)
		return
	}
	log.Debug("Successfully stored ClickUp auth token for nonce: ", nonce)
	http.Redirect(w, r, finalUrl+"?success=true", http.StatusFound)
}

// IsAuthenticated godoc
// @Summary Check ClickUp authentication status
// @Description Check if the current user has authenticated with ClickUp
// @Tags ClickUp
// @Produce json
// @Success 200 {string} string "true"
// @Failure 403 {string} string "User not found"
// @Failure 404 "Not authenticated"
// @Router /api/integrations/clickup/auth [get]
// @Security XUserId
func (g *ClickUpAuth) IsAuthenticated(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userId, err := user.CurrentId(r.Context())
	if err != nil {
		log.Error("unable to retrieve current user: ", err)
		http.Error(w, "unable to retrieve current user", http.StatusInternalServerError)
		return
	}
	row := g.db.QueryRow(r.Context(), "SELECT 1 FROM clickup_auth WHERE user_id = $1", userId)
	var isAuthenticated int
	err = row.Scan(&isAuthenticated)
	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("true"))
}

func (g *ClickUpAuth) getToken(ctx context.Context, userId int) (*oauth2.Token, error) {
	var token oauth2.Token
	var expiryTimestamp sql.NullInt64
	err := g.db.QueryRow(ctx, "SELECT access_token, refresh_token, expiry FROM clickup_auth WHERE user_id = $1", userId).
		Scan(&token.AccessToken, &token.RefreshToken, &expiryTimestamp)
	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("unable to retrieve ClickUp auth token: %v", err)
	}

	if expiryTimestamp.Valid {
		token.Expiry = time.Unix(expiryTimestamp.Int64, 0)
	}
	// If expiryTimestamp is not valid, token.Expiry remains zero time which is fine
	return &token, nil
}

func (g *ClickUpAuth) getClient(ctx context.Context, userId int) (*http.Client, error) {
	token, err := g.getToken(ctx, userId)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	if token == nil {
		return nil, nil
	}
	return g.oauthConfig.Client(ctx, token), nil
}
