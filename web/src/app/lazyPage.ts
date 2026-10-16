import { createPath, type Location, type RouteObject } from 'react-router-dom'

type PageRoute = Pick<RouteObject, 'Component' | 'element'>

export const chunkReloadKey = 'grace:page-chunk-reload'

/**
 * SPEC-0052: `lazy` маршрута страницы. Если чанк не загрузился (вкладка прошлого выпуска
 * после деплоя или нет сети), браузер один раз открывает целевой адрес заново и получает
 * новый `index.html`. Повторная ошибка уходит в React Router.
 */
export function lazyPage(load: () => Promise<PageRoute>, target: () => Location) {
  return async (): Promise<PageRoute> => {
    try {
      const route = await load()
      forgetChunkReload()
      return route
    } catch (error) {
      if (!rememberChunkReload()) {
        throw error
      }
      window.location.assign(createPath(target()))
      return new Promise<never>(() => {})
    }
  }
}

function rememberChunkReload(): boolean {
  try {
    if (window.sessionStorage.getItem(chunkReloadKey)) {
      return false
    }
    window.sessionStorage.setItem(chunkReloadKey, '1')
    return true
  } catch {
    return false
  }
}

function forgetChunkReload() {
  try {
    window.sessionStorage.removeItem(chunkReloadKey)
  } catch {
    // Без sessionStorage перезагрузки не было, снимать нечего.
  }
}
