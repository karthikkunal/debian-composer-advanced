# Contributing to Debian Composer

Thank you for your interest in contributing to Debian Composer!

## Ways to Contribute

- **Report bugs** — Open an issue with details about the problem
- **Suggest features** — Share your ideas for improving the project
- **Write code** — Implement features or fix bugs
- **Improve documentation** — Enhance docs, README, or examples
- **Share recipes** — Contribute your system configurations
- **Test** — Try the tool and provide feedback

## Development Setup

```bash
# Clone the repository
git clone https://github.com/debian-composer/debian-composer-go.git
cd debian-composer-go

# Build the project
make build

# Run tests
make test

# Run the CLI
./bin/debian-composer-go --help
```

## Code Standards

- Follow Go idioms and best practices
- Run `go fmt` before committing
- Add tests for new functionality
- Keep functions focused and small
- Document public APIs

## Submitting Changes

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Make your changes
4. Add tests if applicable
5. Run `make test` to verify
6. Commit with clear messages
7. Push and create a pull request

## Recipe Format

If contributing recipe examples, follow the schema in `internal/types/recipe.go`:

```yaml
name: my-recipe
version: "1.0"
description: A brief description
kind: recipe  # recipe, distro, blend, persona

packages:
  - package-name

categories:
  my-category:
    condition: "optional"
    install:
      packages:
        - pkg1
        - pkg2
```

## Communication

- GitHub Issues for bug reports and feature requests
- Pull requests for code contributions

## Recognition

Contributors will be acknowledged in the project (with permission).