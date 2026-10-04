import { z } from 'zod'

const ApiBaseUrlSchema = z.union([z.literal(''), z.url()])

const EnvSchema = z.object({
  VITE_API_BASE_URL: ApiBaseUrlSchema,
})

export type AppEnv = z.infer<typeof EnvSchema>

/**
 * Конфигурация окружения, валидируемая через Zod при старте приложения.
 */
export const env: AppEnv = EnvSchema.parse({
  VITE_API_BASE_URL: import.meta.env.VITE_API_BASE_URL ?? '',
})
