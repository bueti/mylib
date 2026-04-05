## Golang

- In tests, use testify assert/require rather than t.Error
- Use require in tests to eliminate possible runtime panics, e.g. require.Len(t, someSlice, 1) before accessing a slice index
- Order files by importance to readers: exported types, tests and methods first, helper methods and unexported types later.
- Write tests where needed to ensure future refactoring isn't introducting regressions. Don't test simple things.

## TypeScript, Svelte

- This project uses nodejs and pnpm. Check in the root .tool-versions file if you need to know which versions.
- ALWAYS use pnpm, NEVER use npm.
- ALWAYS check for type errors using the get_diagnostics tool where available or if not then use turbo check:types --affected
- Fix formatting issues with pnpm fix:format or turbo fix:format --affected
- Run unit tests with pnpm test:unit or turbo test:unit --affected
- Run unit tests in watch mode with pnpm test:unit:watch or turbo test:unit:watch --affected
- For continuous development, run watch commands in background shells:
  - pnpm test:watch or turbo test:watch --affected for watching all tests
  - pnpm test:unit:watch or turbo test:unit:watch --affected for watching unit tests only
