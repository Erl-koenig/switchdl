# switchdl

`switchdl` is a CLI for downloading videos from SwitchTube.

## Installation

### From release

Download a pre-compiled binary for your operating system from the [GitHub Releases](https://github.com/Erl-koenig/switchdl/releases) page.

### Using Homebrew (macOS)

If you have Homebrew installed, install using the following tap:

```bash
brew install erlkoenig/homebrew-switchdl/switchdl
```

### From source

Ensure you have a working Go environment (Go 1.24.4+ is required). Note: The `version` command will show `dev` as it is built from source.

```bash
go install github.com/Erl-koenig/switchdl@latest
```

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
  channel     Download videos from one or multiple channels
  completion  Generate the autocompletion script for the specified shell
  configure   Manage your SwitchTube access token
  help        Help about any command
  version     Show the version of switchdl
  video       Download one or more videos specified by their id

Flags:
  -h, --help                help for switchdl
  -o, --output-dir string   Output directory path (default ".")
  -w, --overwrite           Force overwrite of existing files
  -v, --select-variant      List all video variants (quality) and prompt for selection
  -s, --skip                Skip existing files
  -t, --token string        Access token for API authentication (overrides configured token)

Use "switchdl [command] --help" for more information about a command.
```

### Download a video

```bash
Download one or more videos specified by their id

Usage:
  switchdl video <id> [flags]

Examples:
  switchdl video 1234567890
  switchdl video 1234567890 9876543210 3134859203
  switchdl video 1234567890 -o /path/to/dir -f custom_name.mp4 -w -v

Flags:
  -f, --filename string   Output filename (defaults to video title)
  -h, --help              help for video

Global Flags:
  -o, --output-dir string   Output directory path (default ".")
  -w, --overwrite           Force overwrite of existing files
  -v, --select-variant      List all video variants (quality) and prompt for selection
  -s, --skip                Skip existing files
  -t, --token string        Access token for API authentication (overrides configured token)
```

### Download a channel

```bash
Download videos from one or more SwitchTube channels by providing their unique channel IDs.
You can either download all videos at once or select which ones specifically.

Usage:
  switchdl channel <id> [flags]

Examples:
 switchdl channel abcdef1234
 switchdl channel abcdef1234 ghijk56789 -a

Flags:
  -a, --all    Download all videos without prompting
  -h, --help   help for channel

Global Flags:
  -o, --output-dir string   Output directory path (default ".")
  -w, --overwrite           Force overwrite of existing files
  -v, --select-variant      List all video variants (quality) and prompt for selection
  -s, --skip                Skip existing files
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
