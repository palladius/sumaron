# SUMARON User Manual

`sumaron` is a CLI tool that recursively summarizes the contents of a directory using the Gemini API. It uses a local and global caching mechanism to ensure that the directory is only summarized when files matching the target extensions are modified.

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

## Output Format & Metadata

When summarizing a folder, `sumaron` prints the output markdown to `stdout`, and automatically writes it to `sumaron-summary.md` inside the target directory. 

The output `sumaron-summary.md` contains a YAML frontmatter block detailing the run metadata and the processed file paths:

```yaml
---
sumaron_version: 0.1.0
date: 2026-06-25
path: /path/to/bin/sumaron
hostname: computer-hostname
timestamp: 2026-06-25T11:00:00+02:00
model: gemini-3.5-flash
files:
  - README.md
  - docs/USER_MANUAL.md
  - main.go
---
```

## Caching Strategy

To avoid redundant calls to Gemini:
1. `sumaron` computes a content hash (MD5) of all supported files inside the directory, sorted by relative path.
2. It looks for a matching hash in:
   - The directory's local `.sumaron.json` file.
   - The central `~/.sumaron-cache.json` registry file.
3. If a match is found, it prints `Cache hit!` to stderr and outputs the cached summary to stdout.
4. On a cache miss, it calls Gemini, outputs the summary, and updates both caches.
