package handler

import (
	"net/url"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestDiscordParseUserInfoParsesIDAndUsername(t *testing.T) {
	cfg := config.DiscordConnectConfig{
		UserInfoURL: "https://discord.com/api/v10/users/@me",
	}

	email, username, subject, err := discordParseUserInfo(`{"id":"1234567890","username":"alice"}`, cfg)
	require.NoError(t, err)
	require.Equal(t, "1234567890", subject)
	require.Equal(t, "alice", username)
	require.Equal(t, "discord-1234567890@discord-connect.invalid", email)
}

func TestDiscordParseUserInfoUsesGlobalNameAndEmail(t *testing.T) {
	cfg := config.DiscordConnectConfig{
		UserInfoURL: "https://discord.com/api/v10/users/@me",
	}

	email, username, subject, err := discordParseUserInfo(`{"id":"1234567890","global_name":"Alice A","email":"alice@example.com"}`, cfg)
	require.NoError(t, err)
	require.Equal(t, "1234567890", subject)
	require.Equal(t, "Alice A", username)
	require.Equal(t, "alice@example.com", email)
}

func TestDiscordParseUserInfoRejectsUnsafeSubject(t *testing.T) {
	cfg := config.DiscordConnectConfig{
		UserInfoURL: "https://discord.com/api/v10/users/@me",
	}

	_, _, _, err := discordParseUserInfo(`{"id":"abc123"}`, cfg)
	require.Error(t, err)

	tooLong := strings.Repeat("1", discordOAuthMaxSubjectLen+1)
	_, _, _, err = discordParseUserInfo(`{"id":"`+tooLong+`"}`, cfg)
	require.Error(t, err)
}

func TestBuildDiscordAuthorizeURLIncludesPKCE(t *testing.T) {
	cfg := config.DiscordConnectConfig{
		AuthorizeURL: "https://discord.com/oauth2/authorize",
		ClientID:     "client-1",
		Scopes:       "identify email",
		UsePKCE:      true,
	}

	authURL, err := buildDiscordAuthorizeURL(cfg, "state-1", "challenge-1", "https://example.com/callback")
	require.NoError(t, err)

	u, err := url.Parse(authURL)
	require.NoError(t, err)
	q := u.Query()
	require.Equal(t, "code", q.Get("response_type"))
	require.Equal(t, "client-1", q.Get("client_id"))
	require.Equal(t, "identify email", q.Get("scope"))
	require.Equal(t, "state-1", q.Get("state"))
	require.Equal(t, "challenge-1", q.Get("code_challenge"))
	require.Equal(t, "S256", q.Get("code_challenge_method"))
}
