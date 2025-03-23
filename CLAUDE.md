# CLAUDE.md - Guidelines for LDAPEnforcer

## Build/Run/Test Commands
- Build: `go build`
- Run: `./ldapenforcer [command]`
- Test all: `go test ./...`
- Test single: `go test ./path/to/package -run TestName`
- Lint: `golangci-lint run`

## Code Style Guidelines
- **Formatting**: Run `go fmt ./...` before commits
- **Imports**: Group stdlib, third-party, and project imports with blank lines
- **Types**: Use explicit types, avoid unnecessary interface{}
- **Naming**:
  - CamelCase for exported symbols
  - mixedCase for non-exported
  - ALL_CAPS for constants
- **Error Handling**:
  - Always check errors
  - Use descriptive error messages
  - Wrap errors with context using `fmt.Errorf("context: %w", err)`
- **Comments**: Document all functions, types, and constants
- **Logging**: Use structured logging with appropriate levels
- **Testing**: Write tests for all features

## Project Structure
- `/cmd`: Main applications
- `/internal`: Private application code
- `/pkg`: Public library code
- `/api`: API definitions

Keep code organized in logically separated packages. Prefer composition over inheritance.
