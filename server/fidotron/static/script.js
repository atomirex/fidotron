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

    pathSections(path) {
        let s = path.split("/");
        let p = [];
        for(let i=0; i<s.length; i++) {
            if(s[i] !== "") {
                p.push(s[i]);
            }
        }
        return p;
    }
    
    match(output, path) {
        let sections = this.pathSections(path);
        return this.innerMatch(output, sections, 0);
    }

    innerMatch(output, path, index) {
        if (index < path.length) {
            if (this.children["#"] != null) {
                for(let s of this.children["#"].subscribers) {
                    output.add(s);
                }
            }

            if (this.children["+"] != null) {
                this.children["+"].innerMatch(output, path, index+1);
            }

            if (this.children[path[index]] != null) {
                this.children[path[index]].innerMatch(output, path, index+1);
            }
        } else if (index == path.length) {
            for(let s of this.subscribers) {
                output.add(s);
            }
        }
    }

    addSubscription(sub, path) {
        this.innerAddSubscription(sub, this.pathSections(path), 0);
    }

    innerAddSubscription(sub, path, index) {
        if(index < path.length) {
            if( this.children[path[index]] == null) {
                this.children[path[index]] = new FidotronMatchNode();
            }

            this.children[path[index]].innerAddSubscription(sub, path, index+1);
        } else if (index == path.length) {
            this.subscribers.add(sub);
        }
    }

    removeSubscription(sub, path) {
        this.innerRemoveSubscription(sub, this.pathSections(path), 0);
    }

    innerRemoveSubscription(sub, path, index) {
        if (index < path.length) {
            if (this.children[path[index]] == null) {
                return;
            }

            this.children[path[index]].innerRemoveSubscription(sub, path, index+1);
        } else if (index == path.length) {
            this.subscribers.delete(sub);
        }
    }
}

class FidotronPanel {
    constructor(title) {
        this.title = title;

        let e = document.createElement("div");
        e.className = "panel";

        let titleElement = document.createElement("p");
        titleElement.className = "panel-title";
        titleElement.innerHTML = this.title;

        let panelCloser = document.createElement("p");
        panelCloser.className = "panel-closer";
        panelCloser.innerHTML = "x";
        var that = this;
        panelCloser.onclick = function() {
            that.close();
        };

        let panelBody = document.createElement("p");
        panelBody.className = "panel-body";

        e.appendChild(titleElement);
        e.appendChild(panelCloser);
        e.appendChild(panelBody);

        this.element = e;
        this.panelBody = panelBody;
    }

    SetCancelHandler(cancelHandler) {
        this.cancelHandler = cancelHandler;
    }

    Element() {
        return this.element;
    }

    Append(message) {
        this.panelBody.innerHTML = message + "<br />" + this.panelBody.innerHTML;
    }

    close() {
        this.cancelHandler(this);
        document.getElementById("panels-container").removeChild(this.element);
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

    match(topic) {
        let result = new Set();

	    this.matcher.match(result, topic);

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
        if(this.socket != null && this.socket.readyState == WebSocket.OPEN) {
            this.socket.send(JSON.stringify(msg));
        }
    }

    Subscribe(pattern, listener) {
        if(!this.subscriptions.hasOwnProperty(pattern)) {
            this.subscriptions[pattern] = [];
        }

        if(!this.subscriptions[pattern].includes(listener)) {
            this.subscriptions[pattern].push(listener);

            this.matcher.addSubscription(listener, pattern);

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

                this.matcher.removeSubscription(listener, pattern);

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
 
function subscribePanel(pattern) {
    let p = new FidotronPanel(pattern);

    document.getElementById("panels-container").appendChild(p.Element());

    let subListener = function(topic, payload) {
        p.Append(topic+": "+payload);
    };

    p.SetCancelHandler(function(panel) {
        c.Unsubscribe(pattern, subListener);
    });

    c.Subscribe(pattern, subListener);
}

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

    c.ShouldConnect(true);

    subscribePanel("#");
}