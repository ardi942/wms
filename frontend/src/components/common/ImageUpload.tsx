import { useEffect, useRef, useState, type ReactNode } from 'react'
import { ImageIcon, Trash2, Undo2, Upload } from 'lucide-react'
import { fetchBlob } from '../../api/client'
import { useT } from '../../i18n/I18nProvider'
import { cn } from '../../lib/utils'
import { Button } from '../ui'

const MAX_IMAGE_BYTES = 2 * 1024 * 1024
const ACCEPTED_TYPES = ['image/jpeg', 'image/png']

function imageCachePath(src: string, version?: string | null): string {
    return `${src}${version ? `?v=${encodeURIComponent(version)}` : ''}`
}

export function AuthenticatedImage({
    src,
    version,
    alt,
    className,
    fallback,
}: {
    src: string | null | undefined
    version?: string | null
    alt: string
    className?: string
    fallback?: ReactNode
}) {
    const [objectUrl, setObjectUrl] = useState<string | null>(null)
    const [failed, setFailed] = useState(false)

    useEffect(() => {
        if (!src) {
            setObjectUrl(null)
            setFailed(false)
            return
        }
        const controller = new AbortController()
        let url: string | null = null
        setFailed(false)
        setObjectUrl(null)
        fetchBlob(imageCachePath(src, version), controller.signal)
            .then((blob) => {
                if (controller.signal.aborted) return
                url = URL.createObjectURL(blob)
                setObjectUrl(url)
            })
            .catch(() => {
                if (!controller.signal.aborted) setFailed(true)
            })
        return () => {
            controller.abort()
            if (url) URL.revokeObjectURL(url)
        }
    }, [src, version])

    if (!src || failed || !objectUrl) return <>{fallback ?? null}</>
    return <img src={objectUrl} alt={alt} className={className} />
}

function initials(name: string): string {
    const words = name.trim().split(/\s+/).filter(Boolean)
    const value = words.slice(0, 2).map((word) => word[0]).join('').toUpperCase()
    return value || '?'
}

export function Avatar({
    src,
    version,
    name,
    size = 'md',
    className,
}: {
    src?: string | null
    version?: string | null
    name: string
    size?: 'sm' | 'md' | 'lg'
    className?: string
}) {
    const sizes = {
        sm: 'h-8 w-8 text-xs',
        md: 'h-9 w-9 text-sm',
        lg: 'h-20 w-20 text-xl',
    }
    return (
        <span
            className={cn(
                'inline-grid shrink-0 place-items-center overflow-hidden rounded-full bg-brand-600 font-semibold text-white',
                sizes[size],
                className,
            )}
        >
            <AuthenticatedImage
                src={src}
                version={version}
                alt={name}
                className="h-full w-full object-cover"
                fallback={<span aria-hidden="true">{initials(name)}</span>}
            />
        </span>
    )
}

export interface ImageSelection {
    file: File | null
    remove: boolean
}

export function ImagePicker({
    value,
    onChange,
    currentUrl,
    currentVersion,
    disabled,
    shape = 'product',
    alt,
}: {
    value: ImageSelection
    onChange: (value: ImageSelection) => void
    currentUrl?: string | null
    currentVersion?: string | null
    disabled?: boolean
    shape?: 'product' | 'avatar'
    alt: string
}) {
    const { t } = useT()
    const inputRef = useRef<HTMLInputElement>(null)
    const [preview, setPreview] = useState<string | null>(null)
    const [error, setError] = useState<string | null>(null)

    useEffect(() => {
        if (!value.file) {
            setPreview(null)
            return
        }
        const url = URL.createObjectURL(value.file)
        setPreview(url)
        return () => URL.revokeObjectURL(url)
    }, [value.file])

    const selectFile = (file: File | undefined) => {
        if (!file) return
        if (!ACCEPTED_TYPES.includes(file.type)) {
            setError(t('image.invalidType'))
            return
        }
        if (file.size > MAX_IMAGE_BYTES) {
            setError(t('image.tooLarge'))
            return
        }
        setError(null)
        onChange({ file, remove: false })
    }

    const hasImage = Boolean(value.file || (currentUrl && !value.remove))
    const shapeClass = shape === 'avatar' ? 'rounded-full' : 'rounded-lg'
    const removeItem = () => {
        if (value.file) onChange({ file: null, remove: Boolean(currentUrl) })
        else onChange({ file: null, remove: !value.remove })
    }

    return (
        <div className="space-y-2">
            <div
                className={cn(
                    'relative grid h-36 w-36 place-items-center overflow-hidden border border-dashed border-slate-300 bg-slate-50 text-slate-400',
                    shapeClass,
                )}
            >
                {preview ? (
                    <img src={preview} alt={alt} className="h-full w-full object-cover" />
                ) : (
                    <AuthenticatedImage
                        src={value.remove ? null : currentUrl}
                        version={currentVersion}
                        alt={alt}
                        className="h-full w-full object-cover"
                        fallback={<ImageIcon className="h-8 w-8" />}
                    />
                )}
            </div>
            <input
                ref={inputRef}
                type="file"
                accept="image/jpeg,image/png"
                className="sr-only"
                disabled={disabled}
                onChange={(event) => {
                    selectFile(event.target.files?.[0])
                    event.target.value = ''
                }}
            />
            <div className="flex flex-wrap gap-2">
                <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    disabled={disabled}
                    onClick={() => inputRef.current?.click()}
                >
                    <Upload className="h-4 w-4" />
                    {hasImage ? t('image.replace') : t('image.choose')}
                </Button>
                {(value.file || (currentUrl && (value.remove || !value.file))) && (
                    <Button type="button" variant="ghost" size="sm" disabled={disabled} onClick={removeItem}>
                        {value.remove ? <Undo2 className="h-4 w-4" /> : <Trash2 className="h-4 w-4" />}
                        {value.remove ? t('image.undoRemove') : t('image.remove')}
                    </Button>
                )}
            </div>
            <p className="text-xs text-slate-500">{t('image.hint')}</p>
            {error && <p role="alert" className="text-xs font-medium text-red-600">{error}</p>}
        </div>
    )
}
