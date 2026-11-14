import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { StatusBadge } from './index'

describe('StatusBadge', () => {
  it('renders a readable label', () => {
    render(<StatusBadge status="PARTIALLY_FULFILLED" />)
    expect(screen.getByText('Partially Fulfilled')).toBeInTheDocument()
  })
})
