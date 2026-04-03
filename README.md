# gh-projects

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A TUI tool for managing GitHub Projects (V2) directly from your terminal.

## Features

- **Kanban Board View**: Organize cards into Todo, In Progress, and Done columns.
- **Issue Detail Panel**: Read issue descriptions with full markdown rendering.
- **Card Actions**: Move cards between columns, close or reopen issues.
- **Task Management**: Assign users, add or remove labels.
- **Comments**: View existing comments and add new ones.
- **PR Integration**: Link PRs, view linked PRs, and add PRs to projects.
- **Navigation**: Search and filter cards with a quick fuzzy finder.
- **Multi-project Support**: Interactive picker or direct access via CLI flags.
- **Caching**: Local in-memory session cache for fast navigation.
- **Neovim Integration**: Open your project board directly within Neovim.

## Installation

Install as a GitHub CLI extension:

```bash
gh extension install ifloresarg/gh-projects
```

### Requirements

- `gh` CLI installed.
- Proper authentication scope:

  ```bash
  gh auth refresh -s project
  ```

## Usage

Run the extension from your terminal:

```bash
gh projects
```

### CLI Flags

- `--owner <owner>`: Specify the GitHub owner (user or organization).
- `--number <number>`: Specify the project number.

### Keybindings

| Key | Action |
| --- | --- |
| `k` / `↑` | Move selection up |
| `j` / `↓` | Move selection down |
| `h` / `←` | Move selection left (between columns) |
| `l` / `→` | Move selection right (between columns) |
| `enter` | Select project or open card details |
| `esc` | Go back |
| `q` / `ctrl+c` | Quit |
| `?` | Toggle help |
| `R` | Manual refresh |
| `/` | Search / Filter |
| `H` | Move card left |
| `L` | Move card right |

Additional actions (Issue Detail View):

- `u`: Copy issue/PR URL to clipboard
- `a`: Assign / Unassign users
- `L`: Add / Remove labels
- `x` / `X`: Close / Reopen issue
- `c`: Add comment
- `p`: PR management

## Configuration

Settings are stored in `~/.config/gh-projects/config.yaml`.

```yaml
default_owner: "ifloresarg"
default_project: 0
cache_ttl: 300
```

- `default_owner`: The default GitHub user or organization to search for projects.
- `default_project`: The default project number to load on startup.
- `cache_ttl`: Cache duration in seconds (default: 300).

## Neovim Integration

You can use `gh-projects` inside Neovim as a floating terminal.

### lazy.nvim example

```lua
return {
  "ifloresarg/gh-projects",
  config = function()
    require("gh-projects").setup({
      binary = "gh projects",
      width = 0.9,
      height = 0.95,
      border = "rounded",
    })
  end,
  keys = {
    { "<leader>gp", "<cmd>GhProjects<cr>", desc = "Open GitHub Projects" },
  },
}
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and guidelines.

## License

MIT - See [LICENSE](LICENSE) for details.
