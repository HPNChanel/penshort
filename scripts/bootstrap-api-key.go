package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/penshort/penshort/internal/auth"
	"github.com/penshort/penshort/internal/model"
	"github.com/penshort/penshort/internal/repository"
)

type output struct {
	UserID    string   `json:"user_id"`
	Email     string   `json:"email"`
	KeyID     string   `json:"key_id"`
	Key       string   `json:"key"`
	KeyPrefix string   `json:"key_prefix"`
	Scopes    []string `json:"scopes"`
}

func main() {
	var (
		databaseURL = flag.String("database-url", os.Getenv("DATABASE_URL"), "PostgreSQL connection string")
		userID      = flag.String("user-id", "system", "User ID to own the API key")
		email       = flag.String("email", "system@penshort.local", "User email")
		name        = flag.String("name", "bootstrap", "API key name")
		scopesInput = flag.String("scopes", "admin", "Comma-separated scopes (read,write,webhook,admin)")
		format      = flag.String("format", "plain", "Output format: plain or json")
	)
	flag.Parse()

	if *databaseURL == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL is required")
		os.Exit(1)
	}

	scopes, err := parseScopes(*scopesInput)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	repo, err := repository.New(ctx, *databaseURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect database:", err)
		os.Exit(1)
	}
	defer repo.Close()

	err = ensureUser(ctx, repo, *userID, *email)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	generated, err := auth.GenerateAPIKey(auth.EnvLive)
	if err != nil {
		fmt.Fprintln(os.Stderr, "generate api key:", err)
		os.Exit(1)
	}

	apiKey := &model.APIKey{
		ID:            ulid.Make().String(),
		UserID:        *userID,
		KeyHash:       generated.Hash,
		KeyPrefix:     generated.Prefix,
		Scopes:        scopes,
		RateLimitTier: model.TierUnlimited,
		Name:          *name,
		CreatedAt:     time.Now().UTC(),
	}

	if err := repo.CreateAPIKey(ctx, apiKey); err != nil {
		fmt.Fprintln(os.Stderr, "create api key:", err)
		os.Exit(1)
	}

	out := output{
		UserID:    *userID,
		Email:     *email,
		KeyID:     apiKey.ID,
		Key:       generated.Plaintext,
		KeyPrefix: apiKey.KeyPrefix,
		Scopes:    scopes,
	}

	switch strings.ToLower(*format) {
	case "plain":
		fmt.Println(out.Key)
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(out)
	default:
		fmt.Fprintln(os.Stderr, "invalid format; use plain or json")
		os.Exit(1)
	}
}

func parseScopes(input string) ([]string, error) {
	if strings.TrimSpace(input) == "" {
		return []string{model.ScopeAdmin}, nil
	}
	parts := strings.Split(input, ",")
	scopes := make([]string, 0, len(parts))
	for _, part := range parts {
		scope := strings.TrimSpace(part)
		if scope == "" {
			continue
		}
		if !isValidScope(scope) {
			return nil, fmt.Errorf("invalid scope: %s", scope)
		}
		scopes = append(scopes, scope)
	}
	if len(scopes) == 0 {
		scopes = []string{model.ScopeAdmin}
	}
	return scopes, nil
}

func isValidScope(scope string) bool {
	for _, allowed := range model.ValidScopes {
		if scope == allowed {
			return true
		}
	}
	return false
}

func ensureUser(ctx context.Context, repo *repository.Repository, userID, email string) error {
	existing, err := repo.GetUserByID(ctx, userID)
	if err == nil {
		if existing.Email != email {
			return fmt.Errorf("user %s exists with different email: %s", userID, existing.Email)
		}
		return nil
	}

	byEmail, err := repo.GetUserByEmail(ctx, email)
	if err == nil {
		if byEmail.ID != userID {
			return fmt.Errorf("email %s already used by user %s", email, byEmail.ID)
		}
		return nil
	}

	user := &model.User{
		ID:        userID,
		Email:     email,
		CreatedAt: time.Now().UTC(),
	}
	if err := repo.CreateUser(ctx, user); err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}
