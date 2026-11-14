import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import { dictionaries, type Lang } from './dictionary'
import { setFormatLocale } from '../lib/format'
import { cn } from '../lib/utils'

const LANG_KEY = 'wms.lang'
const supported: Lang[] = ['en', 'id', 'zh']
const localeMap: Record<Lang, string> = { en: 'en-US', id: 'id-ID', zh: 'zh-CN' }

function isLang(value: string | null): value is Lang {
  return value !== null && supported.includes(value as Lang)
}

function resolve(obj: unknown, path: string): string | undefined {
  let cur: unknown = obj
  for (const part of path.split('.')) {
    if (typeof cur !== 'object' || cur === null) return undefined
    cur = (cur as Record<string, unknown>)[part]
  }
  return typeof cur === 'string' ? cur : undefined
}

function interpolate(text: string, params?: Record<string, string | number>): string {
  if (!params) return text
  return text.replace(/\{(\w+)\}/g, (_, key: string) =>
    params[key] !== undefined ? String(params[key]) : `{${key}}`,
  )
}

interface I18nValue {
  lang: Lang
  setLang: (lang: Lang) => void
  t: (key: string, params?: Record<string, string | number>) => string
}

const I18nContext = createContext<I18nValue | null>(null)

export function I18nProvider({ children }: { children: ReactNode }) {
  const [lang, setLangState] = useState<Lang>(() => {
    const stored = typeof localStorage !== 'undefined' ? localStorage.getItem(LANG_KEY) : null
    return isLang(stored) ? stored : 'en'
  })

  const setLang = useCallback((next: Lang) => {
    setLangState(next)
    if (typeof localStorage !== 'undefined') localStorage.setItem(LANG_KEY, next)
  }, [])

  useEffect(() => {
    setFormatLocale(localeMap[lang])
    document.documentElement.lang = lang
  }, [lang])

  const t = useCallback(
    (key: string, params?: Record<string, string | number>) => {
      const text = resolve(dictionaries[lang], key) ?? resolve(dictionaries.en, key) ?? key
      return interpolate(text, params)
    },
    [lang],
  )

  const value = useMemo(() => ({ lang, setLang, t }), [lang, setLang, t])
  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>
}

export function useT() {
  const ctx = useContext(I18nContext)
  if (!ctx) throw new Error('useT must be used inside I18nProvider')
  return ctx
}

export function LanguageSwitcher({ className }: { className?: string }) {
  const { lang, setLang, t } = useT()
  return (
    <select
      aria-label={t('common.language')}
      value={lang}
      onChange={(e) => setLang(e.target.value as Lang)}
      className={cn(
        'h-8 rounded-md border border-slate-300 bg-white px-2 text-xs text-slate-700 outline-none focus:border-brand-500',
        className,
      )}
    >
      <option value="en">English</option>
      <option value="id">Bahasa Indonesia</option>
      <option value="zh">中文</option>
    </select>
  )
}