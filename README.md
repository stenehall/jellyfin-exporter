
This is heavily inspired by https://github.com/frebib/nzbget-exporter

## Getting Started

Generate an api key for Jellyfin via the admin interface, "Administration" -> "Dashboard" -> "Advanced" -> "API Keys"

### Build from source
```sh
git clone https://github.com/stenehall/jellyfin-exporter.git .
go build -o jellyfin_exporter
./jellyfin_exporter --host=http://jellyfin --apikey=<insert api key>
```

### Run with Docker
```sh
docker run -t \
  -e HOST=http://jellyfin \
  -e API_KEY=<insert api key> \
  -p 9452:9452 \
  stenehall/jellyfin-exporter
```

### Run with docker-compose
```yaml
services:
  jellyfin_exporter:
    image: stenehall/jellyfin-exporter
    environment:
    - HOST=http://jellyfin
    - API_KEY=<insert api key>
    ports:
    - 9453:9453
```

## Configuration Options

Configuration should be passed via command-line arguments, or the environment. Every option is described in the `--help` output, as below:

```
Jellyfin Exporter (version 0.1.0)

Usage:
  jellyfin_exporter [OPTIONS]

Options:
      --log-level= log verbosity level (trace, debug, info, warn, error, fatal) (default: info) [$LOG_LEVEL]
      --namespace= metric name prefix (default: jellyfin) [$METRIC_NAMESPACE]
  -l, --listen=    host:port to listen on (default: :9453) [$LISTEN]
  -h, --host=      jellyfin host to export metrics for [$HOST]
  -u, --apikey=    jellyfin apikey for auth [$API_KEY]

Help Options:
  -h, --help       Show this help message

```
