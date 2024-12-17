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

* `api.EventLifecycleInstanceCreated`
* `api.EventLifecycleInstanceRestarted`
* `api.EventLifecycleInstanceResumed`
* `api.EventLifecycleInstanceStarted`

It currently *only* matches against the upstream ipv4 addresses of instances.

## FAQ

### Does this support wildcard certificates?

Yes! You'll need to enable a [DNS plugin](https://caddy.community/t/how-to-use-dns-provider-modules-in-caddy-2/8148j) and wire up something like this in a Caddyfile.

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

Install [`xcaddy`](https://github.com/caddyserver/xcaddy) and [Incus](https://linuxcontainers.org/incus/).

Create this Caddyfile in the root of the project repository.

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

Run Caddy with the plugin baked in.

```
xcaddy run
```

And finally, route a request to the instance via Caddy.

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
