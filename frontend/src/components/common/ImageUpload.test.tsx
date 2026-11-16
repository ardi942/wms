import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import type { ReactNode } from 'react'
import { I18nProvider } from '../../i18n/I18nProvider'
import { server } from '../../test/server'
import { AuthenticatedImage, Avatar, ImagePicker, type ImageSelection } from './ImageUpload'

beforeAll(() => {
  Object.defineProperty(URL, 'createObjectURL', { value: vi.fn(() => 'blob:test'), configurable: true })
  Object.defineProperty(URL, 'revokeObjectURL', { value: vi.fn(), configurable: true })
})

function renderWithI18n(node: ReactNode) {
  return render(<I18nProvider>{node}</I18nProvider>)
}

beforeEach(() => {
  server.use(
    http.get('*', ({ request }) => {
      const { pathname } = new URL(request.url, 'http://localhost')
      if (pathname.endsWith('/products/p1/image')) {
        return new HttpResponse(new Uint8Array([1, 2, 3]), {
          headers: { 'Content-Type': 'image/png' },
        })
      }
      return undefined
    }),
  )
})

function selectFile(container: HTMLElement, file: File) {
  const input = container.querySelector('input[type="file"]') as HTMLInputElement
  fireEvent.change(input, { target: { files: [file] } })
}

const noSelection: ImageSelection = { file: null, remove: false }

describe('ImagePicker', () => {
  it('rejects non JPEG/PNG files', () => {
    const onChange = vi.fn()
    const { container } = renderWithI18n(
      <ImagePicker value={noSelection} onChange={onChange} alt="Product image" />,
    )
    selectFile(container, new File(['text'], 'notes.txt', { type: 'text/plain' }))
    expect(onChange).not.toHaveBeenCalled()
    expect(screen.getByRole('alert')).toHaveTextContent('Choose a JPEG or PNG image.')
  })

  it('rejects files over 2 MiB', () => {
    const onChange = vi.fn()
    const { container } = renderWithI18n(
      <ImagePicker value={noSelection} onChange={onChange} alt="Product image" />,
    )
    const file = new File([new Uint8Array(2 * 1024 * 1024 + 1)], 'large.png', { type: 'image/png' })
    selectFile(container, file)
    expect(onChange).not.toHaveBeenCalled()
    expect(screen.getByRole('alert')).toHaveTextContent('Image must be 2 MiB or smaller.')
  })

  it('accepts a valid JPEG and clears the error', () => {
    const onChange = vi.fn()
    const { container } = renderWithI18n(
      <ImagePicker value={noSelection} onChange={onChange} alt="Product image" />,
    )
    selectFile(container, new File(['text'], 'notes.txt', { type: 'text/plain' }))
    const file = new File(['image'], 'photo.jpg', { type: 'image/jpeg' })
    selectFile(container, file)
    expect(onChange).toHaveBeenCalledWith({ file, remove: false })
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('toggles removal of the current image', () => {
    const onChange = vi.fn()
    const { rerender } = renderWithI18n(
      <ImagePicker
        value={noSelection}
        onChange={onChange}
        currentUrl="/products/p1/image"
        currentVersion="v1"
        alt="Product image"
      />,
    )
    fireEvent.click(screen.getByRole('button', { name: 'Remove image' }))
    expect(onChange).toHaveBeenCalledWith({ file: null, remove: true })

    rerender(
      <I18nProvider>
        <ImagePicker
          value={{ file: null, remove: true }}
          onChange={onChange}
          currentUrl="/products/p1/image"
          currentVersion="v1"
          alt="Product image"
        />
      </I18nProvider>,
    )
    fireEvent.click(screen.getByRole('button', { name: 'Undo removal' }))
    expect(onChange).toHaveBeenLastCalledWith({ file: null, remove: false })
  })
})

describe('AuthenticatedImage', () => {
  it('renders a fetched image and falls back when missing or failing', async () => {
    server.use(
      http.get('*', ({ request }) => {
        const { pathname } = new URL(request.url, 'http://localhost')
        if (pathname.endsWith('/products/bad/image')) {
          return HttpResponse.json(
            { error: { code: 'NOT_FOUND', message: 'not found' } },
            { status: 404 },
          )
        }
        return undefined
      }),
    )

    const { unmount } = renderWithI18n(
      <AuthenticatedImage
        src="/products/p1/image"
        version="v1"
        alt="Product image"
        fallback={<span>No image</span>}
      />,
    )
    const image = await screen.findByAltText('Product image')
    expect(image).toHaveAttribute('src', 'blob:test')
    unmount()

    renderWithI18n(
      <AuthenticatedImage
        src="/products/bad/image"
        alt="Missing image"
        fallback={<span>No image</span>}
      />,
    )
    expect(await screen.findByText('No image')).toBeInTheDocument()

    renderWithI18n(
      <AuthenticatedImage src={null} alt="No source" fallback={<span>Empty image</span>} />,
    )
    expect(screen.getByText('Empty image')).toBeInTheDocument()
  })

  it('shows initials when no avatar image exists', () => {
    renderWithI18n(<Avatar name="Ada Lovelace" />)
    expect(screen.getByText('AL')).toBeInTheDocument()
  })
})
