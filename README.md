# caddy-incus-upstreams

> Status: **HIGHLY experimental**, patches welcome 🚩

[`Incus`](https://linuxcontainers.org/incus/) dynamic upstreams for
[`Caddy`](https://caddyserver.com/docs/) v2+ 🧨

In other words, `Caddy` can automatically pick up your `Incus` instances when
they have 3 config keys attached to them which specify 1. that they want to be
routed 2. which domain should be routed to them 3. which port they'll answer
on. Combined with the lightweight configuration and the Auto-TLS (magic) powers
of Caddy, provisioning `Incus` instances to serve on the web is much more
convenient.

## Usage

Set the following config on your `Incus` instance.

```bash
incus launch images:alpine/3.20 <instance-name>
incus config set <instance-name> user.caddyserver.http.enable=true
incus config set <instance-name> user.caddyserver.http.matchers.host=<domain>
incus config set <instance-name> user.caddyserver.http.upstream.port=<port>
```

Build a fresh `Caddy` with this plugin.

```bash
xcaddy build \
  --with=git.coopcloud.tech/decentral1se/caddy-incus-upstreams \
  --replace=go.opentelemetry.io/otel/sdk=go.opentelemetry.io/otel/sdk@v1.25.0
```

Wire up a `Caddyfile` based on this example.

```Caddyfile
<domain> {
  reverse_proxy {
    dynamic incus
  }
}
```

And then make sure everything gets picked up with a `reload`/`restart`.

```
caddy reload
incus restart <instance-name>
```

## Notes

The plugin responds to the following `Incus` events:

* `api.EventLifecycleInstanceCreated`
* `api.EventLifecycleInstanceRestarted`
* `api.EventLifecycleInstanceResumed`
* `api.EventLifecycleInstanceStarted`

There is a rather crude implementation for handling these events. We simply
wire up a few seconds of sleep to allow for the network part of the instance to
come up. Otherwise, there is no network address to retrieve.

We currently *only* match against the upstream ipv4 addresses of instances.

The system user that runs `Caddy` must be `root` or be in the `incus-admin`
group so that it can make queries across projects for different instances.

## FAQ

### Does this support wildcard certificates?

Yes! You'll need to enable a [DNS
plugin](https://caddy.community/t/how-to-use-dns-provider-modules-in-caddy-2/8148j)
and wire up something like this in your `Caddyfile`.

```Caddyfile
{
  acme_dns <your-provider-here> <your-token-here>
}

*.example.org {
  reverse_proxy {
    dynamic incus
  }
}
```

## Hackin'

Install [`xcaddy`](https://github.com/caddyserver/xcaddy) and
[`Incus`](https://linuxcontainers.org/incus/).

Create this `Caddyfile` in the root of the project repository.

```Caddyfile
{
  debug
  http_port 6565
}

http://foo.localhost {
  reverse_proxy {
    dynamic incus
  }
}
```

Then create a new instance and assign the relevant config.

```bash
incus launch images:alpine/3.20 foo
incus config set foo user.caddyserver.http.enable=true
incus config set foo user.caddyserver.http.matchers.host=foo.localhost
incus config set foo user.caddyserver.http.upstream.port=80
```

Serve something from your instance.

```
incus shell foo
apk add python3
python3 -m http.server 80
```

Run `Caddy` with the plugin baked in.

```
xcaddy run
```

And finally, route a request to the instance via `Caddy`.

```
curl -X GET http://foo.localhost:6565
```

🧨

## ACK

* [`caddy-docker-upstreams`](https://github.com/invzhi/caddy-docker-upstreams)

## License

<a href="https://git.coopcloud.tech/decentral1se/caddy-incus-upstreams/src/branch/main/LICENSE">
  <img src="https://www.gnu.org/graphics/gplv3-or-later.png" />
</a>
