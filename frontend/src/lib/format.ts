const currency = new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 })

let activeLocale = 'en-US'
let integer = new Intl.NumberFormat(activeLocale)
let dateTime = new Intl.DateTimeFormat(activeLocale, { dateStyle: 'medium', timeStyle: 'short' })
let dateOnly = new Intl.DateTimeFormat(activeLocale, { dateStyle: 'medium' })

export function setFormatLocale(locale: string): void {
  activeLocale = locale
  integer = new Intl.NumberFormat(locale)
  dateTime = new Intl.DateTimeFormat(locale, { dateStyle: 'medium', timeStyle: 'short' })
  dateOnly = new Intl.DateTimeFormat(locale, { dateStyle: 'medium' })
}

export function getFormatLocale(): string {
  return activeLocale
}

export function formatCurrency(value: number | null | undefined): string {
  if (value === null || value === undefined || Number.isNaN(value)) return '—'
  return currency.format(value)
}

export function formatQty(value: number | null | undefined): string {
  if (value === null || value === undefined || Number.isNaN(value)) return '—'
  return integer.format(value)
}

export function formatDateTime(value: string | null | undefined): string {
  if (!value) return '—'
  return dateTime.format(new Date(value))
}

export function formatDate(value: string | null | undefined): string {
  if (!value) return '—'
  return dateOnly.format(new Date(value))
}

export function todayInput(): string {
  return new Date().toISOString().slice(0, 10)
}

export function statusLabel(status: string): string {
  return status
    .toLowerCase()
    .split('_')
    .map((p) => p.charAt(0).toUpperCase() + p.slice(1))
    .join(' ')
}

export function statusTone(status: string): 'amber' | 'blue' | 'green' | 'red' | 'gray' {
  switch (status) {
    case 'PENDING':
      return 'amber'
    case 'PARTIALLY_FULFILLED':
      return 'blue'
    case 'FULFILLED':
    case 'CONFIRMED':
      return 'green'
    case 'REJECTED':
      return 'red'
    default:
      return 'gray'
  }
}
