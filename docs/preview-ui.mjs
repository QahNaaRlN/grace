// Reproducible review only. Mocked data/mutations are isolated from production.
import { copyFile, mkdir } from 'node:fs/promises'
import { spawn } from 'node:child_process'
import { fileURLToPath } from 'node:url'
const web = new URL('../web/', import.meta.url)
const destination = new URL('node_modules/.grace-ui-qa/', web)
await mkdir(destination, { recursive: true })
for (const file of ['ui-preview.tsx', 'ui-preview.html']) {
  await copyFile(new URL(`ui-review/${file}`, import.meta.url), new URL(file, destination))
}
const child = spawn(process.execPath, ['node_modules/vite/bin/vite.js', '--host', '127.0.0.1', '--port', '6224', '--strictPort'], { cwd: fileURLToPath(web), stdio: 'inherit', windowsHide: true })
console.log('Grace review: http://127.0.0.1:6224/node_modules/.grace-ui-qa/ui-preview.html')
child.on('exit', code => process.exit(code ?? 0))
process.on('SIGINT', () => child.kill())
process.on('SIGTERM', () => child.kill())
