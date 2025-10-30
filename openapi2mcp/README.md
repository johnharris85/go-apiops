# openapi2mcp

Convert OpenAPI specifications to Kong Gateway `ai-mcp-proxy` plugin configurations.

## Overview

The `openapi2mcp` tool reads an OpenAPI 3.0 specification and generates Kong declarative configuration for the `ai-mcp-proxy` plugin. It converts API operations into MCP (Model Context Protocol) tool definitions that can be used by AI models to interact with your API.

## Building

From the root of the `go-apiops` project:

```bash
go build -o go-apiops .
```

This creates the `go-apiops` binary which includes the `openapi2mcp` command.

## Quick Start

```bash
./go-apiops openapi2mcp -s api-spec.yaml -o mcp-config.yaml
```

**Note:** The tool accepts both YAML and JSON input formats for OpenAPI specs. The format is auto-detected.

## Required: Tagging Operations

Operations in your OpenAPI spec **must** have tags starting with `mcp:` to be included in the output. The value after `mcp:` becomes the server tag for that plugin.

```yaml
openapi: 3.0.0
paths:
  /users:
    get:
      summary: Get users
      tags:
        - mcp:users     # Creates plugin with server.tag = "users"
      responses:
        '200':
          description: Success

  /health:
    get:
      summary: Health check
      tags:
        - mcp:monitoring  # Creates plugin with server.tag = "monitoring"
      responses:
        '200':
          description: Success
```

**Multiple tags:** Operations can have multiple `mcp:` tags - the operation will appear in multiple plugin configurations:

```yaml
  /flights:
    get:
      tags:
        - mcp:flights
        - mcp:monitoring  # This operation will be in BOTH plugins
```

## Usage

### Basic Command

```bash
./go-apiops openapi2mcp [options]
```

### CLI Options

| Option | Short | Default | Description |
|--------|-------|---------|-------------|
| `--spec` | `-s` | `-` (stdin) | OpenAPI spec file to process |
| `--output-file` | `-o` | `-` (stdout) | Output file to write |
| `--format` | | `yaml` | Output format: `json` or `yaml` |
| `--route-name` | | | Kong route name/ID to associate plugin with |
| `--service-name` | | | Kong service name/ID to associate plugin with |
| `--path-prefix` | | | Path prefix to prepend to all tool paths |
| `--mode` | | `conversion-listener` | MCP proxy mode |
| `--server-timeout` | | `60000` | Timeout in milliseconds |
| `--log-statistics` | | `false` | Enable logging of MCP metrics |
| `--log-payloads` | | `false` | Enable logging of request/response bodies |
| `--forward-client-headers` | | `true` | Forward client headers to upstream |
| `--ignore-circular-refs` | | `false` | Ignore circular references in the spec |

**Note:** `--route-name` and `--service-name` are mutually exclusive.

## Examples

### Basic Conversion

Convert an OpenAPI spec to MCP plugin configuration:

```bash
./go-apiops openapi2mcp -s api.yaml -o mcp-config.yaml
```

### With JSON Input

The tool accepts JSON format OpenAPI specs:

```bash
./go-apiops openapi2mcp -s api.json -o mcp-config.yaml
```

### With Route Association

Associate the plugin with a specific Kong route:

```bash
./go-apiops openapi2mcp -s api.yaml -o mcp-config.yaml --route-name my-api-route
```

### With Service Association

Associate the plugin with a specific Kong service:

```bash
./go-apiops openapi2mcp -s api.yaml --service-name my-api-service
```

### With Path Prefix

Prepend a path prefix to all tool paths (useful for versioned APIs):

```bash
./go-apiops openapi2mcp -s api.yaml --path-prefix /api/v1
```

This converts a path like `/users` to `/api/v1/users` in the output. The prefix is normalized:
- Leading `/` is added if missing
- Trailing `/` is removed if present

### JSON Output

```bash
./go-apiops openapi2mcp -s api.yaml -o mcp-config.json --format json
```

### With Logging Enabled

```bash
./go-apiops openapi2mcp -s api.yaml -o mcp-config.yaml \
  --log-statistics \
  --log-payloads
```

### From stdin to stdout

```bash
cat api.yaml | ./go-apiops openapi2mcp | kubectl apply -f -
```

## Output Format

The tool generates Kong declarative configuration in the following format:

```yaml
_format_version: "3.0"
plugins:
- name: ai-mcp-proxy
  route: my-route  # Optional, if --route-name specified
  config:
    mode: conversion-listener
    server:
      tag: users  # From mcp:users tag
      timeout: 60000
      forward_client_headers: true
    tools:
    - method: GET
      path: /users/{id}
      description: Get user by ID
      parameters:
      - name: id
        in: path
        required: true
        schema:
          type: string
      annotations:
        title: getUserById
        read_only_hint: true
    - method: POST
      path: /users
      description: Create a new user
      request_body:
        content:
          application/json:
            schema:
              type: object
              properties:
                name:
                  type: string
                email:
                  type: string
              required:
              - name
              - email
      annotations:
        title: createUser
        open_world_hint: true
- name: ai-mcp-proxy
  config:
    mode: conversion-listener
    server:
      tag: monitoring  # From mcp:monitoring tag
      timeout: 60000
      forward_client_headers: true
    tools:
    - method: GET
      path: /health
      description: Health check
      annotations:
        title: healthCheck
        read_only_hint: true
```

## What Gets Converted

### Operations

- **Method & Path:** HTTP method and URL path
- **Description:** From operation summary or description
- **Parameters:** Path, query, and header parameters with schemas
- **Request Body:** Full schema for request bodies (POST, PUT, PATCH)
- **Annotations:** Auto-generated hints based on HTTP method

### Annotations

Annotations are automatically added based on the HTTP method:

- **GET, HEAD, OPTIONS:** `read_only_hint: true`
- **PUT, PATCH:** `idempotent_hint: true`
- **DELETE:** `destructive_hint: true`
- **POST:** `open_world_hint: true`

The `title` annotation is set from the operation's `operationId`.

## Operations Excluded

Operations are **excluded** if they:
- Don't have any tag starting with `mcp:`
- Are missing from the spec

## MCP Proxy Modes

The `--mode` flag controls how the proxy behaves:

- `conversion-listener` (default): Converts OpenAPI requests to MCP format and listens for responses
- `conversion-only`: Only converts requests, doesn't wait for responses
- `listener`: Only listens for MCP responses
- `passthrough-listener`: Passes requests through while listening

## Use as Go Library

You can also use `openapi2mcp` as a Go library:

```go
package main

import (
    "github.com/kong/go-apiops/openapi2mcp"
    "os"
)

func main() {
    spec, _ := os.ReadFile("api.yaml")

    opts := openapi2mcp.O2MCPOptions{
        RouteName:     "my-route",
        PathPrefix:    "/api/v1",
        Mode:          "conversion-listener",
        ServerTimeout: 60000,
    }

    result, err := openapi2mcp.Convert(spec, opts)
    if err != nil {
        panic(err)
    }

    // result is map[string]interface{} with _format_version and plugins
}
```

## Testing

Run the test suite:

```bash
go test ./openapi2mcp
```

Run with verbose output:

```bash
go test -v ./openapi2mcp
```

## Examples

See the `flights-oas-with-mcp-tags.yaml` file in the project root for a complete example OpenAPI specification with proper `mcp:` tags.

Generate example output:

```bash
./go-apiops openapi2mcp -s flights-oas-with-mcp-tags.yaml -o example-output.yaml
```

## Troubleshooting

### No plugins in output

**Cause:** No operations have `mcp:` tags.

**Solution:** Add tags starting with `mcp:` to your operations:

```yaml
paths:
  /users:
    get:
      tags:
        - mcp:users  # Add this!
```

### Empty tools array

**Cause:** Operations have tags but none start with `mcp:`.

**Solution:** Ensure tags have the `mcp:` prefix:

```yaml
tags:
  - users       # ❌ Won't work
  - mcp:users   # ✅ Correct
```

### Circular reference errors

**Cause:** Your OpenAPI spec has circular schema references.

**Solution:** Use the `--ignore-circular-refs` flag:

```bash
./go-apiops openapi2mcp -s api.yaml --ignore-circular-refs
```

## Related

- [Kong Gateway Documentation](https://docs.konghq.com/)
- [ai-mcp-proxy Plugin](https://docs.konghq.com/hub/kong-inc/ai-mcp-proxy/)
- [Model Context Protocol (MCP)](https://modelcontextprotocol.io/)
