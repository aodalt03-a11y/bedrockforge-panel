package proxy

import (
	"fmt"
	"log"

	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"golang.org/x/oauth2"

	"bedrockforge/config"
	"bedrockforge/mods"
)

func Start(cfg *config.Config) error {
	src, err := getTokenSource()
	if err != nil {
		return err
	}
	rPacks := mods.LoadResourcePacks()
	listener, err := minecraft.ListenConfig{
		AuthenticationDisabled: true,
		StatusProvider:         minecraft.NewStatusProvider("BedrockForge", "BedrockForge v2"),
		ResourcePacks:          rPacks,
	}.Listen("raknet", cfg.ListenAddr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer listener.Close()
	log.Printf("[BF] listening on %s -> %s", cfg.ListenAddr, cfg.ServerAddr)
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("[BF] accept: %v", err)
			continue
		}
		go runSession(conn.(*minecraft.Conn), cfg, src)
	}
}

func runSession(client *minecraft.Conn, cfg *config.Config, src oauth2.TokenSource) {
	log.Printf("[BF] client connected from %s", client.RemoteAddr())
	dialer := minecraft.Dialer{
		ErrorLog:    logger,
		ClientData:  client.ClientData(),
		TokenSource: src,
	}
	server, err := dialer.Dial("raknet", cfg.ServerAddr)
	if err != nil {
		log.Printf("[BF] server dial failed: %v", err)
		client.Close()
		return
	}
	if err := client.StartGame(server.GameData()); err != nil {
		log.Printf("[BF] StartGame failed: %v", err)
		client.Close()
		server.Close()
		return
	}

	allMods := mods.All()
	sendToClient := func(pk packet.Packet) error { return client.WritePacket(pk) }
	sendToServer := func(pk packet.Packet) error { return server.WritePacket(pk) }
	base := initMods(allMods, sendToClient, sendToServer)
	stopTick := runTicker(allMods)
	defer stopTick()

	// Send welcome after mods are initialized
	mods.SendWelcome(sendToClient)

	errc := make(chan error, 2)
	go func() { errc <- pipe(client, server, mods.FromClient, allMods, base) }()
	go func() { errc <- pipe(server, client, mods.FromServer, allMods, base) }()
	err = <-errc
	log.Printf("[BF] session ended: %v", err)
	client.Close()
	server.Close()
}
