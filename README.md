# fidotron
Experiments in small scale distributed applications

It's a super basic and simplistic message broker with the intention of providing a sandpit 
with which to experiment with smart home type applications. Ideas are being shamelessly 
stolen from things like MQTT and Android and hoping to avoid some of the problems.

This is in the process of being extracted from a larger mess and cleaned up for wider
consumption.

## TODO
* Move to gorilla websockets and remove our ping/pong noise DONE
* Evaluate the app running logic to see if it should be kept/modified/discarded (removed) DONE
* Applications
** Some sort of reusable application framework
** MDNS application
** Nanoleaf application
** DLNA application
** "One bus" style persistence, as an application(!)
** MQTT bridge
** Zigbee bridge
* JavaScript and web UI
** Ensure the pub/sub logic is working both in the server and the js
** Different panel "types" (sliders etc.) in js
** Reorderable panels
* Core
** Change socket so binary not json based
** Use of name tables per connection to reduce size needed
** Raw socket as well as websocket support
** Authentication/authorization (and sub parts)
* Android
** AndroidAsync based client (derived from the pub/sub js)
** Android video player controlled from bus