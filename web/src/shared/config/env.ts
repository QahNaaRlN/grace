import { z } from 'zod'

const EnvSchema = z.object({
  VITE_API_BASE_URL: z.url(),
})

export type AppEnv = z.infer<typeof EnvSchema>

/**
 * Конфигурация окружения, валидируемая через Zod при старте приложения.
 */
export const env: AppEnv = EnvSchema.parse({
  VITE_API_BASE_URL:
    import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080',
})
