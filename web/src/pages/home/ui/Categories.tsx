import { useState, type FormEvent } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { Pencil, Plus, Tag } from 'lucide-react'
import type { Category } from '@/shared/api/types'
import { apiClient } from '@/shared/api/client'
import { csrfToken } from '@/shared/model/auth-session'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { Label } from '@/shared/ui/label'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/shared/ui/card'
import { Dialog, DialogContent, DialogTitle, DialogDescription, DialogHeader } from '@/shared/ui/dialog'

export function Categories({ items }: { items: Category[] }) {
  const cache = useQueryClient()
  const [editing, setEditing] = useState<Category | 'new' | null>(null)
  const [name, setName] = useState('')
  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')
  function edit(item: Category | 'new') { setEditing(item); setName(item === 'new' ? '' : item.name); setError('') }
  async function save(e: FormEvent) {
    e.preventDefault(); if (!editing || pending) return
    setPending(true); setError('')
    try {
      const result = editing === 'new'
        ? await apiClient.POST('/api/v1/categories', { params: { header: { 'X-CSRF-Token': csrfToken } }, body: { name: name.trim() } })
        : await apiClient.PATCH('/api/v1/categories/{categoryId}', { params: { path: { categoryId: editing.id }, header: { 'X-CSRF-Token': csrfToken } }, body: { name: name.trim() } })
      if (!result.data) throw new Error(result.error?.message ?? 'Не удалось сохранить категорию.')
      await Promise.all([['categories'], ['financial-events'], ['statistics']].map(queryKey => cache.invalidateQueries({ queryKey })))
      setEditing(null)
    } catch (e) { setError(e instanceof Error ? e.message : 'Не удалось сохранить категорию.') }
    finally { setPending(false) }
  }
  return <>
    <Card><CardHeader><CardTitle>Категории расходов</CardTitle><CardDescription>Выбирайте их при добавлении покупки или проверке импорта. Свои категории можно переименовать; общие доступны всем и сохраняют названия.</CardDescription></CardHeader>
      <CardContent className="space-y-4"><Button onClick={() => edit('new')}><Plus />Добавить категорию</Button>
        <div className="space-y-2">{items.filter(c => c.kind === 'expense').map(c => <div key={c.id} className="flex items-center gap-3 rounded-xl border p-3"><Tag className="size-5 shrink-0 text-primary" aria-hidden="true" /><div className="min-w-0 flex-1"><p className="break-words font-medium">{c.name}</p><p className="text-sm text-muted-foreground">{c.editable ? 'Ваша категория' : 'Общая категория'}</p></div>{c.editable && <Button size="icon" variant="ghost" aria-label={`Переименовать ${c.name}`} onClick={() => edit(c)}><Pencil /></Button>}</div>)}</div>
        <p className="text-sm text-muted-foreground">Переименование сохраняет связи с покупками. Удаление категорий пока не предусмотрено.</p>
      </CardContent></Card>
    <Dialog open={editing !== null} onOpenChange={open => { if (!open && !pending) setEditing(null) }}><DialogContent><DialogHeader><DialogTitle>{editing === 'new' ? 'Новая категория' : 'Переименовать категорию'}</DialogTitle><DialogDescription>Название поможет найти покупки в операциях и статистике.</DialogDescription></DialogHeader><form onSubmit={save} className="space-y-4"><div className="space-y-2"><Label htmlFor="category-name">Название категории</Label><Input id="category-name" value={name} onChange={e => setName(e.target.value)} required maxLength={80} placeholder="Например, Спорт" /></div>{error && <p role="alert" className="text-sm text-destructive">{error}</p>}<Button type="submit" disabled={pending || !name.trim()}>{pending ? 'Сохраняем…' : 'Сохранить'}</Button></form></DialogContent></Dialog>
  </>
}
