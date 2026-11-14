import { describe, expect, it } from 'vitest'
import { dictionaries } from './dictionary'

function flatten(value: unknown, prefix = ''): string[] {
  if (typeof value !== 'object' || value === null) return [prefix]
  return Object.entries(value as Record<string, unknown>).flatMap(([key, child]) =>
    flatten(child, prefix ? `${prefix}.${key}` : key),
  )
}

describe('i18n dictionaries', () => {
  it('has identical keys in every language', () => {
    const en = flatten(dictionaries.en).sort()
    expect(flatten(dictionaries.id).sort()).toEqual(en)
    expect(flatten(dictionaries.zh).sort()).toEqual(en)
  })
})