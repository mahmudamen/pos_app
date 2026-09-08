# Contributing to Sales Pro

First off, thank you for considering contributing to Sales Pro! It's people like you that make the open-source community such an amazing place to learn, inspire, and create.

## Getting Started

### Prerequisites
- [Flutter SDK](https://flutter.dev/docs/get-started/install) (Latest Stable)
- [Dart SDK](https://dart.dev/get-dart)
- An IDE (VS Code or Android Studio) with Flutter/Dart plugins installed.

### Installation
1. **Fork** the repository on GitHub.
2. **Clone** your fork locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/Sales-Pro.git
   cd Sales-Pro
   ```
3. **Install Dependencies**:
   ```bash
   flutter pub get
   ```
4. **Generate Code** (Required for Drift/SQLite and JSON serialization):
   ```bash
   dart run build_runner build --delete-conflicting-outputs
   ```
5. **Run the App**:
   ```bash
   flutter run
   ```

## Development Workflow

1. **Create a Branch**: Always work on a new branch for your changes.
   ```bash
   git checkout -b feature/amazing-feature
   # or
   git checkout -b fix/annoying-bug
   ```
2. **Make Changes**: Implement your feature or fix.
3. **Test**: Ensure the app builds and runs correctly. Run existing tests if applicable:
   ```bash
   flutter test
   ```
4. **Commit**: Write clear, concise commit messages.
   ```bash
   git commit -m "Add feature X to inventory screen"
   ```

## Project Structure
Familiarize yourself with the folder structure before making changes:
- `lib/features/`: Contains the core business logic and UI for specific modules (auth, invoice, etc.).
- `lib/db/`: Database definitions (Drift). If you modify tables, remember to run the build runner.
- `lib/state/`: State management using Provider.
- `assets/lang/`: Localization files (`ar.json`, `en.json`).

## Code Style
- Follow [Effective Dart](https://dart.dev/guides/language/effective-dart) guidelines.
- Use `flutter_lints` (configured in `analysis_options.yaml`) to ensure code quality.
- Run `dart format .` before committing to ensure consistent formatting.

## Reporting Bugs
- Check if the issue has already been reported.
- Use the **Bug Report** template.
- Include screenshots, logs, and steps to reproduce.

## Pull Requests
1. Push your branch to your fork.
2. Open a Pull Request against the `main` branch of the original repository.
3. Describe your changes clearly.
4. Reference any related issues (e.g., "Closes #123").

## License
By contributing, you agree that your contributions will be licensed under the project's [MIT License](LICENSE).
