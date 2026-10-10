// Local, synthetic-only preview. No API, database, secret or external requests.
import { createServer } from 'node:http'
import { readFile } from 'node:fs/promises'
const routes = new Map([
  ['/', [new URL('./design-preview.html', import.meta.url), 'text/html; charset=utf-8']],
  ['/favicon.svg', [new URL('../web/public/favicon.svg', import.meta.url), 'image/svg+xml']],
])
const server = createServer(async (req, res) => {
  const route = routes.get(new URL(req.url, 'http://localhost').pathname)
  if (!route || req.method !== 'GET') { res.writeHead(404).end(); return }
  try {
    res.writeHead(200, { 'Content-Type': route[1], 'Cache-Control': 'no-store',
      'Content-Security-Policy': "default-src 'self'; style-src 'unsafe-inline'; script-src 'unsafe-inline'; img-src 'self'; connect-src 'none'; form-action 'none'; frame-ancestors 'none'" })
    res.end(await readFile(route[0]))
  } catch { res.writeHead(500).end('Preview unavailable') }
})
server.listen(6219, '127.0.0.1', () => console.log('Grace synthetic design preview: http://127.0.0.1:6219'))
