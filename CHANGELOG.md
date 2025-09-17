# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.0] - 2025-09-17

### Added

- Added `-n` / `--lines` option to specify the number of lines to be processed in a single batch.

## [1.0.0] - 2025-09-12

### Added

- Initial release of `llm-batch`.
- Batch processing of JSON/JSONL data from file or stdin.
- Sequential and parallel execution modes (`-c` option).
- Real-time output monitoring with streaming mode (`--stream` option).
- Flexible output formats: `text`, `json`, `jsonl` (`-o` option).
- Support for specifying `llm-cli` profiles (`-L` option).
- Prompt can be provided directly (`-P`) or from a file (`-F`).
- Cross-platform build support for Linux, macOS, and Windows.
- English (`README.md`) and Japanese (`README.ja.md`) documentation.
- MIT License.
- Makefile for easy building and packaging.