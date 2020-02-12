const FidotronCmds = [
    "ping", // 0
    "pong", // 1
    "update", // 2
    "error", // 3
    "subrequest", // 4
    "substarted", // 5
    "unsubrequest", // 6
    "substopped" // 7
];

function mapOfArray(a) {
    let r = {};

    for(let i=0; i<a.length; i++) {
        r[a[i]] = i;
    }

    return r;
}

const FidotronCmdsMapped = mapOfArray(FidotronCmds);

class FidotronMatchNode {
    constructor() {
		this.children = {};
        this.subscribers = new Set();
    }
    
    match(output, path, index) {
        if (index < path.length) {
            if (this.children["#"] != null) {
                for(let s of this.children["#"].subscribers) {
                    output.add(s);
                }
            }

            if (this.children["+"] != null) {
                n.children["+"].match(output, path, index+1);
            }

            if (this.children[path[index]] != null) {
                this.children[path[index]].match(output, path, index+1);
            }
        } else if (index == path.length) {
            for(let s of this.subscribers) {
                output.add(s);
            }
        }
    }

    addSubscription(sub, path, index) {
        if(index < path.length) {
            if( this.children[path[index]] == null) {
                this.children[path[index]] = new FidotronMatchNode();
            }

            this.children[path[index]].addSubscription(sub, path, index+1);
        } else if (index == path.length) {
            this.subscribers.add(sub);
        }
    }

    removeSubscription(sub, path, index) {
        if (index < path.length) {
            if (this.children[path[index]] == null) {
                return;
            }

            this.children[path[index]].removeSubscription(sub, path, index+1);
        } else if (index == path.length) {
            this.subscribers.delete(sub);
        }
    }
}

class FidotronConnection {
    constructor(observer) {
        this.shouldConnect = false;
        this.backoffMinimum = 20;
        this.backoff = this.backoffMinimum;
        this.backoffMaximum = 10000;

        this.observer = observer;
        this.subscriptions = {};
        this.matcher = new FidotronMatchNode();

        this.toSendPing = null;
        this.pingSent = null;
    }

    pathSections(path) {
        let s = path.split("/");
        let p = [];
        for(let i=0; i<s.length; i++) {
            if(s[i] !== "") {
                p.push(s);
            }
        }
        return p;
    }

    match(topic) {
        let result = new Set();

	    this.matcher.match(result, this.pathSections(topic), 0);

	    return result;
    }

    checkConnectionState() {
        if(this.shouldConnect) {
            if(this.socket == null) {
                this.backoff = this.backoffMinimum;
                this.Connect();
            }
        } else {
            if(this.socket != null) {
                this.socket.close();
            }
        }
    }

    Connect() {
        if(this.socket != null || this.shouldConnect == false) {
            return;
        }

        this.observer.Connecting();
        this.socket = new WebSocket("ws://127.0.0.1:8080/websocket");

        var that = this;

        this.socket.onopen = function(event) {
            this.backoff = this.backoffMinimum;
            that.observer.Connected();
            for(var s in that.subscriptions) {
                if(that.subscriptions.hasOwnProperty(s)) {
                    that.Send({Cmd:4, Topic:s});
                }
            }
            that.schedulePing();
        };

        this.socket.onclose = function(event) {
            that.observer.Disconnected();
            that.socket = null;

            if(that.pingSent != null) {
                clearTimeout(that.pingSent);
                that.pingSent = null;
            }

            if(that.toSendPing != null) {
                clearTimeout(that.toSendPing);
                that.toSendPing = null;
            }

            if(that.shouldConnect) {
                that.backoff = that.backoff * 2;
                if(that.backoff > that.backoffMaximum) {
                    that.backoff = that.backoffMaximum;
                }

                setTimeout(function() {
                    that.Connect();
                }, that.backoff);
            }
        };

        this.socket.onerror = function(event) {
            that.observer.Error("Socket error");
        };

        this.socket.onmessage = function(event) {
            let data = JSON.parse(event.data);

            if(that.pingSent != null) {
                clearTimeout(that.pingSent);
                that.pingSent = null;
            }

            switch(data.Cmd) {
            case 0: // Ping
                that.Send({Cmd:1});
                break;
            case 1: // Pong
                break;
            case 2: // Update
                let topic = data.Topic;
                let payload = data.Payload;
                let matches = that.match(topic);
                for(let m of matches) {
                    m(topic, payload);
                }
                break;
            }
        }
    }

    cancelPing() {
        if(this.toSendPing != null) {
            clearTimeout(this.toSendPing);
        }
    }

    pongTimedOut() {
        if(this.socket != null) {
            this.socket.close();
        }
    }

    schedulePing() {
        this.cancelPing();

        var that = this;

        this.toSendPing = setTimeout(function() {
            that.Send({Cmd:0});
            that.pingSent = setTimeout(that.pongTimedOut, 2000);
        }, 2000);
    }

    Send(msg) {
        if(this.socket != null) {
            this.schedulePing();
            this.socket.send(JSON.stringify(msg));
        }
    }

    Subscribe(pattern, listener) {
        if(!this.subscriptions.hasOwnProperty(pattern)) {
            this.subscriptions[pattern] = [];
        }

        if(!this.subscriptions[pattern].includes(listener)) {
            this.subscriptions[pattern].push(listener);

            this.matcher.addSubscription(listener, this.pathSections(pattern), 0);

            this.Send({Cmd:4, Topic:pattern});
        }
    }

    Unsubscribe(pattern, listener) {
        if(this.subscriptions.hasOwnProperty(pattern)) {
            let index = this.subscriptions[pattern].indexOf(listener);

            if(index >= 0) {
                this.subscriptions[pattern].splice(index, 1);
            }

            if(this.subscriptions[pattern].length == 0) {
                delete this.subscriptions[pattern]

                this.matcher.removeSubscription(listener, this.pathSections(pattern), 0);

                this.Send({Cmd:6, Topic:pattern});
            }
        }
    }

    RunApp(name) {
        var that = this;

        var xhr = new XMLHttpRequest();
        xhr.open('GET', '/runapp/'+name, true);

        xhr.onerror = function() {
            that.observer.Error("Error with launch request for "+name);
        };

        xhr.send(null);
    }

    Push(topic, payload) {
        /**
        var xhr = new XMLHttpRequest();
        var fd = new FormData();

        xhr.open('POST', '/push', true);

        fd.append("topic", topic);
        fd.append("payload", payload);

        xhr.onerror = function() {
            that.Log("Error with push request for "+topic);
        };

        xhr.send(fd);
        */

        this.Send({"Cmd":2, "Topic": topic, "Payload": payload});
    }

    ShouldConnect(shouldConnect) {
        this.shouldConnect = shouldConnect;

        this.checkConnectionState();
    }
}

var c = null;
 
function init() {
    var logview = document.getElementById("log");

    function log(msg) {
        logview.innerHTML += msg + "<br />";
    }

    c = new FidotronConnection({
        Log: function(msg) {
            log(msg);
        },

        Connecting: function() {
            log("Connecting");
        },

        Connected: function() {
            log("Connected");
        },

        Disconnected: function() {
            log("Disconnected");
        }, 

        Error: function(msg) {
            log("Error "+msg);
        }
    });

    c.Subscribe("#", function(topic, payload) {
        log("Received "+topic+" "+payload);
    });

    c.ShouldConnect(true);
}