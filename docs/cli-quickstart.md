# CLI quick start

Initialize configuration and a profile, then authenticate with either cookies or credentials.

```sh
meta config init
meta profile create default
meta profile use default
```

Import cookies from a file:

```sh
meta auth import --input cookies.txt
```

or authenticate interactively:

```sh
meta auth login
```

Inspect health/connectivity:

```sh
meta doctor --online
```

Global machine-readable options include:

- `--json` for JSON output;
- `--jqi` to filter JSON input before strict decoding;
- `--jqo` to filter JSON output and imply JSON mode.

Use `meta <command> --help` for the current command surface. Avoid placing passwords, cookies or TOTP seeds directly in shell history when an interactive or file/stdin input path is available.
