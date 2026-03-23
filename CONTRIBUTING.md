# Contributing to gh-projects

Thank you for your interest in contributing to `gh-projects`. We appreciate your help in making this tool better.

## Development Setup

To get started with development, ensure you have Go 1.26.1 or later installed.

1. Clone the repository:

   ```bash
   git clone https://github.com/ifloresarg/gh-projects.git
   cd gh-projects
   ```

2. Install dependencies:

   ```bash
   go mod download
   ```

3. Build the project:

   ```bash
   go build ./...
   ```

4. Run tests:

   ```bash
   go test ./...
   ```

## Project Structure

- `cmd/gh-projects/`: Entry point for the CLI.
- `internal/tui/`: Bubble Tea UI components and application logic.
- `internal/github/`: GitHub API client and authentication handling.
- `internal/config/`: Configuration management.
- `nvim-plugin/`: Neovim integration plugin.

## Common Tasks

We use a `Makefile` for common development tasks. Run `make` to see available targets.

To create a snapshot build with GoReleaser:

```bash
goreleaser build --snapshot --clean
```

## Pull Request Guidelines

1. Create a new branch for your changes.
2. Ensure your code follows the existing style and conventions.
3. Include tests for any new features or bug fixes.
4. Verify that all tests pass and `go build ./...` succeeds.
5. Provide a clear and descriptive pull request summary.

## Reporting Issues

If you find a bug or have a feature request, please open an issue on GitHub. Provide as much detail as possible, including steps to reproduce for bugs.
