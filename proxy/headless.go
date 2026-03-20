package proxy

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"golang.org/x/oauth2"

	"bedrockforge/config"
	"bedrockforge/mods"
)

type botSession struct {
	mu          sync.Mutex
	server      *minecraft.Conn
	client      *minecraft.Conn
	clientReady chan struct{}
	cachedPkts  []packet.Packet
}

var (
	activeBotMu sync.Mutex
	activeBot   *botSession
)

func TakeoverBot(client *minecraft.Conn) bool {
	activeBotMu.Lock()
	bot := activeBot
	activeBotMu.Unlock()
	if bot == nil {
		return false
	}
	bot.mu.Lock()
	bot.client = client
	bot.mu.Unlock()
	close(bot.clientReady)
	return true
}

func StartHeadlessAndListen(cfg *config.Config) {
	src, err := getTokenSource()
	if err != nil {
		log.Fatalf("[Bot] auth failed: %v", err)
	}

	rPacks := mods.LoadResourcePacks()
	listener, err := minecraft.ListenConfig{
		AuthenticationDisabled: true,
		StatusProvider:         minecraft.NewStatusProvider("BedrockForge", "AFK Bot"),
		ResourcePacks:          rPacks,
	}.Listen("raknet", cfg.ListenAddr)
	if err != nil {
		log.Fatalf("[Bot] listen failed: %v", err)
	}
	defer listener.Close()
	log.Printf("[Bot] listening for client on %s", cfg.ListenAddr)
	fmt.Printf("[Bot] listening for client on %s\n", cfg.ListenAddr)

	// Accept real clients in background
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				continue
			}
			client := conn.(*minecraft.Conn)
			if !TakeoverBot(client) {
				go runSession(client, cfg, src)
			}
		}
	}()

	// Bot loop — reconnects automatically
	for {
		runBot(cfg, src)
		delay := cfg.ReconnectDelay
		if delay <= 0 {
			delay = 5
		}
		log.Printf("[Bot] reconnecting in %ds...", delay)
		time.Sleep(time.Duration(delay) * time.Second)
	}
}

func runBot(cfg *config.Config, src oauth2.TokenSource) {
	log.Println("[Bot] connecting to server...")
	fmt.Println("[Bot] connecting to server...")

	dialer := minecraft.Dialer{
		ErrorLog:    logger,
		TokenSource: src,
	}
	server, err := dialer.Dial("raknet", cfg.ServerAddr)
	if err != nil {
		log.Printf("[Bot] dial failed: %v", err)
		return
	}
	defer server.Close()

	bot := &botSession{
		clientReady: make(chan struct{}),
	}
	activeBotMu.Lock()
	activeBot = bot
	activeBotMu.Unlock()
	defer func() {
		activeBotMu.Lock()
		activeBot = nil
		activeBotMu.Unlock()
	}()

	log.Println("[Bot] connected to server")
	fmt.Println("[BOT_CONNECTED]")

	allMods := mods.All()
	sendToServer := func(pk packet.Packet) error { return server.WritePacket(pk) }
	sendToClient := func(pk packet.Packet) error {
		bot.mu.Lock()
		c := bot.client
		bot.mu.Unlock()
		if c == nil {
			return nil
		}
		return c.WritePacket(pk)
	}
	base := initMods(allMods, sendToClient, sendToServer)
	stopTick := runTicker(allMods)
	defer stopTick()

	// Read server packets, cache state, forward to client if connected
	go func() {
		for {
			pk, err := server.ReadPacket()
			if err != nil {
				return
			}
			ctx := &mods.Context{
				Packet:       pk,
				Direction:    mods.FromServer,
				SendToClient: sendToClient,
				SendToServer: sendToServer,
			}
			for _, m := range allMods {
				m.HandlePacket(ctx)
				if ctx.Drop {
					break
				}
			}
			if !ctx.Drop {
				// Cache key state packets for replay on takeover
				bot.mu.Lock()
				switch pk.(type) {
				case *packet.UpdateAbilities, *packet.UpdateAttributes,
					*packet.PlayerList, *packet.SetTime,
					*packet.SetDifficulty, *packet.GameRulesChanged,
					*packet.SetScore, *packet.SetScoreboardIdentity:
					bot.cachedPkts = append(bot.cachedPkts, pk)
				}
				c := bot.client
				bot.mu.Unlock()
				if c != nil {
					c.WritePacket(pk)
				}
			}
		}
	}()

	// Wait for real client takeover
	<-bot.clientReady
	log.Println("[Bot] client connected, handing over")
	fmt.Println("[BOT_HANDOVER]")

	client := bot.client
	if err := client.StartGame(server.GameData()); err != nil {
		log.Printf("[Bot] StartGame failed: %v", err)
		return
	}

	// Replay cached state
	bot.mu.Lock()
	for _, pk := range bot.cachedPkts {
		client.WritePacket(pk)
	}
	cached := len(bot.cachedPkts)
	bot.mu.Unlock()
	log.Printf("[Bot] replayed %d cached packets", cached)

	// Request chunk reload
	server.WritePacket(&packet.PlayerAction{
		EntityRuntimeID: mods.State.RuntimeID,
		ActionType:      protocol.PlayerActionRespawn,
	})

	// Normal pipe session
	errc := make(chan error, 2)
	go func() { errc <- pipe(client, server, mods.FromClient, allMods, base) }()
	go func() { errc <- pipe(server, client, mods.FromServer, allMods, base) }()
	log.Printf("[Bot] handover session ended: %v", <-errc)
	client.Close()
}
