const FidotronCmds = [
    "update", // 0
    "error", // 1
    "subrequest", // 2
    "substarted", // 3
    "unsubrequest", // 4
    "substopped" // 5
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
                    that.Send({Cmd:FidotronCmdsMapped["subrequest"], Topic:s});
                }
            }
        };

        this.socket.onclose = function(event) {
            that.observer.Disconnected();
            that.socket = null;

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

            switch(data.Cmd) {
            case FidotronCmdsMapped["update"]:
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

    Send(msg) {
        if(this.socket != null) {
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

            this.Send({Cmd:FidotronCmdsMapped["subrequest"], Topic:pattern});
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

                this.Send({Cmd:FidotronCmdsMapped["unsubrequest"], Topic:pattern});
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

        this.Send({"Cmd":FidotronCmdsMapped["update"], "Topic": topic, "Payload": payload});
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