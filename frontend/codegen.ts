import type { CodegenConfig } from '@graphql-codegen/cli'

const config: CodegenConfig = {
  overwrite: true,
  schema: 'http://localhost:8080/graphql',
  documents: ['src/**/*.{ts,tsx}'],
  ignoreNoDocuments: true,
  generates: {
    './src/types/__generated__/graphql.ts': {
      plugins: ['typescript', 'typescript-operations'],
      config: {
        avoidOptionals: {
          field: true,
          inputValue: false,
        },
        defaultScalarType: 'unknown',
        nonOptionalTypename: true,
        skipTypeNameForRoot: true,
      },
    },
  },
}

export default config
