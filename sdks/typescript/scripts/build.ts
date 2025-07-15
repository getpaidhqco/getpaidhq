import { execSync } from 'child_process'
import { readFileSync, writeFileSync } from 'fs'
import { resolve, dirname } from 'path'
import { fileURLToPath } from 'url'

// Get the directory name in ES modules
const __filename = fileURLToPath(import.meta.url)
const __dirname = dirname(__filename)
const PROJECT_ROOT = resolve(__dirname, '..')

async function build() {
  console.log('🔨 Building TypeScript SDK...')

  // Clean previous build
  execSync('npm run clean', { stdio: 'inherit' })

  // Type check
  console.log('🔍 Type checking...')
  execSync('npm run typecheck', { stdio: 'inherit' })

  // Lint
  console.log('🧹 Linting...')
  execSync('npm run lint', { stdio: 'inherit' })

  // Build with Vite
  console.log('📦 Building bundles...')
  execSync('npx vite build', { stdio: 'inherit' })

  // Copy package.json to dist for publishing
  const packageJsonPath = resolve(PROJECT_ROOT, 'package.json')
  const distPackageJsonPath = resolve(PROJECT_ROOT, 'dist/package.json')
  const packageJson = JSON.parse(readFileSync(packageJsonPath, 'utf-8'))
  writeFileSync(distPackageJsonPath, JSON.stringify(packageJson, null, 2))

  // Copy README and other files
  const readmePath = resolve(PROJECT_ROOT, 'README.md')
  const changelogPath = resolve(PROJECT_ROOT, 'CHANGELOG.md')
  const distDir = resolve(PROJECT_ROOT, 'dist')
  execSync(`cp ${readmePath} ${changelogPath} ${distDir}/`, { stdio: 'inherit' })

  console.log('✅ Build completed successfully!')
}

build().catch(console.error)
