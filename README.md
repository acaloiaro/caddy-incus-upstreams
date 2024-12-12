# caddy-incus-upstreams

> Status: **HIGHLY experimental**, patches welcome 🚩

Incus dynamic upstreams for Caddy v2+ 🧨

## Usage

Set the following config on your Incus instance.

```bash
incus config set <instance-name> user.caddyserver.http.enable=true
incus config set <instance-name> user.caddyserver.http.matchers.host=<domain>
incus config set <instance-name> user.caddyserver.http.upstream.port=<port>
```

Build a fresh caddy with this plugin.

```bash
xcaddy build \
  --with git.coopcloud.tech/decentral1se/caddy-incus-upstreams
```

Wire up a Caddyfile based on this example.

```Caddyfile
example.com {
  reverse_proxy {
    dynamic incus
  }
}
```

## Notes

The plugin responds to the following Incus events:

* `api.EventLifecycleInstanceStarted`
* `api.EventLifecycleInstanceRestarted`

It currently *only* retrieves the ipv4 addresses of the instances.

## Hackin'

Install [`xcaddy`](https://github.com/caddyserver/xcaddy) and [Incus](https://linuxcontainers.org/incus/).

Create this Caddyfile in the root of the project repository.

```Caddyfile
{
  debug
  http_port 6565
}

http://foo.localhost,
http://bar.localhost {
  reverse_proxy {
    dynamic incus
  }
}
```

Then run commands based on this example.

```bash
incus launch images:alpine/3.20 foo
incus config set foo user.caddyserver.http.enable=true
incus config set foo user.caddyserver.http.matchers.host=foo.localhost
incus config set foo user.caddyserver.http.upstream.port=80

incus launch images:alpine/3.20 bar
incus config set bar user.caddyserver.http.enable=true
incus config set bar user.caddyserver.http.matchers.host=bar.localhost
incus config set bar user.caddyserver.http.upstream.port=80

# wire up a simple web server on your 2 instances
# $ incus shell foo / bar
# $ apk add python3
# $ python3 -m http.server 80

xcaddy run

# fire a request via caddy to your instances
# curl -X GET http://foo.localhost:6565
# curl -X GET http://bar.localhost:6565
```

🧨

## ACK

* [`caddy-docker-upstreams`](https://github.com/invzhi/caddy-docker-upstreams)

## License

<a href="https://git.coopcloud.tech/decentral1se/caddy-incus-upstreams/src/branch/main/LICENSE">
  <img src="https://www.gnu.org/graphics/gplv3-or-later.png" />
</a>
