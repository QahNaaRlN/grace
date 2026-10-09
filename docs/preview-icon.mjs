import { createServer } from 'node:http'
import { readFile } from 'node:fs/promises'
const routes = new Map([
  ['/', [new URL('./icon-preview.html', import.meta.url), 'text/html; charset=utf-8']],
  ['/mark.svg', [new URL('./grace-ruble-pulse.svg', import.meta.url), 'image/svg+xml']],
])
createServer(async (req, res) => {
  const route = routes.get(new URL(req.url, 'http://localhost').pathname)
  if (!route || req.method !== 'GET') { res.writeHead(404).end(); return }
  res.writeHead(200, { 'Content-Type': route[1], 'Cache-Control': 'no-store' })
  res.end(await readFile(route[0]))
}).listen(6221, '127.0.0.1')
