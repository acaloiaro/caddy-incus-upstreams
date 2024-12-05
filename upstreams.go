package caddy_incus_upstreams

import (
	"encoding/json"
	"net"
	"net/http"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp/reverseproxy"
	incus "github.com/lxc/incus/v6/client"
	"github.com/lxc/incus/v6/shared/api"
	"go.uber.org/zap"
)

const (
	UserConfigEnable       = "user.caddyserver.http.enable"
	UserConfigMatchHost    = "user.caddyserver.http.matchers.host"
	UserConfigUpstreamPort = "user.caddyserver.http.upstream.port"
)

var (
	candidates   []candidate
	candidatesMu sync.RWMutex
	producers    = map[string]func(string) (caddyhttp.RequestMatcher, error){
		UserConfigMatchHost: func(value string) (caddyhttp.RequestMatcher, error) {
			return &caddyhttp.MatchHost{value}, nil
		},
	}
)

type candidate struct {
	matchers caddyhttp.MatcherSet
	upstream *reverseproxy.Upstream
}

func init() {
	caddy.RegisterModule(Upstreams{})
}

type Upstreams struct {
}

func (Upstreams) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.reverse_proxy.upstreams.incus",
		New: func() caddy.Module { return new(Upstreams) },
	}
}

func (u *Upstreams) Provision(ctx caddy.Context) error {
	conn, err := incus.ConnectIncusUnix("", nil)
	if err != nil {
		return err
	}

	ctx.Logger().Info("connected to incus")

	return u.provision(ctx, conn)
}

func (u *Upstreams) GetUpstreams(r *http.Request) ([]*reverseproxy.Upstream, error) {
	upstreams := make([]*reverseproxy.Upstream, 0, 1)

	candidatesMu.RLock()
	defer candidatesMu.RUnlock()

	for _, c := range candidates {
		if c.matchers.Match(r) {
			upstreams = append(upstreams, c.upstream)
		}
	}

	return upstreams, nil
}

func (u *Upstreams) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		if d.NextArg() {
			return d.ArgErr()
		}
		if d.NextBlock(0) {
			return d.Errf("unrecognized incus option '%s'", d.Val())
		}
	}
	return nil
}

func (u *Upstreams) provision(ctx caddy.Context, conn incus.InstanceServer) error {
	err := u.provisionCandidates(ctx, conn)
	if err != nil {
		return err
	}

	go u.keepUpdated(ctx, conn)

	return nil
}

func (u *Upstreams) provisionCandidates(ctx caddy.Context, conn incus.InstanceServer) error {
	instances, err := conn.GetInstancesAllProjects(api.InstanceTypeContainer)
	if err != nil {
		return err
	}

	updated := make([]candidate, 0, len(instances))

	for _, i := range instances {
		matchers := buildMatchers(ctx, i.Config)

		ctx.Logger().Debug("matched instance", zap.String("instance_name", i.Name))

		enabled, ok := i.Config[UserConfigEnable]
		if !ok {
			ctx.Logger().Error("dynamic incus upstream not enabled",
				zap.String("instance_name", i.Name),
			)
			continue
		}

		if enabled != "true" {
			ctx.Logger().Error("dynamic incus upstream disabled",
				zap.String("instance_name", i.Name),
				zap.String("enabled", enabled),
			)
			continue
		}

		port, ok := i.Config[UserConfigUpstreamPort]
		if !ok {
			ctx.Logger().Error("unable to get port from instance config",
				zap.String("instance_name", i.Name),
			)
			continue
		}

		ctx.Logger().Debug("port retrieved",
			zap.Any("instance_name", i.Name),
			zap.String("port", port),
		)

		instanceFull, _, err := conn.GetInstanceFull(i.Name)
		if err != nil {
			ctx.Logger().Error("unable to get full instance info",
				zap.String("instance_name", i.Name),
			)
			continue
		}

		ipv4s := []string{}
		if instanceFull.IsActive() && instanceFull.State != nil && instanceFull.State.Network != nil {
			for _, net := range instanceFull.State.Network {
				if net.Type == "loopback" {
					continue
				}

				for _, addr := range net.Addresses {
					if slices.Contains([]string{"link", "local"}, addr.Scope) {
						continue
					}

					if addr.Family == "inet" {
						ipv4s = append(ipv4s, addr.Address)
					}
				}
			}
		}

		if len(ipv4s) == 0 {
			ctx.Logger().Error("unable to get ipv4",
				zap.String("instance_name", i.Name),
			)
			continue
		}

		sort.Sort(sort.Reverse(sort.StringSlice(ipv4s)))
		addr := ipv4s[0]

		ctx.Logger().Debug("ipv4 retrieved",
			zap.Any("instance_name", i.Name),
			zap.String("ipv4", addr),
		)

		address := net.JoinHostPort(addr, port)
		updated = append(updated, candidate{
			matchers: matchers,
			upstream: &reverseproxy.Upstream{Dial: address},
		})
	}

	candidatesMu.Lock()
	candidates = updated
	candidatesMu.Unlock()

	return nil
}

func (u *Upstreams) keepUpdated(ctx caddy.Context, conn incus.InstanceServer) {
	defer conn.Disconnect()

	listener, err := conn.GetEventsAllProjects()
	if err != nil {
		ctx.Logger().Warn("unable to monitor instance events, will retry", zap.Error(err))
		time.Sleep(500 * time.Millisecond)
		u.keepUpdated(ctx, conn)
	}

	ctx.Logger().Debug("initialised event listener")

	events := []string{
		api.EventLifecycleInstanceStarted,
		api.EventLifecycleInstanceRestarted,
	}

	if _, err := listener.AddHandler([]string{"lifecycle"}, func(event api.Event) {
		metadata := &api.EventLifecycle{}
		if err := json.Unmarshal(event.Metadata, &metadata); err != nil {
			ctx.Logger().Debug("unable to marshal event metadata", zap.Any("event", event))
			return
		}

		if !slices.Contains(events, metadata.Action) {
			return
		}

		ctx.Logger().Debug("handling event",
			zap.String("instance_name", metadata.Name),
			zap.String("event", metadata.Action),
		)

		if err := u.provisionCandidates(ctx, conn); err != nil {
			ctx.Logger().Error("unable to provision candidates", zap.Error(err))
		}
	}); err != nil {
		ctx.Logger().Error("event listener handler setup error", zap.Error(err))
	}

	chError := make(chan error, 1)
	chError <- listener.Wait()
	if chError != nil {
		ctx.Logger().Error("event listener wait error", zap.Error(err))
	}

	ctx.Logger().Debug("event listener saying goodbye 👋")
}

func buildMatchers(ctx caddy.Context, config map[string]string) caddyhttp.MatcherSet {
	var matchers caddyhttp.MatcherSet

	for key, producer := range producers {
		value, ok := config[key]
		if !ok {
			continue
		}

		matcher, err := producer(value)
		if err != nil {
			ctx.Logger().Error("unable to load matcher",
				zap.String("key", key),
				zap.String("value", value),
				zap.Error(err),
			)
			continue
		}

		if prov, ok := matcher.(caddy.Provisioner); ok {
			err = prov.Provision(ctx)
			if err != nil {
				ctx.Logger().Error("unable to provision matcher",
					zap.String("key", key),
					zap.String("value", value),
					zap.Error(err),
				)
				continue
			}
		}

		matchers = append(matchers, matcher)
	}

	return matchers
}

var (
	_ caddy.Provisioner           = (*Upstreams)(nil)
	_ reverseproxy.UpstreamSource = (*Upstreams)(nil)
	_ caddyfile.Unmarshaler       = (*Upstreams)(nil)
)
