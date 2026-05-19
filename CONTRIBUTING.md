# Contributing to git-user

Thank you for your interest in contributing! This document provides guidelines and instructions for contributing.

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/git-user.git
   cd git-user
   ```
3. **Create a branch** for your changes:
   ```bash
   git checkout -b feature/your-feature-name
   ```

## Development Setup

### Prerequisites
- Go 1.21 or later
- Git
- ssh-keygen (for testing SSH features)

### Building
```bash
go build -o git-user .
```

### Running Tests
```bash
go test ./...
```

### Running Locally
```bash
./git-user --help
```

## Making Changes

### Code Style
- Follow standard Go conventions
- Run `go fmt` before committing
- Keep functions small and focused
- Add comments for complex logic

### Commit Messages
Use clear, descriptive commit messages:
- `feat: add new command for X`
- `fix: resolve issue with Y`
- `docs: update README with Z`
- `refactor: simplify W logic`
- `test: add tests for V`

### Testing
- Add tests for new features
- Ensure existing tests pass
- Test on multiple platforms if possible

## Submitting Changes

1. **Push your changes** to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```

2. **Create a Pull Request** on GitHub:
   - Provide a clear title and description
   - Reference any related issues
   - Explain what changes you made and why

3. **Wait for review**:
   - Address any feedback from maintainers
   - Make requested changes if needed

## Reporting Issues

When reporting bugs, please include:
- Your operating system and version
- Go version (`go version`)
- Steps to reproduce the issue
- Expected vs actual behavior
- Any error messages or logs

## Feature Requests

We welcome feature requests! Please:
- Check if the feature already exists
- Explain the use case clearly
- Describe how it would work
- Consider if it fits the project's scope (simple, focused Git identity management)

## Code of Conduct

- Be respectful and constructive
- Welcome newcomers
- Focus on the code, not the person
- Assume good intentions

## Questions?

Open an issue with the `question` label, and we'll help you out!

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
