# 📜 Changelog

All notable changes to the **Sumaron** project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html) with standard [Gitmoji](https://gitmoji.dev/) icons.

---

## [0.2.0] - 2026-08-27 🐴✨

### 🎨 Added
- :sparkles: **AI Image Generation with Imagen 3**:
  - Automatically generates `assets/logo.png` (with a fiery Eye of Sauron signature watermark on the bottom right) if not present.
  - Automatically generates `assets/arch_diagram.png` visualizing system architecture and flow.
  - Automatically generates `assets/er_diagram.png` depicting code structure and relationships.
- :framed_picture: **Markdown Visual Integration**: Embeds generated visual artifacts directly into `sumaron-summary.md`.
- :gear: **CLI Flags & Controls**: Added `--images`, `--no-images`, `--no-imagen`, `--imagen-model`, and `--force-images` flags, alongside `SUMARON_GENERATE_IMAGES` and `SUMARON_IMAGEN_MODEL` environment variables.
- :memo: **Documentation**: Added comprehensive documentation in `docs/USER_MANUAL.md`.

---

## [0.1.0] - 2026-06-25 🐴👁️

### 🚀 Initial Release
- :tada: Initial release of `sumaron` CLI tool.
- :robot: Directory summarization powered by Google Gemini API.
- :zap: Dual caching mechanism with local `.sumaron.json` and global `~/.sumaron-cache.json`.
- :art: Beautiful terminal output using Lip Gloss with spinner animations.
- :page_facing_up: Automatic generation of `sumaron-summary.md` with structured YAML frontmatter.
- :mag: Configurable file extensions, recursion depth, and file count limits.

