# SUMARON User Manual

`sumaron` is a CLI tool that recursively summarizes the contents of a directory using the Gemini API and generates visual architecture diagrams and branding using Imagen 3. It features a local and global caching mechanism to ensure that directories are only summarized when files matching target extensions are modified.

## Prerequisites

- **Go**: Version 1.26 or higher.
- **Gemini API Key**: Set in the `GEMINI_API_KEY` environment variable, or passed via `--key` command line argument.

## Installation

You can build and install the binary to your local bin directory (`~/bin` by default):

```bash
just build
just install
```

Make sure `~/bin` is in your system's `PATH`.

## Command Line Interface

```bash
sumaron [flags]
```

### Flags & Environment Variables

All flags support both double-dash (`--`) and single-dash (`-`) prefixes.

*   `--folder` / `-f` `<path>`: The path to the directory to summarize (default: `.`).
*   `--key` / `-k` `<api-key>`: Override or provide the Gemini API key (defaults to `GEMINI_API_KEY` environment variable).
*   `--model` / `-m` `<model-name>`: The Gemini model to use (default: `gemini-flash-latest`, recommended: `gemini-flash-latest` or `gemini-3.5-flash`).
*   `--extensions` / `-e` `<exts>`: Comma-separated list of file extensions to include (default: `.md,.html,.json`).
*   `--force`: Skip cache checks and force a re-summarization of the folder.
*   `--global-cache` / `-g` `<path>`: Custom location for the global central cache file (defaults to `~/.sumaron-cache.json`).
*   `--max-files` / `--max-file` / `-n` `<count>`: Maximum number of files to summarize (default: `20`). Can also be overridden globally using the environment variable `SUMARON_MAX_FILES`.
*   `--max-file-depth` / `--max-depth` / `-p` `<depth>`: Maximum subfolder depth relative to the target directory to search (default: `2`, where 0 means only base directory files). Can also be overridden globally using the environment variable `SUMARON_MAX_FILE_DEPTH`.
*   `--images`: Enable automatic image generation (default: `true`).
*   `--no-images`, `--no-imagen`: Disable automatic image generation (can also be disabled with environment variable `SUMARON_GENERATE_IMAGES=false`).
*   `--force-images`: Force re-generating images even if they already exist in the target `assets/` folder.
*   `--imagen-model` `<model-name>`: The Google Imagen model to use for image generation (default: `imagen-3.0-generate-002`, can also be configured with `SUMARON_IMAGEN_MODEL`).
*   `--version` / `-v`: Print the current `sumaron` version and exit.

## Image Generation Capabilities 🎨

When enabled (default), `sumaron` automatically checks for and creates visual assets in the `{target_dir}/assets/` directory:

1.  **`assets/logo.png`**: A modern vector logo and app icon for the project, featuring a subtle glowing **Eye of Sauron** in the bottom-right corner as a signature watermark.
2.  **`assets/arch_diagram.png`**: A clean technical architecture blueprint diagram illustrating system components, interfaces, and data flow.
3.  **`assets/er_diagram.png`**: An entity-relationship and code structure diagram showing schemas, structs/classes, and structural relationships.

If an image already exists in `assets/`, `sumaron` skips generating it to save API quota and time, unless `--force-images` is specified.

## Output Format & Metadata

When summarizing a folder, `sumaron` prints the output markdown to `stdout`, and automatically writes it to `sumaron-summary.md` inside the target directory. 

The output `sumaron-summary.md` contains a YAML frontmatter block detailing the run metadata, processed file paths, and any visual assets:

```yaml
---
sumaron_version: 0.2.0
date: 2026-08-27
path: /path/to/bin/sumaron
hostname: computer-hostname
timestamp: 2026-08-27T12:50:00+02:00
model: gemini-flash-latest
files:
  - README.md
  - docs/USER_MANUAL.md
  - main.go
images:
  - assets/logo.png
  - assets/arch_diagram.png
  - assets/er_diagram.png
---
```

Below the frontmatter, the project logo is embedded at the top, followed by the AI-generated markdown summary, and concluded with a side-by-side **Architecture & Code Structure Visuals** section displaying the generated diagrams.

## Caching Strategy

To avoid redundant calls to Gemini and Imagen:
1. `sumaron` computes a content hash (MD5) of all supported files inside the directory, sorted by relative path.
2. It looks for a matching hash in:
   - The directory's local `.sumaron.json` file.
   - The central `~/.sumaron-cache.json` registry file.
3. If a match is found, it prints `Cache hit!` to stderr and outputs the cached summary to stdout.
4. If images exist in `assets/`, they are automatically linked in `sumaron-summary.md`.
5. On a cache miss, it calls Gemini to generate the summary, calls Imagen 3 for any missing visual assets, writes `sumaron-summary.md`, and updates both caches.
