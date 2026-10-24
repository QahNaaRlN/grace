import { useEffect, useState } from 'react'
import { Button } from './button'

export function PrivateSourceImage({
  src,
  label,
  onLoaded,
}: {
  src: string
  label: string
  onLoaded?: () => void
}) {
  const [zoom, setZoom] = useState(false)
  const [result, setResult] = useState({ src: '', image: '', error: false })
  useEffect(() => {
    const controller = new AbortController()
    let objectURL = ''
    let active = true
    void fetch(src, {
      credentials: 'include',
      cache: 'no-store',
      redirect: 'error',
      signal: controller.signal,
    })
      .then(async (response) => {
        if (
          !response.ok ||
          !['image/png', 'image/jpeg'].includes(
            response.headers.get('Content-Type')?.split(';')[0] ?? '',
          )
        )
          throw new Error('source_unavailable')
        const blob = await response.blob()
        if (!active) return
        objectURL = URL.createObjectURL(blob)
        setResult({ src, image: objectURL, error: false })
      })
      .catch(() => {
        if (active) setResult({ src, image: '', error: true })
      })
    return () => {
      active = false
      controller.abort()
      if (objectURL) URL.revokeObjectURL(objectURL)
    }
  }, [src])
  if (result.src !== src) return <p role="status">Загрузка страницы…</p>
  if (result.error)
    return <p role="alert">Изображение недоступно. Проверьте срок хранения и обновите источник.</p>
  if (!result.image) return <p role="status">Загрузка страницы…</p>
  return (
    <div className="space-y-2">
      <Button variant="outline" onClick={() => setZoom(!zoom)}>
        {zoom ? 'Вписать страницу' : 'Увеличить страницу'}
      </Button>
      <div className="max-h-[80vh] overflow-auto">
        <img
          className={
            zoom ? 'max-w-none rounded border' : 'max-h-[75vh] w-full rounded border object-contain'
          }
          src={result.image}
          alt={label}
          onLoad={onLoaded}
        />
      </div>
    </div>
  )
}
