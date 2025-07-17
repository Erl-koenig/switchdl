# switchdl

`switchdl` is a CLI for downloading videos from SwitchTube.

## Installation

### From source

Ensure you have a working Go environment (Go 1.24.4+ is required).

```bash
go install github.com/Erl-koenig/switchdl@latest
```

### From release

Alternatively, you can download a pre-compiled binary for your operating system from the [GitHub Releases](https://github.com/Erl-koenig/switchdl/releases) page.

## Configuration

Before you can download videos, you need to configure `switchdl` with your SwitchTube access token.

The CLI will store it using your operating system's credential management system (macOS Keychain, Windows Credential Manager, Linux Secret Service). Once configured, you can download videos without specifiying it with the token flag. Note from SwitchTube: "Access tokens expire automatically when they have not been used for 60 days".

- **Set or Update Token:** `switchdl configure`, will prompt you to enter your access token.
- **Delete Stored Token:** `switchdl configure delete`, removes the token from your system's credential store.
- **Show Stored Token:** `switchdl configure show`, shows if an access token is currently stored or not.
- **Validate Stored Token:** `switchdl configure validate`, validates the stored token with the SwitchTube API.

## Usage

```bash
A CLI tool for downloading videos from SwitchTube

Usage:
  switchdl [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  configure   Manage your SwitchTube access token
  help        Help about any command
  version     Show the version of your CLI tool
  video       Download a video specified by its id

Flags:
  -h, --help   help for switchdl

Use "switchdl [command] --help" for more information about a command.
```

### Download a video

Use the `video` command with the video ID.The access token will be automatically retrieved from your system's credential store if configured (can be overriden with the `--token` flag).

**Command:**

```
Usage:
  switchdl video <video_id> [flags]

Flags:
  -f, --filename string     Output filename (defaults to video title)
  -h, --help                help for video
  -o, --output-dir string   Output directory path (default ".")
  -w, --overwrite           Overwrite existing files
  -s, --select-variant      List all video variants and prompt for selection
  -t, --token string        Access token for API authentication (overrides configured token)
```

### Shell Autocompletion

The `completion` command provides autocompletion scripts for various shells. To make it permanent, add these commands to your according shell config file (`~/.bashrc`, `~/.zshrc`, `~/.fishrc`, ...).

Load completions for your current session:

- **Bash:** `source <(switchdl completion bash)`
- **Zsh:** `source <(switchdl completion zsh)`
- **Fish:** `switchdl completion fish | source`
- **PowerShell:** `switchdl completion powershell | Out-String | Invoke-Expression`

## Building from Source

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/Erl-koenig/switchdl.git
    cd switchdl
    ```
2.  **Build the binary:**
    ```bash
    go build .
    ```
3.  **Run the executable:**
    ```bash
    ./switchdl --help
    ```

## License

This project is licensed under the [MIT License](LICENSE).
