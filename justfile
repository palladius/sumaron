# sumaron justfile

export GOROOT := env_var_or_default("GOROOT", "")
export PATH := if GOROOT == "" { env_var("PATH") } else { GOROOT + "/bin:" + env_var("PATH") }

# Default task: build
default: build

# Build the sumaron binary
build:
    go build -o bin/sumaron main.go
    @echo "Updating user manual on obsidian..."
    mkdir -p {{env_var("HOME")}}/obsidian-pbt/Wiki/UserManuals
    cp docs/USER_MANUAL.md {{env_var("HOME")}}/obsidian-pbt/Wiki/UserManuals/Sumaron.md

# Install the sumaron binary to ~/bin
install: build
    mkdir -p {{env_var("HOME")}}/bin
    cp bin/sumaron {{env_var("HOME")}}/bin/sumaron
    @echo "Installed sumaron to ~/bin/sumaron"

# Run tests
test:
    go test -v ./...

# Clean build artifacts
clean:
    rm -rf bin/
