type TranslateLike = (key: string, values?: Record<string, unknown>) => string

type ErrorMetadata = Record<string, string>

interface ErrorPayloadLike {
  detail?: string
  message?: string
  code?: string | number
  reason?: string
  error?: string
  metadata?: ErrorMetadata
}

interface APIErrorLike {
  message?: string
  code?: string | number
  reason?: string
  error?: string
  metadata?: ErrorMetadata
  response?: {
    data?: ErrorPayloadLike
  }
}

interface AuthErrorDetails {
  message: string
  reason: string
  metadata: ErrorMetadata
}

type OAuthProvider = 'linuxdo' | 'discord'

const AUTH_MESSAGE_KEY_MAP: Record<string, string> = {
  'invalid or expired 2fa session': 'auth.errors.invalidOrExpired2FASession',
  'backend mode is active. only admin login is allowed.': 'auth.errors.backendModeAdminOnly',
  'network error. please check your connection.': 'auth.errors.networkError',
  'session expired. please log in again.': 'auth.errors.sessionExpired'
}

const OAUTH_CALLBACK_CODE_KEY_MAP: Record<OAuthProvider, Record<string, string>> = {
  linuxdo: {
    provider_error: 'auth.linuxdo.errors.providerError',
    missing_params: 'auth.linuxdo.errors.missingParams',
    invalid_state: 'auth.linuxdo.errors.invalidState',
    missing_verifier: 'auth.linuxdo.errors.missingVerifier',
    config_error: 'auth.linuxdo.errors.configError',
    token_exchange_failed: 'auth.linuxdo.errors.tokenExchangeFailed',
    userinfo_failed: 'auth.linuxdo.errors.userinfoFailed',
    login_failed: 'auth.linuxdo.errors.loginFailed'
  },
  discord: {
    provider_error: 'auth.discord.errors.providerError',
    missing_params: 'auth.discord.errors.missingParams',
    invalid_state: 'auth.discord.errors.invalidState',
    missing_verifier: 'auth.discord.errors.missingVerifier',
    config_error: 'auth.discord.errors.configError',
    token_exchange_failed: 'auth.discord.errors.tokenExchangeFailed',
    userinfo_failed: 'auth.discord.errors.userinfoFailed',
    guild_verify_failed: 'auth.discord.errors.guildVerifyFailed',
    login_failed: 'auth.discord.errors.loginFailed'
  }
}

function isNonEmptyString(value: unknown): value is string {
  return typeof value === 'string' && value.trim() !== ''
}

function normalizeString(value: unknown): string {
  return isNonEmptyString(value) ? value.trim() : ''
}

function normalizeCode(value: unknown): string {
  if (typeof value !== 'string') {
    return ''
  }

  const normalized = value.trim()
  if (!normalized || /^\d+$/.test(normalized)) {
    return ''
  }

  return normalized
}

function normalizeMessageKey(message: string): string {
  return message.trim().toLowerCase().replace(/\s+/g, ' ')
}

function extractAuthErrorDetails(error: unknown): AuthErrorDetails {
  const err = (error || {}) as APIErrorLike
  const payload = (err.response?.data || {}) as ErrorPayloadLike

  const message =
    normalizeString(payload.detail) || normalizeString(payload.message) || normalizeString(err.message)

  const reason =
    normalizeString(payload.reason) ||
    normalizeString(err.reason) ||
    normalizeString(payload.error) ||
    normalizeCode(payload.code) ||
    normalizeString(err.error) ||
    normalizeCode(err.code)

  return {
    message,
    reason,
    metadata: payload.metadata || err.metadata || {}
  }
}

function translateIfExists(
  t: TranslateLike | undefined,
  key: string,
  values?: Record<string, unknown>
): string {
  if (!t) {
    return ''
  }

  const translated = t(key, values)
  return translated !== key ? translated : ''
}

function resolveReasonMessage(details: AuthErrorDetails, t?: TranslateLike): string {
  if (!t || !details.reason) {
    return ''
  }

  if (details.reason === 'EMAIL_SUFFIX_NOT_ALLOWED' && isNonEmptyString(details.metadata.allowed_suffixes)) {
    return (
      translateIfExists(t, 'auth.emailSuffixNotAllowedWithAllowed', {
        suffixes: details.metadata.allowed_suffixes
          .split(',')
          .map((item) => item.trim())
          .filter(Boolean)
          .join(', ')
      }) || translateIfExists(t, 'auth.emailSuffixNotAllowed')
    )
  }

  return translateIfExists(t, `auth.errors.reasons.${details.reason}`)
}

function resolveMessageLiteral(message: string, t?: TranslateLike): string {
  if (!t || !message) {
    return ''
  }

  const key = AUTH_MESSAGE_KEY_MAP[normalizeMessageKey(message)]
  return key ? translateIfExists(t, key) : ''
}

export function buildAuthErrorMessage(
  error: unknown,
  options: {
    fallback: string
    t?: TranslateLike
  }
): string {
  const { fallback, t } = options
  const details = extractAuthErrorDetails(error)

  return resolveReasonMessage(details, t) || resolveMessageLiteral(details.message, t) || details.message || fallback
}

export function resolveOAuthCallbackErrorMessage(options: {
  provider: OAuthProvider
  code?: string | null
  reason?: string | null
  description?: string | null
  fallback: string
  t: TranslateLike
}): string {
  const { provider, code, reason, description, fallback, t } = options
  const normalizedCode = normalizeString(code)
  const details = extractAuthErrorDetails({
    message: description || '',
    reason: reason || '',
    error: normalizedCode
  })

  const reasonMessage =
    translateIfExists(t, `auth.${provider}.errors.reasons.${details.reason}`) ||
    resolveReasonMessage(details, t)

  if (reasonMessage) {
    return reasonMessage
  }

  const codeKey = OAUTH_CALLBACK_CODE_KEY_MAP[provider][normalizedCode]
  if (codeKey) {
    const codeMessage = translateIfExists(t, codeKey)
    if (codeMessage) {
      return codeMessage
    }
  }

  return (
    resolveMessageLiteral(details.message, t) ||
    details.message ||
    details.reason ||
    normalizedCode ||
    fallback
  )
}
