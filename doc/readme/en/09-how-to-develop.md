# How to Develop

[Back to English README](../../../README.EN.md) · [Installation and Initialization](01-installation-and-initialization.md) · [中文](../zh-CN/09-如何开发.md)

Instructions for **Go development, Bee, dependency installation, startup, compilation, testing, packaging, and Web UI development**, with commands updated to match the scripts in the current repository. This document is intended for a development environment. For initial database setup and account configuration, read [01 · Installation and Initialization](01-installation-and-initialization.md) first.

## 1. Code Structure

The project consists of two separate directories, which are built independently:

| Directory | Purpose |
| --- | --- |
| `go_binance_futures/` | Go backend, trading logic, Web APIs, database synchronization, CLI, and built frontend static assets |
| `go_binance_futrues_new_ui/` | Vue/Vite management UI source code (directory spelling matches the actual project) |
| Backend `controllers/`, `routers/`, `middlewares/` | API controllers, routes, and login authentication |
| Backend `feature/`, `spot/`, `service/` | Futures/Spot trading and business services |
| Backend `models/` | Database models |
| Backend `conf/` | Configuration example and actual local configuration |
| Backend `scripts/test-go.sh` | Go test entry point for application packages |
| Backend `static/` | Production static assets synchronized after Vue `pnpm build` |

The commands below assume both project directories share the same parent directory, and that commands are run from the root of the corresponding repository.

## 2. Go Backend Development

### 2.1 Environment and Dependencies

The current backend `go.mod` specifies **Go 1.25.0**. Install the matching Go toolchain and check the version:

```bash
go version
go env GOPATH
```

From the backend project root, download dependencies:

```bash
go mod download
```

You can use `go mod tidy` to manage dependencies. Note that it may also modify `go.mod` / `go.sum`; therefore, use it only when intentionally maintaining dependencies rather than running it every time you develop.

### 2.2 Development Configuration and Database

Copy the example configuration:

```bash
cp conf/app.conf.example conf/app.conf
```

Edit `conf/app.conf` to use **a local development database and test-environment credentials**. Do not commit real secrets to Git. This is a sensitive local configuration file; do not expose it in logs, screenshots, or code reviews.

For initial setup or schema migrations, build the application first and then run the project's designated synchronization command:

```bash
go build -o go_binance_futures .

./go_binance_futures sync db
```

- **Use only `./go_binance_futures sync db` for database schema synchronization**; do not rely on normal startup to migrate the database.
- Verify the target database and current schema version before migration. Development and testing must not accidentally connect to or write to the production database.
- Ordinary Go business-code changes do not require repeated database synchronization. Run it according to the project workflow only when there are corresponding schema changes.

### 2.3 Running and Debugging

After verifying your development database and configuration:

```bash
go run .
```

Bee is also available as an optional development workflow:

```bash
go install github.com/beego/bee/v2@latest
bee run
```

Bee is not required to run this project. If the `bee` command cannot be found, check `go env GOPATH` and add `$(go env GOPATH)/bin` to `PATH`. Updating Bee may change its behavior, so prefer a version already verified with the project.

The local Web UI is typically available at:

```text
http://localhost:3333/zmkm/index.html
```

The actual port and URL prefix are determined by `web.port` and `web.index` in `conf/app.conf`.

### 2.4 Compilation and Testing

Build the current application:

```bash
go build -o go_binance_futures .
```

Run the project's test script (which automatically excludes historical research archives that are not buildable application packages):

```bash
./scripts/test-go.sh
./scripts/test-go.sh -count=5 -p=2 -timeout=120s
```

You may also test specific business packages:

```bash
go test ./controllers ./middlewares -count=1
go test ./service/... -count=1 -p=2
```

Parameter meanings: `-count=5` repeats tests five times; `-p=2` controls the number of package build/test operations the Go tool runs concurrently; `-timeout=120s` sets the timeout for an individual test binary. For tests involving trading, databases, or networks, verify that a safe test environment is in use.

### 2.5 Packaging

The old README provided this Windows packaging command:

```bash
bee pack -be GOOS=windows
```

This is a **historical command reference**; successful cross-compilation has not been verified with the current Go/CGO/SQLite dependencies. Refer to actual CI and Release verification results for Windows/Linux release packages. Do not assume a binary compiled locally on macOS will run on an ARM64 server.

## 3. Web UI Development (Vue)

The management UI is a separate frontend project. Its current `.nvmrc` specifies Node **22.14.0**, while `package.json` requires `pnpm >= 9` and uses Vite.

### 3.1 Installation and Startup

```bash
cd ../go_binance_futrues_new_ui
corepack enable
pnpm install --frozen-lockfile
pnpm dev
```

Use the address printed by the development server. Frontend API URLs and proxy settings are determined by the Vite configuration and environment configuration in that project; do not directly modify the production server's `conf/app.conf`.

### 3.2 Checks and Build

```bash
pnpm typecheck
pnpm exec eslint src
pnpm build
```

The production build output is in the frontend repository's `dist/` directory. After confirming that the backend target `static/` contains only the corresponding built assets, synchronize them using the current project workflow:

```bash
rsync -a --delete dist/ ../go_binance_futures/static/
```

**Note:** `--delete` removes files in the target `static/` that are absent from the new `dist/`. Use this command only after verifying the paths. Always rebuild the frontend and synchronize the assets after frontend changes; otherwise, the Go server may continue serving the old UI.
