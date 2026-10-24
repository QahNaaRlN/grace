import type { components } from '@/shared/api/generated/schema'
import { Field } from '@/shared/ui/field'
import { Input } from '@/shared/ui/input'
import { Disclosure } from '@/shared/ui/disclosure'

type VisualField = components['schemas']['VisualField']
export function VisualFieldInputs<K extends string>({
  fields,
  labels,
  onChange,
}: {
  fields: Record<K, VisualField | null>
  labels: Record<K, string>
  onChange: (fields: Record<K, VisualField | null>) => void
}) {
  function change(key: K, value: string) {
    const previous = fields[key]
    onChange({
      ...fields,
      [key]:
        value === ''
          ? null
          : {
              value,
              evidence: previous?.evidence ?? [{ page: 1, quote: value, box: [0, 0, 1, 1] }],
            },
    })
  }
  return (
    <div className="grid gap-3 sm:grid-cols-2">
      {(Object.keys(labels) as K[]).map((key) => (
        <div key={key} className="space-y-1">
          <Field label={labels[key]}>
            <Input
              value={fields[key]?.value ?? ''}
              placeholder="Неизвестно"
              onChange={(e) => change(key, e.target.value)}
            />
          </Field>
          {fields[key] && (
            <Disclosure title={`Источник: ${labels[key]}`}>
              {fields[key].evidence.map((e, i) => (
                <div key={i} className="space-y-2">
                  <Field label={`Страница источника: ${labels[key]}`}>
                    <Input
                      type="number"
                      min={1}
                      max={4}
                      value={e.page}
                      onChange={(event) =>
                        onChange({
                          ...fields,
                          [key]: {
                            ...fields[key],
                            evidence: fields[key]!.evidence.map((q, n) =>
                              n === i ? { ...q, page: Number(event.target.value) } : q,
                            ),
                          },
                        })
                      }
                    />
                  </Field>
                  <Field label={`Цитата: ${labels[key]}`}>
                    <Input
                      value={e.quote}
                      onChange={(event) =>
                        onChange({
                          ...fields,
                          [key]: {
                            ...fields[key],
                            evidence: fields[key]!.evidence.map((q, n) =>
                              n === i ? { ...q, quote: event.target.value } : q,
                            ),
                          },
                        })
                      }
                    />
                  </Field>
                  <p className="text-xs text-muted-foreground">
                    Область страницы: {e.box.join(', ')}. Предложение для ручной сверки.
                  </p>
                </div>
              ))}
            </Disclosure>
          )}
        </div>
      ))}
    </div>
  )
}
