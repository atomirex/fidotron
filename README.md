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
    * Some sort of reusable application framework (go)
    * MDNS application
    * Nanoleaf application
    * DLNA application
    * "One bus" style persistence, as an application(!)
    * MQTT bridge
    * Zigbee bridge
    * BTLE bridge
    * Fuzzy logic
* JavaScript and web UI
    * Ensure the pub/sub logic is working both in the server and the js
    * Move status to bottom of window
    * Different panel "types" (sliders etc.) in js
    * Generally movable panels/layouts in addition to the ordered ones
    * Panel configuration via the message bus
* Core
    * Change socket so binary not json based
    * Use of name tables per connection to reduce size needed
    * Raw socket as well as websocket support
    * Authentication/authorization (and sub parts)
* Android
    * AndroidAsync based client (derived from the pub/sub js)
    * Android video player controlled from bus
* Arduino
    * Blast implementation
* Build and tools
    * Continuous build and test on OpenBSD(x86) and Linux(x86 and ARM)
    * Remote updates for OpenBSD(x86) and Linux(x86 and ARM)
    * Ability to actually perform an end-to-end OpenBSD update remotely