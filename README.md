# libp2p-musings

This repo is active WIP, testing out the libp2p modules in golang. Haven't had much success with the js libraries and thought of giving the golang-counterparts a "go". Get it? no? okay

## Instructions

1. To run the app, you need to pass in a couple args, 


To run as a "Publisher"

`make run -publisher`


To run as a "Subscriber",

`make run -subscriber`

2. Build with `make build`

Currently uses mDNS for peer discovery, and ws(hasn't been added to the transport chain LOL oops) or quic as transports. Uses GossipSub for the pubsub.


