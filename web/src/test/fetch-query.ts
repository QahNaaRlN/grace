import { QueryClient, type FetchQueryOptions, type QueryKey } from '@tanstack/react-query'

/** Runs a query factory's queryFn once through a fresh client, without retries. */
export function fetchQuery<T, K extends QueryKey>(options: FetchQueryOptions<T, Error, T, K>) {
  return new QueryClient({ defaultOptions: { queries: { retry: false } } }).fetchQuery(options)
}

/** True when `key` starts with `prefix`, i.e. invalidating the prefix refreshes the key. */
export function startsWith(key: readonly unknown[], prefix: readonly unknown[]) {
  return prefix.every((part, index) => key[index] === part)
}
