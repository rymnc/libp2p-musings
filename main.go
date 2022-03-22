package main

import (
	"context"
	"flag"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	noise "github.com/libp2p/go-libp2p-noise"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	quic "github.com/libp2p/go-libp2p-quic-transport"
	muxer "github.com/libp2p/go-libp2p-yamux"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	ws "github.com/libp2p/go-ws-transport"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
)

const DiscoveryInterval = time.Minute
const DiscoveryServiceTag = "rymnc-pubsub"

func Publish(ctx context.Context, topic *pubsub.Topic, msg string) error {
	return topic.Publish(ctx, []byte(msg))
}

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	publisher := flag.Bool("publisher", false, "publisher")
	subscriber := flag.Bool("subscriber", false, "subscriber")
	flag.Parse()

	if *publisher && *subscriber {
		log.Fatal().
			Msg("cannot be both publisher and subscriber")
	}

	if !*publisher && !*subscriber {
		log.Fatal().
			Msg("must be either publisher or subscriber")
	}

	ctx := context.Background()

	h, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip6/::1/udp/0/quic"),
		libp2p.ChainOptions(
			libp2p.Transport(ws.New),
			libp2p.Transport(quic.NewTransport),
		),
		libp2p.Security(noise.ID, noise.New),
		libp2p.Muxer("/yamux/1.0.0", muxer.DefaultTransport),
	)

	if err != nil {
		log.Fatal().Err(err).Msg("error creating libp2p host")
	}

	log.Debug().
		Str("id", h.ID().Pretty()).
		Str("address", h.Addrs()[0].String()).
		Msg("libp2p node created with config")

	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		log.Fatal().AnErr("error creating gossipsub", err)
	}

	err = setupDiscovery(h)
	if err != nil {
		log.Fatal().AnErr("error setting up mDNS discovery", err)
	}

	topic, err := ps.Join("test")
	if err != nil {
		log.Fatal().AnErr("error joining topic", err)
	}

	if *subscriber == true {
		sub, err := topic.Subscribe()
		if err != nil {
			log.Fatal().AnErr("error subscribing to topic", err)
		}
		for {
			msg, err := sub.Next(ctx)
			if err != nil {
				log.Fatal().AnErr("error getting next message", err)
			}
			if msg.ReceivedFrom == h.ID() {
				continue
			}
			log.Info().
				Str("from", msg.ReceivedFrom.Pretty()).
				Str("msg", string(msg.GetData())).
				Msg("received message")
		}
	}

	if *publisher == true {
		log.Info().Msg("publishing...")
		for range time.Tick(time.Millisecond * 5) {
			Publish(ctx, topic, "heartbeat")
		}
	}
}

type discoveryNotifee struct {
	h host.Host
}

func setupDiscovery(h host.Host) error {
	// setup mDNS discovery to find local peers
	s := mdns.NewMdnsService(h, DiscoveryServiceTag, &discoveryNotifee{h: h})
	return s.Start()
}

func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	log.Debug().
		Str("id", pi.ID.Pretty()).
		Msg("discovered new peer")

	err := n.h.Connect(context.Background(), pi)
	if err != nil {
		log.Error().
			Str("id", pi.ID.Pretty()).
			AnErr("err", err).
			Msg("error connecting to peer")
	}
}
