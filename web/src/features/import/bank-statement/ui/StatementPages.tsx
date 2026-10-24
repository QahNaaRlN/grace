import { useState } from 'react'
import { env } from '@/shared/config'
import { Button } from '@/shared/ui/button'
import { PrivateSourceImage } from '@/shared/ui/private-source-image'

export function StatementPages({
  id,
  count,
  onViewed,
}: {
  id: string
  count: number
  onViewed?: (page: number) => void
}) {
  const [page, setPage] = useState(1)
  return (
    <section className="space-y-3">
      <div className="flex flex-wrap gap-2" aria-label="Страницы источника">
        {Array.from({ length: count }, (_, i) => i + 1).map((n) => (
          <Button key={n} variant={page === n ? 'default' : 'outline'} onClick={() => setPage(n)}>
            Страница {n}
          </Button>
        ))}
      </div>
      <PrivateSourceImage
        src={`${env.VITE_API_BASE_URL}/api/imports/statements/${id}/pages/${page}`}
        label={`Страница ${page} источника`}
        onLoaded={() => onViewed?.(page)}
      />
    </section>
  )
}
