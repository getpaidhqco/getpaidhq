import {execSync} from 'child_process'
import {existsSync, mkdirSync, writeFileSync} from 'fs'
import {dirname, resolve} from 'path'
import {fileURLToPath} from 'url'

// Get the directory name in ES modules
const __filename = fileURLToPath(import.meta.url)
const __dirname = dirname(__filename)

const SWAGGER_PATH = resolve(__dirname, '../../../swagger.yml')
const OUTPUT_DIR = resolve(__dirname, '../src/generated')
const CONFIG_FILE = resolve(__dirname, 'openapi-config.json')

// OpenAPI Generator configuration
const generatorConfig = {
  packageName: '@getpaidhq/sdk',
  packageVersion: '0.1.0',
  npmName: '@getpaidhq/sdk',
  supportsES6: true,
  withInterfaces: true,
  useSingleRequestParameter: true,
  withNodeImports: true,
  modelPropertyNaming: 'camelCase',
  enumPropertyNaming: 'UPPERCASE',
  stringEnums: true,
  npmRepository: 'https://registry.npmjs.org/',
  additionalProperties: {
    platformVersion: '18.0.0',
    withoutRuntimeChecks: true,
    withSeparateModelsAndApi: true,
    apiPackage: 'api',
    modelPackage: 'models'
  }
}

async function generateSDK() {
  console.log('🚀 Generating TypeScript SDK from OpenAPI spec...')

  // Ensure output directory exists
  if (!existsSync(OUTPUT_DIR)) {
    mkdirSync(OUTPUT_DIR, { recursive: true })
  }

  // Write generator config
  writeFileSync(CONFIG_FILE, JSON.stringify(generatorConfig, null, 2))

  try {
    // Generate the base SDK using OpenAPI Generator
    const command = [
      'npx @openapitools/openapi-generator-cli generate',
      `-i ${SWAGGER_PATH}`,
      `-g typescript-axios`,
      `-o ${OUTPUT_DIR}`,
      `-c ${CONFIG_FILE}`,
      '--skip-validate-spec',
      '--remove-operation-id-prefix'
    ].join(' ')

    console.log('📦 Running OpenAPI Generator...')
    execSync(command, { stdio: 'inherit' })

    // Post-process generated files
    await postProcessGenerated()

    console.log('✅ SDK generation completed successfully!')
  } catch (error) {
    console.error('❌ SDK generation failed:', error)
    process.exit(1)
  }
}

async function postProcessGenerated() {
  console.log('🔧 Post-processing generated files...')

  // Add custom header to generated files
  const header = `/**
 * GetPaidHQ API SDK
 * Generated from OpenAPI specification
 * 
 * @version ${generatorConfig.packageVersion}
 * @generated
 */

`

  // You can add additional post-processing logic here
  // For example, fixing imports, adding custom methods, etc.
}

// Run the generation
generateSDK().catch(console.error)
