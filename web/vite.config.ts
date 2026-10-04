import tailwindcss from '@tailwindcss/vite';
import react from '@vitejs/plugin-react';
import path from 'node:path';
import { defineConfig } from 'vitest/config';

export default defineConfig({
	plugins: [react(), tailwindcss()],
	test: {
		environment: 'jsdom',
		setupFiles: './src/test/setup.ts',
	},
	resolve: {
		alias: {
			'@': path.resolve(__dirname, './src'),
		},
	},
	server: {
		host: true,
		port: 5173,
	},
	preview: {
		host: true,
		port: 4173,
	},
});
