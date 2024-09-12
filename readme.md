# Headscale Discovery Service

This package provides a simple HTTP server that interacts with the Headscale API to list machines and expose them as
discovery targets for service monitoring.

## Overview

The service queries the Headscale API for the list of machines and exposes them as JSON discovery targets on a specified
endpoint. It filters the machine tags to extract those that contain `scrape_` prefixes, interprets them as monitoring
targets, and formats them in a way that can be consumed by monitoring systems like Prometheus.

## Environment Variables

The following environment variables are required to configure the service:

- `HEADSCALE_API_URL`: The base URL for the Headscale API.
- `HEADSCALE_API_KEY`: The API key for authenticating with the Headscale API.
- `LISTEN_ADDR`: (Optional) The address and port on which the HTTP server should listen (default is `:8080`).
## Installation
1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd <repository-folder>
    ```
2. Build the application:
    ```bash
    go build -o headscale_sd
    ```

## Usage

Run the service with the required environment variables:

```bash
export HEADSCALE_API_URL="http://example.com"
export HEADSCALE_API_KEY="your-api-key"
export LISTEN_ADDR=":8080"  # Optional
./headscale_sd
```

## Endpoints

The service exposes the following endpoint:

- `/`: Returns a JSON array of discovery targets that can be used for service monitoring.

## Example Response

A successful response from the service looks like this:

```json
[
  {
    "targets": ["192.168.1.10:9090"],
    "labels": {
      "node_name": "node1",
      "app": "app1"
    }
  },
  {
    "targets": ["192.168.1.11:8080"],
    "labels": {
      "node_name": "node2",
      "app": "app2"
    }
  }
]
