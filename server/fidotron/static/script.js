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
    constructor() {
        this.shouldConnect = false;
        this.backoffMinimum = 20;
        this.backoff = this.backoffMinimum;
        this.backoffMaximum = 10000;

        this.observers = new Set();
        this.logData = [];

        this.subscriptions = {};
        this.matcher = new FidotronMatchNode();
    }

    AddObserver(observer) {
        this.observers.add(observer);
    }

    RemoveObserver(observer) {
        this.observers.delete(observer);
    }

    log(msg) {
        this.logData.unshift(msg);
        
        while(this.logData.length > 25) {
            this.logData.pop();
        }

        let html = "";
        for(var i=0; i<this.logData.length; i++) {
            html += this.logData[i];
            html += "<br />";
        }

        for (let o of this.observers) {
            o.LogUpdated(msg, html);
        }
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

        this.log("Connecting");
        for (let o of this.observers) {
            o.Connecting();
        }
        this.socket = new WebSocket("ws://127.0.0.1:8080/websocket");

        var that = this;

        this.socket.onopen = function(event) {
            this.backoff = this.backoffMinimum;
            that.log("Connected");
            for (let o of that.observers) {
                o.Connected();
            }
            for(var s in that.subscriptions) {
                if(that.subscriptions.hasOwnProperty(s)) {
                    that.Send({Cmd:FidotronCmdsMapped["subrequest"], Topic:s});
                }
            }
        };

        this.socket.onclose = function(event) {
            that.log("Disconnected");
            for (let o of that.observers) {
                o.Disconnected();
            }
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
            that.log("Socket error");
            for (let o of that.observers) {
                o.Error("Socket error");
            }
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
            this.log("Error with launch request for "+name);
            for (let o of that.observers) {
                o.Error("Error with launch request for "+name);
            }
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

function addSubscriptionPanel(pattern) {
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

function addLogPanel() {
    let p = new FidotronPanel("System Log");

    document.getElementById("panels-container").appendChild(p.Element());

    let observer = {
        LogUpdated: function(msg, html) {
            p.Append(msg);
        },

        Connecting: function() {
        },

        Connected: function() {
        },

        Disconnected: function() {
        }, 

        Error: function(msg) {
        }
    };

    p.SetCancelHandler(function(panel) {
        c.RemoveObserver(observer);
    });

    c.AddObserver(observer);
}
 
function addPanel(panelType, pattern) {
    switch(panelType) {
        case "subscription":
            addSubscriptionPanel(pattern);
            break;
        case "syslog":
            addLogPanel();
            break;
        default:
            console.log("Unhandled panel addition request " + panelType);
            break;
    }
}

function init() {
    var statusView = document.getElementById("status");
    var logview = document.getElementById("log");

    function log(msg, html) {
        statusView.innerHTML = msg;
        logview.innerHTML = html;
    }

    c = new FidotronConnection();

    c.AddObserver({
        LogUpdated: function(msg, html) {
            log(msg, html);
        },

        Connecting: function() {
        },

        Connected: function() {
        },

        Disconnected: function() {
        }, 

        Error: function(msg) {
        }
    });

    c.ShouldConnect(true);

    addPanel("subscription", "#");
}

function showStartMenu() {
    let e = document.getElementById("startmenu");
    let style = e.style.visibility;
    if(style == "visible") {
        e.style.visibility = "hidden";
    } else {
        e.style.visibility = "visible";
    }
}