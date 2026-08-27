package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

const Version = "0.2.0"

type SumaronCache struct {
	Timestamp string `json:"timestamp"`
	Hash      string `json:"hash"`
	Summary   string `json:"summary"`
}

type CentralCache map[string]SumaronCache

var (
	styleSuccess = lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)
	styleWarn    = lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true)
	styleInfo    = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	stylePath    = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Underline(true)
	styleHash    = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	styleModel   = lipgloss.NewStyle().Foreground(lipgloss.Color("211"))
	styleTitle   = lipgloss.NewStyle().Foreground(lipgloss.Color("197")).Bold(true)
)

// Default extensions to summarize
const defaultExtensions = ".md,.html,.json"

// Model to use for text summarization
const defaultModel = "gemini-flash-latest"

// Model to use for image generation
const defaultImageModel = "gemini-2.5-flash-image"

func main() {
	// Get defaults from env or constants
	defaultMaxFiles := 20
	if envMax := os.Getenv("SUMARON_MAX_FILES"); envMax != "" {
		if val, err := strconv.Atoi(envMax); err == nil {
			defaultMaxFiles = val
		}
	}

	defaultMaxDepth := 2
	if envDepth := os.Getenv("SUMARON_MAX_FILE_DEPTH"); envDepth != "" {
		if val, err := strconv.Atoi(envDepth); err == nil {
			defaultMaxDepth = val
		}
	}

	defaultGenerateImages := true
	if envImages := os.Getenv("SUMARON_GENERATE_IMAGES"); envImages != "" {
		envLower := strings.ToLower(envImages)
		if envLower == "false" || envLower == "0" || envLower == "no" {
			defaultGenerateImages = false
		}
	}

	defaultImgModel := defaultImageModel
	if envImgModel := os.Getenv("SUMARON_IMAGE_MODEL"); envImgModel != "" {
		defaultImgModel = envImgModel
	} else if envImgModel := os.Getenv("SUMARON_IMAGEN_MODEL"); envImgModel != "" {
		defaultImgModel = envImgModel
	}

	var (
		dir          string
		key          string
		model        string
		force        bool
		globalCache  string
		extensions   string
		maxFiles     int
		maxDepth     int
		showVersion  bool
		enableImages bool
		noImages     bool
		noImagen     bool
		forceImages  bool
		imageModel   string
	)

	// Set custom usage function
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags]\n\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "Flags:\n")
		fmt.Fprintf(os.Stderr, "  -f, --folder PATH=\".\"        Directory to summarize\n")
		fmt.Fprintf(os.Stderr, "  -k, --key KEY                 Gemini API key (env: GEMINI_API_KEY)\n")
		fmt.Fprintf(os.Stderr, "  -m, --model MODEL=\"%s\"      Gemini model to use for summarization\n", defaultModel)
		fmt.Fprintf(os.Stderr, "      --force                   Force re-summarization, ignoring cache\n")
		fmt.Fprintf(os.Stderr, "  -g, --global-cache PATH=\"~/.sumaron-cache.json\"  Custom path to central cache file\n")
		fmt.Fprintf(os.Stderr, "  -e, --extensions EXTS=\"%s\"    Comma-separated list of file extensions to include\n", defaultExtensions)
		fmt.Fprintf(os.Stderr, "  -n, --max-files N=%d          Maximum number of files to summarize (env: SUMARON_MAX_FILES)\n", defaultMaxFiles)
		fmt.Fprintf(os.Stderr, "  -p, --max-depth N=%d          Maximum folder depth to recursively walk (env: SUMARON_MAX_FILE_DEPTH)\n", defaultMaxDepth)
		fmt.Fprintf(os.Stderr, "      --images                  Enable automatic image generation (logo, architecture, ER) [default: true]\n")
		fmt.Fprintf(os.Stderr, "      --no-images, --no-imagen  Disable automatic image generation (env: SUMARON_GENERATE_IMAGES=false)\n")
		fmt.Fprintf(os.Stderr, "      --force-images            Force re-generating images even if they already exist in assets/\n")
		fmt.Fprintf(os.Stderr, "      --image-model, --imagen-model MODEL=\"%s\"  Image generation model to use (env: SUMARON_IMAGE_MODEL)\n", defaultImageModel)
		fmt.Fprintf(os.Stderr, "  -v, --version                 Show version and exit\n")
	}

	flag.StringVar(&dir, "folder", ".", "Directory to summarize")
	flag.StringVar(&dir, "f", ".", "Directory to summarize")
	flag.StringVar(&dir, "dir", ".", "Directory to summarize (compatibility)")
	flag.StringVar(&dir, "d", ".", "Directory to summarize (compatibility)")

	flag.StringVar(&key, "key", "", "Gemini API key")
	flag.StringVar(&key, "k", "", "Gemini API key")

	flag.StringVar(&model, "model", defaultModel, "Gemini model to use")
	flag.StringVar(&model, "m", defaultModel, "Gemini model to use")

	flag.BoolVar(&force, "force", false, "Force re-summarization, ignoring cache")

	flag.StringVar(&globalCache, "global-cache", "", "Custom path to central cache file")
	flag.StringVar(&globalCache, "g", "", "Custom path to central cache file")

	flag.StringVar(&extensions, "extensions", defaultExtensions, "Comma-separated list of file extensions to include")
	flag.StringVar(&extensions, "e", defaultExtensions, "Comma-separated list of file extensions to include")

	flag.IntVar(&maxFiles, "max-files", defaultMaxFiles, "Maximum number of files to summarize")
	flag.IntVar(&maxFiles, "max-file", defaultMaxFiles, "Maximum number of files to summarize")
	flag.IntVar(&maxFiles, "n", defaultMaxFiles, "Maximum number of files to summarize")

	flag.IntVar(&maxDepth, "max-file-depth", defaultMaxDepth, "Maximum folder depth to recursively walk")
	flag.IntVar(&maxDepth, "max-depth", defaultMaxDepth, "Maximum folder depth to recursively walk")
	flag.IntVar(&maxDepth, "p", defaultMaxDepth, "Maximum folder depth to recursively walk")

	flag.BoolVar(&enableImages, "images", defaultGenerateImages, "Enable automatic image generation")
	flag.BoolVar(&noImages, "no-images", false, "Disable automatic image generation")
	flag.BoolVar(&noImagen, "no-imagen", false, "Disable automatic image generation")
	flag.BoolVar(&forceImages, "force-images", false, "Force re-generating images")
	flag.StringVar(&imageModel, "image-model", defaultImgModel, "Image generation model to use")
	flag.StringVar(&imageModel, "imagen-model", defaultImgModel, "Image generation model to use (alias)")

	flag.BoolVar(&showVersion, "version", false, "Show version and exit")
	flag.BoolVar(&showVersion, "v", false, "Show version and exit")

	flag.Parse()

	if showVersion {
		fmt.Printf("sumaron version %s\n", Version)
		return
	}

	if noImages || noImagen {
		enableImages = false
	}

	// Get API Key
	apiKey := key
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if apiKey == "" {
		log.Fatalf("Error: GEMINI_API_KEY is not set. Please provide it via the GEMINI_API_KEY environment variable or using --key flag.\nUse --help for usage details.")
	}

	// Resolve absolute path of directory to summarize
	absDir, err := filepath.Abs(dir)
	if err != nil {
		log.Fatalf("Error resolving absolute path of %s: %v", dir, err)
	}

	// Verify directory exists
	info, err := os.Stat(absDir)
	if err != nil {
		log.Fatalf("Error: directory %s does not exist or is not accessible: %v", absDir, err)
	}
	if !info.IsDir() {
		log.Fatalf("Error: path %s is a file, not a directory", absDir)
	}

	// Parse extensions
	extList := strings.Split(extensions, ",")
	allowedExts := make(map[string]bool)
	for _, ext := range extList {
		ext = strings.TrimSpace(strings.ToLower(ext))
		if ext != "" {
			if !strings.HasPrefix(ext, ".") {
				ext = "." + ext
			}
			allowedExts[ext] = true
		}
	}

	// Compute directory contents and deterministic hash
	fileList, hash, err := computeDirectoryHash(absDir, allowedExts, maxFiles, maxDepth)
	if err != nil {
		log.Fatalf("Error computing directory hash: %v", err)
	}

	if len(fileList) == 0 {
		fmt.Printf("%s No files matching extensions (%s) found in %s. Nothing to summarize.\n",
			styleWarn.Render("⚠️"),
			styleInfo.Render(extensions),
			stylePath.Render(absDir),
		)
		return
	}

	// Determine cache files
	localCachePath := filepath.Join(absDir, ".sumaron.json")
	centralCachePath := globalCache
	if centralCachePath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Printf("Warning: could not find user home directory: %v", err)
		} else {
			centralCachePath = filepath.Join(homeDir, ".sumaron-cache.json")
		}
	}

	// Check cache
	if !force {
		cachedSummary, found := checkCache(localCachePath, centralCachePath, absDir, hash)
		if found {
			// If force-images was explicitly specified, generate missing/forced images even on cache hit
			if enableImages && forceImages {
				generateProjectImages(context.Background(), apiKey, imageModel, absDir, cachedSummary, forceImages)
			}

			// Print cache hit message to stderr, and summary to stdout
			fmt.Fprintf(os.Stderr, "%s %s Reusing summary of %s (hash: %s)\n",
				styleSuccess.Render("🎯"),
				styleSuccess.Render("Cache hit!"),
				stylePath.Render(absDir),
				styleHash.Render(hash[:8]),
			)
			fmt.Println(cachedSummary)
			writeSummaryMarkdown(absDir, cachedSummary, model, fileList)
			return
		}
	}

	// Generate prompt content
	prompt, err := buildPrompt(absDir, fileList)
	if err != nil {
		log.Fatalf("Error building prompt: %v", err)
	}

	fmt.Fprintf(os.Stderr, "%s %s %s The Eye of Sumaron sees all on %s. Summarizing files using %s:\n",
		styleWarn.Render("🎯"),
		styleWarn.Render("Cache miss!"),
		styleTitle.Render("👁️"),
		stylePath.Render(absDir),
		styleModel.Render(model),
	)
	for _, file := range fileList {
		relPath, _ := filepath.Rel(absDir, file)
		fmt.Fprintf(os.Stderr, "  🔥 Summarizing %s...\n", styleInfo.Render(relPath))
	}

	// Start spinner for text summary
	spinnerCancel := showSpinner(context.Background(), "Consulting the Eye of Sumaron...")

	// Call Gemini
	ctx := context.Background()
	summary, err := summarizeWithGemini(ctx, apiKey, model, prompt)
	spinnerCancel()

	if err != nil {
		log.Fatalf("Error generating summary with Gemini: %v", err)
	}

	// Generate images if enabled
	if enableImages {
		generateProjectImages(ctx, apiKey, imageModel, absDir, summary, forceImages)
	}

	// Print summary
	fmt.Println(summary)

	// Save summary markdown file
	writeSummaryMarkdown(absDir, summary, model, fileList)

	// Save to cache
	now := time.Now().Format(time.RFC3339)
	cacheEntry := SumaronCache{
		Timestamp: now,
		Hash:      hash,
		Summary:   summary,
	}

	// Write local cache
	if err := saveLocalCache(localCachePath, cacheEntry); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save local cache: %v\n", err)
	}

	// Write central cache
	if centralCachePath != "" {
		if err := saveCentralCache(centralCachePath, absDir, cacheEntry); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save central cache: %v\n", err)
		}
	}
}

// computeDirectoryHash walks the directory recursively, finds all files with allowed extensions,
// sorts them, and computes a SHA/MD5 hash representing the directory state.
func computeDirectoryHash(baseDir string, allowedExts map[string]bool, maxFiles, maxDepth int) ([]string, string, error) {
	var files []string

	err := filepath.WalkDir(baseDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories (starting with ".") and standard noise directories
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") && name != "." && name != ".." {
				return filepath.SkipDir
			}
			if name == "node_modules" || name == "vendor" || name == "bin" || name == "venv" || name == ".venv" || name == ".git" {
				return filepath.SkipDir
			}

			// Check directory depth relative to baseDir
			relDir, err := filepath.Rel(baseDir, path)
			if err == nil && relDir != "." && relDir != ".." {
				depth := strings.Count(relDir, string(filepath.Separator))
				if depth > maxDepth {
					return filepath.SkipDir
				}
			}
			return nil
		}

		// Check file depth relative to baseDir
		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return nil
		}
		depth := strings.Count(relPath, string(filepath.Separator))
		if depth > maxDepth {
			return nil
		}

		// Check file extension
		ext := strings.ToLower(filepath.Ext(path))
		if allowedExts[ext] {
			name := d.Name()
			if name == ".sumaron.json" || name == "sumaron-summary.md" {
				return nil
			}
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		return nil, "", err
	}

	// Sort files to make the hashing deterministic
	sort.Strings(files)

	// Enforce max files limit
	if len(files) > maxFiles {
		fmt.Fprintf(os.Stderr, "Warning: Directory contains %d files. Limiting summarization to the first %d files (alphabetically sorted).\n", len(files), maxFiles)
		files = files[:maxFiles]
	}

	// Compute MD5 hash of relative path, size, and content of all files
	hasher := md5.New()
	for _, file := range files {
		relPath, err := filepath.Rel(baseDir, file)
		if err != nil {
			return nil, "", err
		}

		info, err := os.Stat(file)
		if err != nil {
			return nil, "", err
		}

		// Ignore .sumaron.json itself to prevent feedback loop
		if relPath == ".sumaron.json" {
			continue
		}

		// Write relative path and size to hash
		hasher.Write([]byte(relPath))
		hasher.Write([]byte(fmt.Sprintf(":%d:", info.Size())))

		// Write content
		f, err := os.Open(file)
		if err != nil {
			return nil, "", err
		}
		_, err = io.Copy(hasher, f)
		f.Close()
		if err != nil {
			return nil, "", err
		}
	}

	hashBytes := hasher.Sum(nil)
	return files, hex.EncodeToString(hashBytes), nil
}

// checkCache checks the local .sumaron.json file and central cache to see if the directory has been summarized.
func checkCache(localPath, centralPath, absDir, hash string) (string, bool) {
	// 1. Try local cache
	if localPath != "" {
		if data, err := os.ReadFile(localPath); err == nil {
			var cache SumaronCache
			if err := json.Unmarshal(data, &cache); err == nil {
				if cache.Hash == hash && cache.Summary != "" {
					return cache.Summary, true
				}
			}
		}
	}

	// 2. Try central cache
	if centralPath != "" {
		if data, err := os.ReadFile(centralPath); err == nil {
			var central CentralCache
			if err := json.Unmarshal(data, &central); err == nil {
				if cache, exists := central[absDir]; exists {
					if cache.Hash == hash && cache.Summary != "" {
						return cache.Summary, true
					}
				}
			}
		}
	}

	return "", false
}

// buildPrompt reads the content of all files and structures them into a prompt.
func buildPrompt(baseDir string, files []string) (string, error) {
	var builder strings.Builder

	builder.WriteString("You are an expert technical writer and codebase summarizer. Below is the content of a directory that needs to be summarized.\n")
	builder.WriteString(fmt.Sprintf("Directory: %s\n\n", baseDir))
	builder.WriteString("Here are the files in the directory:\n\n")

	for _, file := range files {
		relPath, err := filepath.Rel(baseDir, file)
		if err != nil {
			return "", err
		}

		// Don't read the local cache file
		if relPath == ".sumaron.json" {
			continue
		}

		contentBytes, err := os.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("failed to read file %s: %w", file, err)
		}

		// Truncate file content if it is ridiculously long to prevent token overflow
		content := string(contentBytes)
		if len(content) > 50000 {
			content = content[:50000] + "\n... [CONTENT TRUNCATED FOR LENGTH] ..."
		}

		builder.WriteString(fmt.Sprintf("--- FILE: %s ---\n", relPath))
		builder.WriteString(content)
		builder.WriteString(fmt.Sprintf("\n--- END FILE: %s ---\n\n", relPath))
	}

	builder.WriteString("Please provide a concise, high-level summary of the purpose, structure, and key contents of this directory. ")
	builder.WriteString("Mention what kind of project it is, its main components, and any important configuration files or entries found. ")
	builder.WriteString("Format the output in clean Markdown with bullet points, using emojis where appropriate.")

	return builder.String(), nil
}

// summarizeWithGemini calls the Google Gemini API to generate the summary.
func summarizeWithGemini(ctx context.Context, apiKey, modelName, prompt string) (string, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return "", fmt.Errorf("failed to create Gemini client: %w", err)
	}
	defer client.Close()

	model := client.GenerativeModel(modelName)
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", errors.New("empty response from Gemini API")
	}

	var partTexts []string
	for _, part := range resp.Candidates[0].Content.Parts {
		if textPart, ok := part.(genai.Text); ok {
			partTexts = append(partTexts, string(textPart))
		}
	}

	return strings.Join(partTexts, "\n"), nil
}

// generateImageWithModel calls the Gemini generateContent API (or predict API) to generate an image.
func generateImageWithModel(ctx context.Context, apiKey, imageModel, prompt string) ([]byte, error) {
	// 1. Try generateContent (standard for Gemini image models like gemini-2.5-flash-image)
	apiURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		url.PathEscape(imageModel), url.QueryEscape(apiKey))

	generateReq := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{"text": prompt},
				},
			},
		},
	}

	jsonBytes, err := json.Marshal(generateReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal image request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{Timeout: 90 * time.Second}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("image http request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("image API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var generateResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					InlineData *struct {
						MimeType string `json:"mimeType"`
						Data     string `json:"data"`
					} `json:"inlineData"`
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		Predictions []struct {
			BytesBase64Encoded string `json:"bytesBase64Encoded"`
		} `json:"predictions"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Status  string `json:"status"`
		} `json:"error,omitempty"`
	}

	if err := json.Unmarshal(respBody, &generateResp); err != nil {
		return nil, fmt.Errorf("failed to parse response JSON: %w", err)
	}

	if generateResp.Error != nil {
		return nil, fmt.Errorf("image API error (%d %s): %s", generateResp.Error.Code, generateResp.Error.Status, generateResp.Error.Message)
	}

	// Check for inlineData inside candidates
	for _, cand := range generateResp.Candidates {
		for _, part := range cand.Content.Parts {
			if part.InlineData != nil && part.InlineData.Data != "" {
				imgBytes, err := base64.StdEncoding.DecodeString(part.InlineData.Data)
				if err != nil {
					return nil, fmt.Errorf("failed to decode inlineData base64: %w", err)
				}
				return imgBytes, nil
			}
		}
	}

	// Check for predictions (predict endpoint format)
	if len(generateResp.Predictions) > 0 && generateResp.Predictions[0].BytesBase64Encoded != "" {
		imgBytes, err := base64.StdEncoding.DecodeString(generateResp.Predictions[0].BytesBase64Encoded)
		if err != nil {
			return nil, fmt.Errorf("failed to decode prediction base64: %w", err)
		}
		return imgBytes, nil
	}

	return nil, errors.New("no image data returned in API response")
}

// generateProjectImages generates logo, arch diagram, and er diagram if missing.
func generateProjectImages(ctx context.Context, apiKey, imageModel, absDir, summary string, forceImages bool) {
	assetsDir := filepath.Join(absDir, "assets")
	logoPath := filepath.Join(assetsDir, "logo.png")
	archPath := filepath.Join(assetsDir, "arch_diagram.png")
	erPath := filepath.Join(assetsDir, "er_diagram.png")

	projectName := filepath.Base(absDir)
	if projectName == "." || projectName == "/" || projectName == "" {
		projectName = "Project"
	}

	// Extract a concise summary snippet for context
	cleanSummary := strings.ReplaceAll(summary, "\n", " ")
	if len(cleanSummary) > 200 {
		cleanSummary = cleanSummary[:200] + "..."
	}

	type imageTarget struct {
		name   string
		path   string
		prompt string
		emoji  string
	}

	targets := []imageTarget{
		{
			name:  "logo.png",
			path:  logoPath,
			emoji: "🎨",
			prompt: fmt.Sprintf(
				"A sleek modern vector logo and app icon for a software project named '%s'. Clean minimalist developer branding aesthetic, elegant geometry, vibrant colors on a dark background. Concept: %s. Special detail: In the bottom right corner, as a playful subtle signature watermark, there is a small glowing fiery Eye of Sauron.",
				projectName, cleanSummary,
			),
		},
		{
			name:  "arch_diagram.png",
			path:  archPath,
			emoji: "🗺️",
			prompt: fmt.Sprintf(
				"A clean, professional technical software architecture diagram for '%s' (%s). High-tech system blueprint showing components, data pipelines, caching layers, CLI interfaces, and AI services connected by clean glowing lines and arrows on a dark slate technical background. Crisp, highly legible technical illustration.",
				projectName, cleanSummary,
			),
		},
		{
			name:  "er_diagram.png",
			path:  erPath,
			emoji: "📊",
			prompt: fmt.Sprintf(
				"A modern, clear entity-relationship and code structure diagram for '%s' (%s). Showing software entities, structs, classes, modules, and their relationships with clean connecting lines and schema boxes on a modern technical background. Polished developer documentation infographic.",
				projectName, cleanSummary,
			),
		},
	}

	for _, t := range targets {
		// Check if image already exists
		if !forceImages {
			if _, err := os.Stat(t.path); err == nil {
				// File exists, skip
				continue
			}
		}

		// Ensure assets directory exists
		if err := os.MkdirAll(assetsDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create assets directory: %v\n", err)
			return
		}

		spinnerCancel := showSpinner(ctx, fmt.Sprintf("Generating %s with %s...", t.name, imageModel))
		imgBytes, err := generateImageWithModel(ctx, apiKey, imageModel, t.prompt)
		spinnerCancel()

		if err != nil {
			fmt.Fprintf(os.Stderr, "%s Warning: failed to generate %s: %v\n", styleWarn.Render("⚠️"), t.name, err)
			continue
		}

		if err := os.WriteFile(t.path, imgBytes, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "%s Warning: failed to save %s: %v\n", styleWarn.Render("⚠️"), t.name, err)
		} else {
			relPath, _ := filepath.Rel(absDir, t.path)
			fmt.Fprintf(os.Stderr, "%s %s Generated %s\n",
				styleSuccess.Render("✨"),
				t.emoji,
				stylePath.Render(relPath),
			)
		}
	}
}

// saveLocalCache saves the cache entry in the target directory.
func saveLocalCache(path string, cache SumaronCache) error {
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// saveCentralCache adds/updates the cache entry in the central cache file.
func saveCentralCache(path string, absDir string, cache SumaronCache) error {
	central := make(CentralCache)

	// Read existing central cache
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &central)
	}

	// Update entry
	central[absDir] = cache

	// Write back
	data, err := json.MarshalIndent(central, "", "  ")
	if err != nil {
		return err
	}

	// Ensure parent directory of central cache exists
	parentDir := filepath.Dir(path)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// writeSummaryMarkdown writes the summary to sumaron-summary.md in the target directory with YAML frontmatter.
func writeSummaryMarkdown(absDir, summary, model string, files []string) {
	summaryPath := filepath.Join(absDir, "sumaron-summary.md")

	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	// Get binary path
	binPath, err := os.Executable()
	if err != nil {
		binPath = "unknown"
	}

	now := time.Now()
	dateStr := now.Format("2006-01-02")
	timestampStr := now.Format(time.RFC3339)

	// Check which images exist in assets/
	assetsDir := filepath.Join(absDir, "assets")
	logoExists := false
	archExists := false
	erExists := false
	var existingImages []string

	if _, err := os.Stat(filepath.Join(assetsDir, "logo.png")); err == nil {
		logoExists = true
		existingImages = append(existingImages, "assets/logo.png")
	}
	if _, err := os.Stat(filepath.Join(assetsDir, "arch_diagram.png")); err == nil {
		archExists = true
		existingImages = append(existingImages, "assets/arch_diagram.png")
	}
	if _, err := os.Stat(filepath.Join(assetsDir, "er_diagram.png")); err == nil {
		erExists = true
		existingImages = append(existingImages, "assets/er_diagram.png")
	}

	var builder strings.Builder
	builder.WriteString("---\n")
	builder.WriteString(fmt.Sprintf("sumaron_version: %s\n", Version))
	builder.WriteString(fmt.Sprintf("date: %s\n", dateStr))
	builder.WriteString(fmt.Sprintf("path: %s\n", binPath))
	builder.WriteString(fmt.Sprintf("hostname: %s\n", hostname))
	builder.WriteString(fmt.Sprintf("timestamp: %s\n", timestampStr))
	builder.WriteString(fmt.Sprintf("model: %s\n", model))

	// Add files array in frontmatter
	builder.WriteString("files:\n")
	for _, file := range files {
		relPath, err := filepath.Rel(absDir, file)
		if err != nil {
			relPath = file
		}
		builder.WriteString(fmt.Sprintf("  - %s\n", relPath))
	}

	// Add images array in frontmatter if any exist
	if len(existingImages) > 0 {
		builder.WriteString("images:\n")
		for _, img := range existingImages {
			builder.WriteString(fmt.Sprintf("  - %s\n", img))
		}
	}

	builder.WriteString("---\n\n")

	// Embed logo at top if present
	if logoExists {
		builder.WriteString("<p align=\"center\">\n")
		builder.WriteString("  <img src=\"assets/logo.png\" alt=\"Project Logo\" width=\"180\" />\n")
		builder.WriteString("</p>\n\n")
	}

	// Main summary content
	builder.WriteString(summary)
	builder.WriteString("\n")

	// Embed architecture & ER diagrams if present
	if archExists || erExists {
		builder.WriteString("\n---\n\n")
		builder.WriteString("## 🗺️ Architecture & Code Structure Visuals\n\n")
		if archExists && erExists {
			builder.WriteString("| Architecture Diagram | Code Structure & E/R Diagram |\n")
			builder.WriteString("| :---: | :---: |\n")
			builder.WriteString("| ![Architecture Diagram](assets/arch_diagram.png) | ![Code Structure & E/R Diagram](assets/er_diagram.png) |\n")
		} else if archExists {
			builder.WriteString("### Architecture Diagram\n\n")
			builder.WriteString("![Architecture Diagram](assets/arch_diagram.png)\n")
		} else if erExists {
			builder.WriteString("### Code Structure & E/R Diagram\n\n")
			builder.WriteString("![Code Structure & E/R Diagram](assets/er_diagram.png)\n")
		}
	}

	if err := os.WriteFile(summaryPath, []byte(builder.String()), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to write summary markdown file: %v\n", err)
	}
}

// showSpinner displays a Braille spinner in os.Stderr.
// Returns a cancel function that stops the spinner and clears the line.
func showSpinner(ctx context.Context, message string) context.CancelFunc {
	ctx, cancel := context.WithCancel(ctx)
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	go func() {
		i := 0
		for {
			select {
			case <-ctx.Done():
				// Clear the line
				fmt.Fprintf(os.Stderr, "\r\033[K")
				return
			default:
				fmt.Fprintf(os.Stderr, "\r%s %s", styleTitle.Render(frames[i]), message)
				i = (i + 1) % len(frames)
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()
	return cancel
}
