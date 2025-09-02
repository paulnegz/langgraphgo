# Changelog

## [0.1.0] - 2025-01-02

### Added
- Generic state management - works with any type, not just MessageContent
- Performance optimizations for production use
- Support for any LLM client (removed hard dependency on LangChain)

### Changed
- Simplified API for building graphs
- Updated examples to show generic usage

### Fixed
- CI/CD pipeline issues from original repository
- Build errors with recent Go versions

### Removed
- Hard dependency on LangChain - now works with any LLM library