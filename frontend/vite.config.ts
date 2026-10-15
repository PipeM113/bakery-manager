import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig(({ command, mode }) => {
  // A build without the API url used to fall back to localhost and ship a site that
  // cannot log in. Fail the build instead.
  if (command === 'build') {
    const url = loadEnv(mode, process.cwd(), 'VITE_').VITE_API_URL
    if (!url) {
      throw new Error('VITE_API_URL is required to build: set it to the backend URL (see .env.example)')
    }
    try {
      const parsed = new URL(url)
      if (parsed.protocol !== 'https:' && parsed.protocol !== 'http:') throw new Error('protocol')
    } catch {
      throw new Error('VITE_API_URL must be an absolute http(s) URL, for example https://api.example.com')
    }
  }
  return { plugins: [react()] }
})
