| **Release** | [![License](https://img.shields.io/github/license/marvinruder/hetzner-dyndns?label=License&style=flat-square)](/LICENSE) [![Latest Release (GitHub)](https://img.shields.io/github/v/release/marvinruder/hetzner-dyndns?label=Latest%20Release&logo=github&sort=semver&style=flat-square)](https://github.com/marvinruder/hetzner-dyndns/releases/latest) [![Latest Release (Docker)](https://img.shields.io/docker/v/marvinruder/hetzner-dyndns?label=Latest%20Release&logo=docker&sort=semver&style=flat-square)](https://hub.docker.com/r/marvinruder/hetzner-dyndns/tags) [![Docker Image Size](https://img.shields.io/docker/image-size/marvinruder/hetzner-dyndns?label=Docker%20Image%20Size&logo=docker&sort=semver&style=flat-square)](https://hub.docker.com/r/marvinruder/hetzner-dyndns/tags) [![Release Date](https://img.shields.io/github/release-date/marvinruder/hetzner-dyndns?label=Release%20Date&logo=github&style=flat-square)](https://github.com/marvinruder/hetzner-dyndns/releases/latest) [![Commits since latest release](https://img.shields.io/github/commits-since/marvinruder/hetzner-dyndns/latest?logo=github&sort=semver&style=flat-square)](https://github.com/marvinruder/hetzner-dyndns/commits) |
:-:|:-:
| **Quality** | [![GitHub checks](https://img.shields.io/github/checks-status/marvinruder/hetzner-dyndns/main?logo=github&label=Checks&style=flat-square)](https://github.com/marvinruder/rating-tracker/actions) [![Go Report Card](https://goreportcard.com/badge/github.com/marvinruder/hetzner-dyndns?style=flat-square)](https://goreportcard.com/report/github.com/marvinruder/hetzner-dyndns) [![Codacy grade](https://img.shields.io/codacy/grade/c149903c470e4d798cb712a16dc52f9b?label=Code%20Quality&logo=codacy&style=flat-square)](https://app.codacy.com/gh/marvinruder/hetzner-dyndns/dashboard) [![Codacy coverage](https://img.shields.io/codacy/coverage/c149903c470e4d798cb712a16dc52f9b?logo=codacy&label=Coverage&style=flat-square)](https://app.codacy.com/gh/marvinruder/hetzner-dyndns/coverage/dashboard) [![Jenkins build](https://jenkins.mruder.dev/buildStatus/icon?job=hetzner-dyndns-multibranch%2Fmain&subject=Build&style=flat-square)](https://jenkins.internal.mruder.dev/job/hetzner-dyndns-multibranch) <!-- ![Snyk Vulnerabilities](https://img.shields.io/snyk/vulnerabilities/github/marvinruder/hetzner-dyndns?label=Vulnerabilities&style=flat-square) --> |
| **Repository** | [![GitHub Contributors](https://img.shields.io/github/contributors/marvinruder/hetzner-dyndns?label=Contributors&logo=github&style=flat-square)](https://github.com/marvinruder/hetzner-dyndns/graphs/contributors) [![Commit Activity](https://img.shields.io/github/commit-activity/m/marvinruder/hetzner-dyndns?label=Commit%20Activity&logo=github&style=flat-square)](https://github.com/marvinruder/hetzner-dyndns/graphs/commit-activity) [![Last commit](https://img.shields.io/github/last-commit/marvinruder/hetzner-dyndns?label=Last%20Commit&logo=github&style=flat-square)](https://github.com/marvinruder/hetzner-dyndns/commits/main) [![Issues](https://img.shields.io/github/issues/marvinruder/hetzner-dyndns?label=Issues&logo=github&style=flat-square)](https://github.com/marvinruder/hetzner-dyndns/issues) [![Bugs](https://img.shields.io/github/issues/marvinruder/hetzner-dyndns/bug?label=Bug%20Issues&logo=openbugbounty&logoColor=red&style=flat-square)](https://github.com/marvinruder/hetzner-dyndns/issues?q=is%3Aopen+is%3Aissue+label%3Abug) [![Pull Requests](https://img.shields.io/github/issues-pr/marvinruder/hetzner-dyndns?label=Pull%20Requests&logo=github&style=flat-square)](https://github.com/marvinruder/hetzner-dyndns/pulls) |
| **Reference** | [![Go Reference](https://pkg.go.dev/badge/github.com/marvinruder/hetzner-dyndns.svg)](https://pkg.go.dev/github.com/marvinruder/hetzner-dyndns) |

---

# hetzner-dyndns

A proxy server for updating DNS records on Hetzner DNS using the DynDNS protocol.

## Usage

### Prerequisites

*   A DNS zone managed by Hetzner DNS
*   An API token for the Hetzner DNS API (obtainable from the [Hetzner DNS Console](https://dns.hetzner.com/settings/api-token))
*   A DynDNS client (e.g. a router) that supports the DynDNS protocol
*   A server with a static IP address that is reachable from the internet to run the proxy server on

### Set up the server

Start the server by running the provided binary from the [latest release](https://github.com/marvinruder/hetzner-dyndns/releases/latest) or using the Docker image `marvinruder/hetzner-dyndns`:

```bash
docker run -p 8245:8245 -e ZONE=example.com -e TOKEN=eWVzLCBpIGFtIGEgdG9rZW4= marvinruder/hetzner-dyndns:latest
```

A Docker Compose setup could look like this:

```
services:
  hetzner-dyndns:
    image: marvinruder/hetzner-dyndns:latest
    ports:
      - 8245:8245
    environment:
      ZONE: example.com
      TOKEN: eWVzLCBpIGFtIGEgdG9rZW4=
```

The server uses plain HTTP on port 8245. It is recommended to use a reverse proxy like nginx to add HTTPS support. A different port can also be configured using the reverse proxy or Docker.

The following environment variables are supported, none of which are required:

| Variable | Description |
| --- | --- |
| `COLOR` | Whether the log output should be colored. Set to `true` to enforce colored output, or `false` to enforce plain output. If not provided, the output will be colored if the output is a terminal. |
| `ZONE` | The DNS zone to use. The zone must still be provided as the username in every client request, but only those with the configured zone will be forwarded to Hetzner DNS API. If not provided, requests with any zone will be accepted. |
| `TOKEN` | The token to use for authentication against the Hetzner DNS API. The token must still be provided as the password in every client request, but only those with the configured token will be forwarded to Hetzner DNS API. If not provided, requests with any token will be accepted. |

### Configure a client

To update a DNS record, configure your DynDNS client (e.g. a router) to use

*   the DNS zone (e.g. `example.com`) as the username,
*   the token (e.g. `eWVzLCBpIGFtIGEgdG9rZW4=`) as the password,
*   the desired dynamic hostname (e.g. `home.example.com`) as the hostname,
*   the hostname or public IP address of the server (e.g. `dyndns.example.com`) as the update server address,
*   the port of the server (default: `8245`) as the update server port, and
*   `HTTP` as the update protocol (or `HTTPS` if you use a reverse proxy with HTTPS support, which is recommended).

Your client will take care of identifying changes in its public IP address and sending the appropriate requests to the server, keeping the DNS record up to date.

## Documentation

A detailed description of the DynDNS protocol is published by Oracle [here](https://help.dyn.com/remote-access-api/).

### Limitations of this implementation

*   Only the HTTP `GET` method is implemented.
*   Query parameters other than `hostname` and `myip` are not implemented.
*   Only one hostname can be updated per request.
*   It is not checked whether a request contains a valid User-Agent header.

## Contribute

Contributions are welcome!

## License

This software is provided under the conditions of the [MIT License](/LICENSE).

## Authors

-   [Marvin A. Ruder (he/him)](https://github.com/marvinruder)
