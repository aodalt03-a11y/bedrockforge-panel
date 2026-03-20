package mods

import (
    "context"
    "net"
)

func init() {
    net.DefaultResolver = &net.Resolver{
        PreferGo: true,
        Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
            d := net.Dialer{}
            return d.DialContext(ctx, "udp", "8.8.8.8:53")
        },
    }
}
