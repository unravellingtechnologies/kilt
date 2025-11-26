# Kilt — Product Definition Document  
*A minimalist, Git-first, curl-installable bootstrapper for perfect Mac (and optionally Linux) setups*  
November 25, 2025  

## Vision  
One command to turn any fresh Mac into *your* Mac in under 5 minutes — completely from a single Git repository, with zero secrets ever touching Git, full idempotency, and the flexibility to manage arbitrary files in arbitrary locations.

## Tagline  
`curl -sL https://get.kilt.pro | bash -s -- https://github.com/you/dotfiles`

## Core Principles  
- Git is the source of truth — no external backends  
- Zero secrets in Git (even encrypted)  
- Idempotent & safe to run 100 times  
- Works offline after first install  
- No Ruby/Python/Go runtime dependencies after bootstrap  
- macOS-first, Linux-friendly  

## Feature Breakdown  

| # | Feature | Description | Implementation Notes |
|---|---------|-------------|----------------------|
| 1 | **One-line curl installer** | `curl … | bash -s -- <repo-url>` clones and bootstraps everything | Pure Bash installer that works over HTTPS or SSH |
| 2 | **Bare Git repository** | Dotfiles live in a bare Git repo at `~/.dotfiles` (or custom path) | Uses the proven “bare repo + alias” pattern (Mathias-style) |
| 3 | **Flexible file placement** | Files can be placed anywhere on the filesystem with explicit target paths | Config uses `source → target` mapping (supports `~` expansion) |
| 4 | **Arbitrary directory support** | Example: `~/.config/mise/config.toml` ← `./mise/config.toml` | No limitation to $HOME dotfiles |
| 5 | **Run-once bootstrap tasks** | Executed exactly once (tracked via `.kilt/done/<task-id>`) | Install Homebrew, clone extra repos, create folder skeletons, macOS defaults, etc. |
| 6 | **On-demand commands** | `dot sync` → pull latest + apply changes + run onchange tasks | Additional commands: `dot brew`, `dot update`, `dot doctor` |
| 7 | **Declarative extra repositories** | List of external Git repos to clone into specific locations | Example: `~/Projects/zsh-plugins/fast-syntax-highlighting` |
| 8 | **Folder skeleton creation** | Declarative list of directories to always ensure exist | `~/dev`, `~/screenshots`, `~/Documents/notes`, etc. |
| 9 | **1Password CLI secret injection** | Secrets are never stored in Git — pulled live from 1Password during apply | Uses `op read` with templating: `{{ op://Private/github-token }}` |
| 10 | **Templating (Go templates)** | Full Go template support in any file with machine-specific variables | Built-in vars: `{{ .hostname }}`, `{{ .os }}`, `{{ .arch }}`, custom JSON/YAML data file |
| 11 | **OS & hostname alternates** | Automatic file selection: `file.mac.zsh`, `file_linux.zsh`, `file.hostname@work.zsh` | yadm-style alternates without needing yadm |
| 12 | **Idempotency & dry-run** | `dot sync --dry-run` shows exactly what will happen | All tasks are naturally idempotent |
| 13 | **Pre-apply backup** | Existing files are backed up to `~/.kilt/backup/YYYYMMDD-HHMMSS/` before overwrite | Never silently destroys user data |
| 14 | **Post-apply validation** | Optional `validate.sh` that runs shellcheck, brew doctor, zsh -n, etc. | `dot doctor` command |
| 15 | **Auto-update hook** | Optional weekly self-update + brew upgrade + cleanup | Can be enabled with one flag |
| 16 | **Zero runtime dependencies after bootstrap** | Only requires Git and Bash (Homebrew + 1Password CLI are installed if missing) | Keeps it fast and portable |
| 17 | **Single binary fallback (optional)** | Tiny Go binary (`dot`) can be built for users who prefer no Bash wrapper | 100% feature parity with Bash version |

## Configuration File (config.yaml example)

```yaml
# ~/.kilt/config.yaml
data_file: data.yaml                     # optional host-specific variables
template_engine: go

files:
  - source: zsh/zshrc
    target: ~/.zshrc
  - source: mise/config.toml
    target: ~/.config/mise/config.toml
  - source: gitconfig
    target: ~/.gitconfig
    template: true                       # contains {{ op://… }} secrets

extra_repos:
  - url: https://github.com/zdharma-continuum/fast-syntax-highlighting
    path: ~/.zsh/fast-syntax-highlighting
    sparse: true

directories:
  - ~/dev/personal
  - ~/dev/work
  - ~/screenshots
  - ~/Documents/notes

run_once:
  - install_homebrew.sh
  - install_1password_cli.sh
  - setup_macos_defaults.sh

on_change:
  - brew bundle --file=Brewfile
  - mise install --yes
