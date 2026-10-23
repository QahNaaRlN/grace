// Parse the actual workflow files with the YAML parser in the locked Web tools.
// Mode-isolation tests inspect expressions; they do not parse YAML syntax.
import { readdir, readFile } from 'node:fs/promises';
import { createRequire } from 'node:module';

const require = createRequire(new URL('../../web/package.json', import.meta.url));
const { format } = require('prettier');
const directory = new URL('../../.github/workflows/', import.meta.url);
const files = (await readdir(directory)).filter((name) => /\.ya?ml$/.test(name)).sort();
if (files.length === 0) throw new Error('No workflows found');
for (const name of files) {
  try {
    await format(await readFile(new URL(name, directory), 'utf8'), { parser: 'yaml' });
  } catch (error) {
    console.error(`${name}: ${error.message}`);
    process.exitCode = 1;
  }
}
if (!process.exitCode) console.log(`Workflow YAML syntax: ${files.length} files PASS`);
