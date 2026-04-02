import { describe, expect, it } from 'vitest'
import { buildAuthErrorMessage, resolveOAuthCallbackErrorMessage } from '@/utils/authError'

const messages: Record<string, string> = {
  'auth.errors.reasons.INVALID_CREDENTIALS': 'localized invalid credentials',
  'auth.errors.invalidOrExpired2FASession': 'localized 2fa session',
  'auth.emailSuffixNotAllowedWithAllowed': 'allowed: {suffixes}',
  'auth.linuxdo.errors.invalidState': 'localized linuxdo invalid state'
}

const t = (key: string, values?: Record<string, unknown>) => {
  const template = messages[key]
  if (!template) {
    return key
  }

  return Object.entries(values || {}).reduce(
    (result, [name, value]) => result.replace(`{${name}}`, String(value)),
    template
  )
}

describe('buildAuthErrorMessage', () => {
  it('prefers localized reason mapping when available', () => {
    const message = buildAuthErrorMessage(
      {
        reason: 'INVALID_CREDENTIALS',
        message: 'invalid email or password'
      },
      { fallback: 'fallback', t }
    )

    expect(message).toBe('localized invalid credentials')
  })

  it('formats email suffix metadata when present', () => {
    const message = buildAuthErrorMessage(
      {
        response: {
          data: {
            reason: 'EMAIL_SUFFIX_NOT_ALLOWED',
            metadata: {
              allowed_suffixes: '@example.com,@company.com'
            }
          }
        }
      },
      { fallback: 'fallback', t }
    )

    expect(message).toBe('allowed: @example.com, @company.com')
  })

  it('maps known literal messages when structured reason is unavailable', () => {
    const message = buildAuthErrorMessage(
      {
        message: 'Invalid or expired 2FA session'
      },
      { fallback: 'fallback', t }
    )

    expect(message).toBe('localized 2fa session')
  })

  it('falls back to raw message when no translation exists', () => {
    const message = buildAuthErrorMessage(
      {
        message: 'plain message'
      },
      { fallback: 'fallback', t }
    )

    expect(message).toBe('plain message')
  })

  it('uses fallback when no message can be extracted', () => {
    expect(buildAuthErrorMessage({}, { fallback: 'fallback', t })).toBe('fallback')
  })
})

describe('resolveOAuthCallbackErrorMessage', () => {
  it('maps provider callback codes to localized messages', () => {
    const message = resolveOAuthCallbackErrorMessage({
      provider: 'linuxdo',
      code: 'invalid_state',
      fallback: 'fallback',
      t
    })

    expect(message).toBe('localized linuxdo invalid state')
  })
})
