package handler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/oauth"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/imroc/req/v3"
	"github.com/tidwall/gjson"
)

const (
	discordOAuthCookiePath        = "/api/v1/auth/oauth/discord"
	discordOAuthStateCookieName   = "discord_oauth_state"
	discordOAuthVerifierCookie    = "discord_oauth_verifier"
	discordOAuthRedirectCookie    = "discord_oauth_redirect"
	discordOAuthCookieMaxAgeSec   = 10 * 60 // 10 minutes
	discordOAuthDefaultRedirectTo = "/dashboard"
	discordOAuthDefaultFrontendCB = "/auth/discord/callback"

	discordOAuthReasonGuildVerifyFailed       = "DISCORD_GUILD_VERIFY_FAILED"
	discordOAuthReasonRequiredGuildMembership = "DISCORD_REQUIRED_GUILD_MEMBERSHIP"
	discordOAuthReasonRequiredRoleMissing     = "DISCORD_REQUIRED_ROLE_MISSING"

	discordOAuthMaxSubjectLen = 64 - len("discord-")
)

type discordTokenResponse = linuxDoTokenResponse

type discordTokenExchangeError struct {
	StatusCode          int
	ProviderError       string
	ProviderDescription string
	Body                string
}

func (e *discordTokenExchangeError) Error() string {
	if e == nil {
		return ""
	}
	parts := []string{fmt.Sprintf("token exchange status=%d", e.StatusCode)}
	if strings.TrimSpace(e.ProviderError) != "" {
		parts = append(parts, "error="+strings.TrimSpace(e.ProviderError))
	}
	if strings.TrimSpace(e.ProviderDescription) != "" {
		parts = append(parts, "error_description="+strings.TrimSpace(e.ProviderDescription))
	}
	return strings.Join(parts, " ")
}

// DiscordOAuthStart 启动 Discord OAuth 登录流程。
// GET /api/v1/auth/oauth/discord/start?redirect=/dashboard
func (h *AuthHandler) DiscordOAuthStart(c *gin.Context) {
	cfg, err := h.getDiscordOAuthConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	state, err := oauth.GenerateState()
	if err != nil {
		response.ErrorFrom(c, infraerrors.InternalServer("OAUTH_STATE_GEN_FAILED", "failed to generate oauth state").WithCause(err))
		return
	}

	redirectTo := sanitizeFrontendRedirectPath(c.Query("redirect"))
	if redirectTo == "" {
		redirectTo = discordOAuthDefaultRedirectTo
	}

	secureCookie := isRequestHTTPS(c)
	setDiscordCookie(c, discordOAuthStateCookieName, encodeCookieValue(state), discordOAuthCookieMaxAgeSec, secureCookie)
	setDiscordCookie(c, discordOAuthRedirectCookie, encodeCookieValue(redirectTo), discordOAuthCookieMaxAgeSec, secureCookie)

	codeChallenge := ""
	if cfg.UsePKCE {
		verifier, err := oauth.GenerateCodeVerifier()
		if err != nil {
			response.ErrorFrom(c, infraerrors.InternalServer("OAUTH_PKCE_GEN_FAILED", "failed to generate pkce verifier").WithCause(err))
			return
		}
		codeChallenge = oauth.GenerateCodeChallenge(verifier)
		setDiscordCookie(c, discordOAuthVerifierCookie, encodeCookieValue(verifier), discordOAuthCookieMaxAgeSec, secureCookie)
	}

	redirectURI := strings.TrimSpace(cfg.RedirectURL)
	if redirectURI == "" {
		response.ErrorFrom(c, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth redirect url not configured"))
		return
	}

	authURL, err := buildDiscordAuthorizeURL(cfg, state, codeChallenge, redirectURI)
	if err != nil {
		response.ErrorFrom(c, infraerrors.InternalServer("OAUTH_BUILD_URL_FAILED", "failed to build oauth authorization url").WithCause(err))
		return
	}

	c.Redirect(http.StatusFound, authURL)
}

// DiscordOAuthCallback 处理 OAuth 回调：创建/登录用户，然后重定向到前端。
// GET /api/v1/auth/oauth/discord/callback?code=...&state=...
func (h *AuthHandler) DiscordOAuthCallback(c *gin.Context) {
	cfg, cfgErr := h.getDiscordOAuthConfig(c.Request.Context())
	if cfgErr != nil {
		response.ErrorFrom(c, cfgErr)
		return
	}

	frontendCallback := strings.TrimSpace(cfg.FrontendRedirectURL)
	if frontendCallback == "" {
		frontendCallback = discordOAuthDefaultFrontendCB
	}

	if providerErr := strings.TrimSpace(c.Query("error")); providerErr != "" {
		redirectOAuthError(c, frontendCallback, "provider_error", providerErr, c.Query("error_description"))
		return
	}

	code := strings.TrimSpace(c.Query("code"))
	state := strings.TrimSpace(c.Query("state"))
	if code == "" || state == "" {
		redirectOAuthError(c, frontendCallback, "missing_params", "missing code/state", "")
		return
	}

	secureCookie := isRequestHTTPS(c)
	defer func() {
		clearDiscordCookie(c, discordOAuthStateCookieName, secureCookie)
		clearDiscordCookie(c, discordOAuthVerifierCookie, secureCookie)
		clearDiscordCookie(c, discordOAuthRedirectCookie, secureCookie)
	}()

	expectedState, err := readCookieDecoded(c, discordOAuthStateCookieName)
	if err != nil || expectedState == "" || state != expectedState {
		redirectOAuthError(c, frontendCallback, "invalid_state", "invalid oauth state", "")
		return
	}

	redirectTo, _ := readCookieDecoded(c, discordOAuthRedirectCookie)
	redirectTo = sanitizeFrontendRedirectPath(redirectTo)
	if redirectTo == "" {
		redirectTo = discordOAuthDefaultRedirectTo
	}

	codeVerifier := ""
	if cfg.UsePKCE {
		codeVerifier, _ = readCookieDecoded(c, discordOAuthVerifierCookie)
		if codeVerifier == "" {
			redirectOAuthError(c, frontendCallback, "missing_verifier", "missing pkce verifier", "")
			return
		}
	}

	redirectURI := strings.TrimSpace(cfg.RedirectURL)
	if redirectURI == "" {
		redirectOAuthError(c, frontendCallback, "config_error", "oauth redirect url not configured", "")
		return
	}

	tokenResp, err := discordExchangeCode(c.Request.Context(), cfg, code, redirectURI, codeVerifier)
	if err != nil {
		description := ""
		var exchangeErr *discordTokenExchangeError
		if errors.As(err, &exchangeErr) && exchangeErr != nil {
			log.Printf(
				"[Discord OAuth] token exchange failed: status=%d provider_error=%q provider_description=%q body=%s",
				exchangeErr.StatusCode,
				exchangeErr.ProviderError,
				exchangeErr.ProviderDescription,
				truncateLogValue(exchangeErr.Body, 2048),
			)
			description = exchangeErr.Error()
		} else {
			log.Printf("[Discord OAuth] token exchange failed: %v", err)
			description = err.Error()
		}
		redirectOAuthError(c, frontendCallback, "token_exchange_failed", "failed to exchange oauth code", singleLine(description))
		return
	}

	email, username, subject, err := discordFetchUserInfo(c.Request.Context(), cfg, tokenResp)
	if err != nil {
		log.Printf("[Discord OAuth] userinfo fetch failed: %v", err)
		redirectOAuthError(c, frontendCallback, "userinfo_failed", "failed to fetch user info", "")
		return
	}

	// Guild / Role 校验（在 userinfo 之后、login/register 之前）
	if cfg.GuildVerifyEnabled && strings.TrimSpace(cfg.RequiredGuildID) != "" {
		if err := discordVerifyGuildAndRoles(c.Request.Context(), cfg, tokenResp); err != nil {
			reason := strings.TrimSpace(infraerrors.Reason(err))
			description := strings.TrimSpace(infraerrors.Message(err))
			if reason == "" {
				reason = discordOAuthReasonGuildVerifyFailed
			}
			if description == "" || description == infraerrors.UnknownMessage {
				description = "failed to verify Discord server membership"
			}
			log.Printf("[Discord OAuth] guild/role verification failed: %v", err)
			redirectOAuthError(c, frontendCallback, "guild_verify_failed", reason, description)
			return
		}
	}

	// 安全考虑：不要把第三方返回的 email 直接映射到本地账号（可能与本地邮箱用户冲突导致账号被接管）。
	// 统一使用基于 subject 的稳定合成邮箱来做账号绑定。
	if subject != "" {
		email = discordSyntheticEmail(subject)
	}

	// 传入空邀请码；如果需要邀请码，服务层返回 ErrOAuthInvitationRequired
	tokenPair, _, err := h.authService.LoginOrRegisterOAuthWithTokenPair(c.Request.Context(), email, username, "")
	if err != nil {
		if errors.Is(err, service.ErrOAuthInvitationRequired) {
			pendingToken, tokenErr := h.authService.CreatePendingOAuthToken(email, username)
			if tokenErr != nil {
				redirectOAuthError(c, frontendCallback, "login_failed", "service_error", "")
				return
			}
			fragment := url.Values{}
			fragment.Set("error", "invitation_required")
			fragment.Set("pending_oauth_token", pendingToken)
			fragment.Set("redirect", redirectTo)
			redirectWithFragment(c, frontendCallback, fragment)
			return
		}
		redirectOAuthError(c, frontendCallback, "login_failed", infraerrors.Reason(err), infraerrors.Message(err))
		return
	}

	fragment := url.Values{}
	fragment.Set("access_token", tokenPair.AccessToken)
	fragment.Set("refresh_token", tokenPair.RefreshToken)
	fragment.Set("expires_in", fmt.Sprintf("%d", tokenPair.ExpiresIn))
	fragment.Set("token_type", "Bearer")
	fragment.Set("redirect", redirectTo)
	redirectWithFragment(c, frontendCallback, fragment)
}

type completeDiscordOAuthRequest struct {
	PendingOAuthToken string `json:"pending_oauth_token" binding:"required"`
	InvitationCode    string `json:"invitation_code"     binding:"required"`
}

// CompleteDiscordOAuthRegistration completes a pending OAuth registration by validating
// the invitation code and creating the user account.
// POST /api/v1/auth/oauth/discord/complete-registration
func (h *AuthHandler) CompleteDiscordOAuthRegistration(c *gin.Context) {
	var req completeDiscordOAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "INVALID_REQUEST", "message": err.Error()})
		return
	}

	email, username, err := h.authService.VerifyPendingOAuthToken(req.PendingOAuthToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "INVALID_TOKEN", "message": "invalid or expired registration token"})
		return
	}

	tokenPair, _, err := h.authService.LoginOrRegisterOAuthWithTokenPair(c.Request.Context(), email, username, req.InvitationCode)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"expires_in":    tokenPair.ExpiresIn,
		"token_type":    "Bearer",
	})
}

func (h *AuthHandler) getDiscordOAuthConfig(ctx context.Context) (config.DiscordConnectConfig, error) {
	if h != nil && h.settingSvc != nil {
		return h.settingSvc.GetDiscordConnectOAuthConfig(ctx)
	}
	if h == nil || h.cfg == nil {
		return config.DiscordConnectConfig{}, infraerrors.ServiceUnavailable("CONFIG_NOT_READY", "config not loaded")
	}
	if !h.cfg.Discord.Enabled {
		return config.DiscordConnectConfig{}, infraerrors.NotFound("OAUTH_DISABLED", "oauth login is disabled")
	}
	return h.cfg.Discord, nil
}

func discordExchangeCode(
	ctx context.Context,
	cfg config.DiscordConnectConfig,
	code string,
	redirectURI string,
	codeVerifier string,
) (*discordTokenResponse, error) {
	client := req.C().SetTimeout(30 * time.Second)

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", cfg.ClientID)
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	if cfg.UsePKCE {
		form.Set("code_verifier", codeVerifier)
	}

	r := client.R().
		SetContext(ctx).
		SetHeader("Accept", "application/json")

	switch strings.ToLower(strings.TrimSpace(cfg.TokenAuthMethod)) {
	case "", "client_secret_post":
		form.Set("client_secret", cfg.ClientSecret)
	case "client_secret_basic":
		r.SetBasicAuth(cfg.ClientID, cfg.ClientSecret)
	case "none":
	default:
		return nil, fmt.Errorf("unsupported token_auth_method: %s", cfg.TokenAuthMethod)
	}

	resp, err := r.SetFormDataFromValues(form).Post(cfg.TokenURL)
	if err != nil {
		return nil, fmt.Errorf("request token: %w", err)
	}
	body := strings.TrimSpace(resp.String())
	if !resp.IsSuccessState() {
		providerErr, providerDesc := parseOAuthProviderError(body)
		return nil, &discordTokenExchangeError{
			StatusCode:          resp.StatusCode,
			ProviderError:       providerErr,
			ProviderDescription: providerDesc,
			Body:                body,
		}
	}

	tokenResp, ok := parseLinuxDoTokenResponse(body)
	if !ok || strings.TrimSpace(tokenResp.AccessToken) == "" {
		return nil, &discordTokenExchangeError{
			StatusCode: resp.StatusCode,
			Body:       body,
		}
	}
	if strings.TrimSpace(tokenResp.TokenType) == "" {
		tokenResp.TokenType = "Bearer"
	}
	return tokenResp, nil
}

func discordFetchUserInfo(
	ctx context.Context,
	cfg config.DiscordConnectConfig,
	token *discordTokenResponse,
) (email string, username string, subject string, err error) {
	client := req.C().SetTimeout(30 * time.Second)
	authorization, err := buildBearerAuthorization(token.TokenType, token.AccessToken)
	if err != nil {
		return "", "", "", fmt.Errorf("invalid token for userinfo request: %w", err)
	}

	resp, err := client.R().
		SetContext(ctx).
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", authorization).
		Get(cfg.UserInfoURL)
	if err != nil {
		return "", "", "", fmt.Errorf("request userinfo: %w", err)
	}
	if !resp.IsSuccessState() {
		return "", "", "", fmt.Errorf("userinfo status=%d", resp.StatusCode)
	}

	return discordParseUserInfo(resp.String(), cfg)
}

func discordParseUserInfo(body string, cfg config.DiscordConnectConfig) (email string, username string, subject string, err error) {
	email = firstNonEmpty(
		getGJSON(body, cfg.UserInfoEmailPath),
		getGJSON(body, "email"),
	)
	username = firstNonEmpty(
		getGJSON(body, cfg.UserInfoUsernamePath),
		getGJSON(body, "global_name"),
		getGJSON(body, "username"),
		getGJSON(body, "display_name"),
	)
	subject = firstNonEmpty(
		getGJSON(body, cfg.UserInfoIDPath),
		getGJSON(body, "id"),
		getGJSON(body, "sub"),
	)

	subject = strings.TrimSpace(subject)
	if subject == "" {
		return "", "", "", errors.New("userinfo missing id field")
	}
	if !isSafeDiscordSubject(subject) {
		return "", "", "", errors.New("userinfo returned invalid id field")
	}

	email = strings.TrimSpace(email)
	if email == "" {
		email = discordSyntheticEmail(subject)
	}

	username = strings.TrimSpace(username)
	if username == "" {
		username = "discord_" + subject
	}

	return email, username, subject, nil
}

func buildDiscordAuthorizeURL(cfg config.DiscordConnectConfig, state string, codeChallenge string, redirectURI string) (string, error) {
	u, err := url.Parse(cfg.AuthorizeURL)
	if err != nil {
		return "", fmt.Errorf("parse authorize_url: %w", err)
	}

	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", cfg.ClientID)
	q.Set("redirect_uri", redirectURI)
	// 计算最终 scope：基础 scope + guild 校验需要的额外 scope
	finalScopes := strings.TrimSpace(cfg.Scopes)
	if cfg.GuildVerifyEnabled && strings.TrimSpace(cfg.RequiredGuildID) != "" {
		finalScopes = ensureDiscordGuildScopes(finalScopes)
	}
	if finalScopes != "" {
		q.Set("scope", finalScopes)
	}
	q.Set("state", state)
	if cfg.UsePKCE {
		q.Set("code_challenge", codeChallenge)
		q.Set("code_challenge_method", "S256")
	}

	u.RawQuery = q.Encode()
	return u.String(), nil
}

func setDiscordCookie(c *gin.Context, name string, value string, maxAgeSec int, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     discordOAuthCookiePath,
		MaxAge:   maxAgeSec,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearDiscordCookie(c *gin.Context, name string, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     discordOAuthCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func isSafeDiscordSubject(subject string) bool {
	subject = strings.TrimSpace(subject)
	if subject == "" || len(subject) > discordOAuthMaxSubjectLen {
		return false
	}
	for _, r := range subject {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func discordSyntheticEmail(subject string) string {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return ""
	}
	return "discord-" + subject + service.DiscordConnectSyntheticEmailDomain
}

// ---------------------------------------------------------------------------
// Guild / Role verification
// ---------------------------------------------------------------------------

// discordGuildScopes are required when guild/role verification is enabled.
var discordGuildScopes = []string{"guilds", "guilds.members.read"}

// ensureDiscordGuildScopes appends guilds + guilds.members.read if not already present.
func ensureDiscordGuildScopes(scopes string) string {
	existing := strings.Fields(scopes)
	set := make(map[string]struct{}, len(existing))
	for _, s := range existing {
		set[strings.TrimSpace(s)] = struct{}{}
	}
	for _, required := range discordGuildScopes {
		if _, ok := set[required]; !ok {
			existing = append(existing, required)
		}
	}
	return strings.Join(existing, " ")
}

// discordVerifyGuildAndRoles checks that the user belongs to the required guild
// and (optionally) has at least one of the required roles.
func discordVerifyGuildAndRoles(
	ctx context.Context,
	cfg config.DiscordConnectConfig,
	token *discordTokenResponse,
) error {
	guildID := strings.TrimSpace(cfg.RequiredGuildID)
	if guildID == "" {
		return nil
	}

	authorization, err := buildBearerAuthorization(token.TokenType, token.AccessToken)
	if err != nil {
		return fmt.Errorf("invalid token: %w", err)
	}
	client := req.C().SetTimeout(10 * time.Second)

	// Step 1 – verify user is in the guild via GET /users/@me/guilds
	guildsResp, err := client.R().
		SetContext(ctx).
		SetHeader("Authorization", authorization).
		SetHeader("Accept", "application/json").
		Get("https://discord.com/api/v10/users/@me/guilds")
	if err != nil {
		return fmt.Errorf("failed to fetch user guilds: %w", err)
	}
	if !guildsResp.IsSuccessState() {
		return fmt.Errorf("user guilds request failed (status=%d), possibly missing 'guilds' scope", guildsResp.StatusCode)
	}

	// guilds response is a JSON array; search for our guild
	guildsBody := guildsResp.String()
	found := false
	gjson.Parse(guildsBody).ForEach(func(_, v gjson.Result) bool {
		if v.Get("id").String() == guildID {
			found = true
			return false // break
		}
		return true
	})
	if !found {
		return infraerrors.Forbidden(
			discordOAuthReasonRequiredGuildMembership,
			"user is not a member of the required Discord server",
		)
	}

	// Step 2 – optional role verification
	requiredRoleIDs := parseCommaSeparatedIDs(cfg.RequiredRoleIDs)
	if len(requiredRoleIDs) == 0 {
		return nil // no role requirement, guild membership is sufficient
	}

	// GET /users/@me/guilds/{guildId}/member
	memberResp, err := client.R().
		SetContext(ctx).
		SetHeader("Authorization", authorization).
		SetHeader("Accept", "application/json").
		Get(fmt.Sprintf("https://discord.com/api/v10/users/@me/guilds/%s/member", guildID))
	if err != nil {
		return fmt.Errorf("failed to fetch guild member: %w", err)
	}
	if !memberResp.IsSuccessState() {
		return fmt.Errorf("guild member request failed (status=%d), possibly missing 'guilds.members.read' scope", memberResp.StatusCode)
	}

	memberBody := memberResp.String()
	userRoles := make(map[string]struct{})
	gjson.Get(memberBody, "roles").ForEach(func(_, v gjson.Result) bool {
		userRoles[v.String()] = struct{}{}
		return true
	})

	for _, rid := range requiredRoleIDs {
		if _, ok := userRoles[rid]; ok {
			return nil // user has at least one required role
		}
	}
	return infraerrors.Forbidden(
		discordOAuthReasonRequiredRoleMissing,
		"user does not have any of the required roles in the Discord server",
	)
}

// parseCommaSeparatedIDs splits a comma-separated string of IDs, trims whitespace,
// and returns non-empty entries.
func parseCommaSeparatedIDs(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
