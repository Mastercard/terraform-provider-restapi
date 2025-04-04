# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands
- Run all tests: `cd restapi && go test`
- Run a single test: `cd restapi && go test -run TestName`
- Generate documentation: `go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs`
- Run tests with script: `./scripts/test.sh [test args]`

## Code Style
- Run `go fmt` before submitting changes
- Follow standard Go conventions for naming (CamelCase)
- Error handling: Check all errors and provide meaningful error messages
- Import order: standard library first, then third-party packages
- Use Go 1.24+ compatible syntax and features
- Write tests for all new functionality
- Add appropriate documentation for public APIs

## Development Guidelines
- Ensure new attributes can be set by environment variables
- Update documentation when adding new features
- Maintain backward compatibility when possible
- Test with the fakeserver CLI tool for API interactions
- Follow the provider pattern established in the codebase