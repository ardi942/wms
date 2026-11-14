import { describe, expect, it } from 'vitest'
import { formatCurrency, formatQty, setFormatLocale, statusLabel, statusTone } from './format'

describe('format helpers', () => {
  it('formats currency as IDR', () => {
    const s = formatCurrency(1234.5)
    expect(s.startsWith('Rp')).toBe(true)
    expect(s).toContain('1.235')
  })
  it('formats integers with the active locale', () => {
    setFormatLocale('en-US')
    expect(formatQty(12000)).toBe('12,000')
    setFormatLocale('de-DE')
    expect(formatQty(12000)).toBe('12.000')
    setFormatLocale('en-US')
  })
  it('labels statuses', () => {
    expect(statusLabel('PARTIALLY_FULFILLED')).toBe('Partially Fulfilled')
    expect(statusTone('REJECTED')).toBe('red')
  })
})
