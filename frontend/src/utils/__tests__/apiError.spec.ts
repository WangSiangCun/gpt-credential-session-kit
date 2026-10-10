import { describe, expect, it } from 'vitest'
import { extractI18nErrorMessage } from '@/utils/apiError'

describe('extractI18nErrorMessage', () => {
  const messages: Record<string, string> = {
    'redeem.errors.REDEEM_BATCH_ALREADY_USED':
      '你已经兑换过这一批兑换码，每位用户仅可兑换一次。',
  }

  const t = (key: string) => messages[key] ?? key

  it('uses the localized reason returned by the redeem API', () => {
    const message = extractI18nErrorMessage(
      {
        status: 409,
        reason: 'REDEEM_BATCH_ALREADY_USED',
        message: 'you have already redeemed a code from this batch',
      },
      t,
      'redeem.errors',
      '兑换失败，请检查兑换码后重试。',
    )

    expect(message).toBe('你已经兑换过这一批兑换码，每位用户仅可兑换一次。')
  })

  it('keeps the backend message when the reason has no translation', () => {
    const message = extractI18nErrorMessage(
      {
        status: 409,
        reason: 'UNKNOWN_REDEEM_ERROR',
        message: 'specific backend reason',
      },
      t,
      'redeem.errors',
      'generic fallback',
    )

    expect(message).toBe('specific backend reason')
  })
})
