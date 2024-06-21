package bot

import (
	"EverythingSuckz/fsb/config"
	"EverythingSuckz/fsb/internal/commands"
	"context"
	"github.com/celestix/gotgproto/sessionMaker"
	"github.com/glebarez/sqlite"
	"github.com/gotd/td/telegram/dcs"
	"golang.org/x/net/proxy"
	"time"

	"go.uber.org/zap"

	"github.com/celestix/gotgproto"
)

var Bot *gotgproto.Client

func StartClient(log *zap.Logger) (*gotgproto.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	resultChan := make(chan struct {
		client *gotgproto.Client
		err    error
	})

	sock5, _ := proxy.SOCKS5("tcp", config.ValueOf.PROXY, &proxy.Auth{}, proxy.Direct)
	dc := sock5.(proxy.ContextDialer)

	go func(ctx context.Context) {
		client, err := gotgproto.NewClient(
			int(config.ValueOf.ApiID),
			config.ValueOf.ApiHash,
			gotgproto.ClientTypeBot(config.ValueOf.BotToken),
			&gotgproto.ClientOpts{
				Session: sessionMaker.SqlSession(sqlite.Open("fsb")),
				Resolver: dcs.Plain(dcs.PlainOptions{
					Dial: dc.DialContext,
				}),
				DisableCopyright: true,
			},
		)
		resultChan <- struct {
			client *gotgproto.Client
			err    error
		}{client, err}
	}(ctx)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-resultChan:
		if result.err != nil {
			return nil, result.err
		}
		commands.Load(log, result.client.Dispatcher)
		log.Info("Client started", zap.String("username", result.client.Self.Username))
		Bot = result.client
		return result.client, nil
	}
}
