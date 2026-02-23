<p align="center">
  <img src="assets/vecna.jpg" alt="Vecna" width="120" />
</p>

<p align="center">Minimalistic SSH TUI.</p>

## Install

```bash
curl -sSL https://raw.githubusercontent.com/shravan20/vecna/master/scripts/install.sh | sh
```

- **Go:** `go install github.com/shravan20/vecna@latest`
- **From source:**

```bash
make build
./bin/vecna
```

## Usage

```bash
vecna          # Launch TUI
vecna version  # Print version
```

## Config

**Path:** `~/.config/vecna/config.yaml`

```yaml
hosts:
  - name: prod
    hostname: 192.168.1.100
    user: admin
    port: 22
    identity_file: ~/.ssh/id_rsa
    tags: [production]
    proxy_jump: bastion   # optional: name of another host to use as jump/bastion

commands:                  # optional: saved commands for "Run command" (r)
  - label: "disk usage"
    command: "df -h"
  - label: "memory"
    command: "free -m"
```

---

## Contributing

Issues and PRs welcome. Tag releases with semver (`v1.0.0`); CI builds and publishes.

## License

MIT
