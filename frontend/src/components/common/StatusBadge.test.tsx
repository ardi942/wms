import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { I18nProvider } from '../../i18n/I18nProvider'
import { StatusBadge } from './index'

describe('StatusBadge', () => {
  it('renders a readable label', () => {
    render(
      <I18nProvider>
        <StatusBadge status="PARTIALLY_FULFILLED" />
      </I18nProvider>,
    )
    expect(screen.getByText('Partially Fulfilled')).toBeInTheDocument()
  })
})