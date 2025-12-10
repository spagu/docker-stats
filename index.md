---
layout: default
title: Home
---

# Docker Stats Monitor

A real-time terminal UI tool for monitoring Docker container statistics, similar to 'top' or 'htop' for Linux systems.

## Features

- Real-time container statistics (CPU, Memory, Network, Disk)
- Terminal UI with keyboard navigation
- Sorting by different columns
- Auto-refresh with configurable interval
- Color-coded resource usage indicators

## Architecture

<div class="mermaid">
graph TB
    A[Docker Stats Monitor] --> B[Main Process]
    B --> C[TUI Module]
    B --> D[Docker Client]
    
    C --> E[Bubbletea Framework]
    C --> F[Lipgloss Styling]
    
    D --> G[Docker API]
    G --> H[Container Stats]
    G --> I[Image Info]
    G --> J[System Info]
    
    H --> K[CPU Usage]
    H --> L[Memory Usage]
    H --> M[Network I/O]
    H --> N[Disk I/O]
    
    E --> O[Update Loop]
    O --> P[Render UI]
    O --> Q[Handle Events]
    
    style A fill:#0366d6,color:#ffffff
    style D fill:#28a745,color:#ffffff
    style G fill:#ffc107,color:#000000
</div>

## Installation

### Pre-compiled Binaries

Download the latest release from [GitHub Releases](https://github.com/spagu/docker-stats/releases):

```bash
# Linux (AMD64)
wget https://github.com/spagu/docker-stats/releases/latest/download/docker-stats-linux-amd64.tar.gz
tar -xzf docker-stats-linux-amd64.tar.gz
sudo mv docker-stats /usr/local/bin/

# macOS (Intel)
wget https://github.com/spagu/docker-stats/releases/latest/download/docker-stats-darwin-amd64.tar.gz
tar -xzf docker-stats-darwin-amd64.tar.gz
sudo mv docker-stats /usr/local/bin/

# Windows (AMD64)
wget https://github.com/spagu/docker-stats/releases/latest/download/docker-stats-windows-amd64.zip
unzip docker-stats-windows-amd64.zip
```

### From Source

```bash
git clone https://github.com/spagu/docker-stats.git
cd docker-stats
make build
sudo mv build/docker-stats /usr/local/bin/
```

## Usage

### Basic Usage

```bash
# Run with default settings
docker-stats

# Show all containers (including stopped)
docker-stats -all

# Set custom refresh interval
docker-stats -interval 5s

# One-shot output (no TUI)
docker-stats -once
```

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| q, Ctrl+C | Quit |
| r | Force refresh |
| c | Sort by CPU |
| m | Sort by Memory |
| n | Sort by Name |
| ↑/↓ | Navigate containers |
| Enter | Show container details |

## Data Flow

<div class="mermaid">
sequenceDiagram
    participant UI as TUI
    participant Main as Main Process
    participant Client as Docker Client
    participant API as Docker API
    
    UI->>Main: Start application
    Main->>Client: Create client
    Main->>API: Ping Docker daemon
    API-->>Main: Connection OK
    
    loop Every 2 seconds
        Main->>Client: Get container stats
        Client->>API: ContainerList
        API-->>Client: Container list
        Client->>API: ContainerStats (each)
        API-->>Client: Stats data
        Client-->>Main: Processed stats
        Main->>UI: Update display
    end
    
    UI->>Main: User input
    Main->>UI: Handle action
</div>

## Requirements

- Docker daemon running
- User must have permissions to access Docker socket
  (typically member of 'docker' group or root)

## Development

```bash
# Install development tools
make dev-tools

# Run all checks
make check

# Build for multiple platforms
make build-all

# Run tests
make test
```

## License

This project is licensed under the BSD 3-Clause License - see the [LICENSE](LICENSE) file for details.

## Badges

<div class="badges">
  <img src="https://img.shields.io/github/v/release/spagu/docker-stats" alt="Release">
  <img src="https://img.shields.io/github/workflow/status/spagu/docker-stats/CI" alt="Build Status">
  <img src="https://img.shields.io/codecov/c/github/spagu/docker-stats" alt="Coverage">
  <img src="https://img.shields.io/github/downloads/spagu/docker-stats/total" alt="Downloads">
  <img src="https://img.shields.io/badge/License-BSD%203--Clause-blue.svg" alt="License">
</div>
