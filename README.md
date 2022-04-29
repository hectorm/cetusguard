# CetusGuard

CetusGuard is a tool that allows to protect the Docker daemon socket by filtering the calls to its API endpoints.

Some highlights:
 * It is written in a memory safe language.
 * Has a small codebase that can be easily audited.
 * Has zero dependencies to mitigate supply chain attacks.

## Docker daemon security

Unless you opt in to [rootless mode][1] (which has [some limitations][2]), the daemon requires root and any service that has access to its API can escalate privileges.

The daemon by default [exposes its API][3] through a non-networked Unix socket that can be restricted by file system permissions and for networked use the daemon supports being exposed over SSH or TCP with TLS client authentication. However, you still have to fully trust any service you give access to its API.

CetusGuard solves this problem by acting as a proxy between the daemon and the services that consume its API, allowing for example read-only access to some endpoints.

## Usage

CetusGuard is distributed as a Docker image available on [Docker Hub][4] and as a statically linked binary available in the [releases section][5] of the project.

A collection of examples for experimenting with CetusGuard, including some real world scenarios with Traefik and Netdata, can be found in the [./examples/](./examples/) directory.

These are the supported options:
```
  -backend-addr string
        Container daemon socket to connect to (env CETUSGUARD_BACKEND_ADDR, CONTAINER_HOST, DOCKER_HOST) (default "unix:///var/run/docker.sock")
  -backend-tls-cacert string
        Path to the backend TLS certificate used to verify the daemon identity (env CETUSGUARD_BACKEND_TLS_CACERT)
  -backend-tls-cert string
        Path to the backend TLS certificate used to authenticate with the daemon (env CETUSGUARD_BACKEND_TLS_CERT)
  -backend-tls-key string
        Path to the backend TLS key used to authenticate with the daemon (env CETUSGUARD_BACKEND_TLS_KEY)
  -frontend-addr string
        Address to bind the server to (env CETUSGUARD_FRONTEND_ADDR) (default "tcp://:2375")
  -frontend-tls-cacert string
        Path to the frontend TLS certificate used to verify the identity of clients (env CETUSGUARD_FRONTEND_TLS_CACERT)
  -frontend-tls-cert string
        Path to the frontend TLS certificate (env CETUSGUARD_FRONTEND_TLS_CERT)
  -frontend-tls-key string
        Path to the frontend TLS key (env CETUSGUARD_FRONTEND_TLS_KEY)
  -log-level int
        The minimum entry level to log, from 0 to 7 (env CETUSGUARD_LOG_LEVEL) (default 6)
  -no-default-rules
        Do not load any default rules (env CETUSGUARD_NO_DEFAULT_RULES)
  -rules value
        Filter rules separated by new lines, can be specified multiple times (env CETUSGUARD_RULES) (default [])
  -rules-file value
        Filter rules file, can be specified multiple times (env CETUSGUARD_RULES_FILE) (default [])
  -version
        Show version number and quit
```

## Filter rules

By default, only a few common harmless endpoints are allowed, `/_ping`, `/info` and `/version`.

All other endpoints are denied and must be explicitly allowed through a rule syntax defined by the following ABNF grammar:
```
blank   = ( SP / HTAB )
method  = 1*%x41-5A                             ; HTTP method
methods = method *( "," method )                ; HTTP method list
pattern = 1*UNICODE                             ; Target path regex
rule    = *blank methods 1*blank pattern *blank ; Rule
```

Only requests that match the specified HTTP methods and target path regex will be allowed.

There are some built-in variables specified by surrounding `%` that can be used to compose rule patterns, the full list and their values can be found in the [`rule.go`](./cetusguard/rule.go) file.

Lines beginning with `!` are ignored.

Some example rules are:
```
! Ping
GET,HEAD %API_PREFIX_PING%

! Get version
GET %API_PREFIX_VERSION%

! Get system information
GET %API_PREFIX_INFO%

! Get data usage information
GET %API_PREFIX_SYSTEM%/df

! Monitor events
GET %API_PREFIX_EVENTS%

! List containers
GET %API_PREFIX_CONTAINERS%/json

! Inspect a container 
GET %API_PREFIX_CONTAINERS%/%CONTAINER_ID_OR_NAME%/json
```

## License

[MIT License](./LICENSE.md) © [Héctor Molinero Fernández](https://hector.molinero.dev).

[1]: https://docs.docker.com/engine/security/rootless/
[2]: https://docs.docker.com/engine/security/rootless/#known-limitations
[3]: https://docs.docker.com/engine/security/protect-access/
[4]: https://hub.docker.com/r/hectorm/cetusguard
[5]: https://github.com/hectorm/cetusguard/releases
