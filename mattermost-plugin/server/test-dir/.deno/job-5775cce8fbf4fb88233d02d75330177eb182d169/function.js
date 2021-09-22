import {events} from "./matterless.ts";

let socket, config;

function init(cfg) {
    console.log("Starting mattermost client");
    config = cfg;
    if(!config.token || !config.url) {
       console.error("Token and URL not configured yet.");
       return
    }

    return connect();
}

async function connect() {
    const url = `${config.url}/api/v4/websocket`.replaceAll("https://", "wss://").replaceAll("http://", "ws://");
    socket = new WebSocket(url);
    socket.addEventListener('open', e => {
        socket.send(JSON.stringify({
                "seq": 1,
                "action": "authentication_challenge",
                "data": {
                    "token": config.token
                }
            }
        ));
    });
    socket.addEventListener('message', function (event) {
        const parsedEvent = JSON.parse(event.data)
        // console.log('Message from server ', parsedEvent);
        if(parsedEvent.seq_reply === 1) {
            // Auth response
            if(parsedEvent.status === "OK") {
                console.log("Authenticated.");
            } else {
                console.error("Could not authenticate", parsedEvent);
            }
        }
        if(config.events.indexOf(parsedEvent.event) !== -1) {
            events.publish(`${config.name}:${parsedEvent.event}`, parsedEvent);
        }
    });
    socket.addEventListener('close', function(event) {
        console.error("Connection closed, authentication failed? Reconnecting in 1s...");
        setTimeout(() => {
            connect();
        }, 1000);
    });
}

function stop() {
   console.log("Shutting down Mattermost client");
   if(socket) {
      socket.close();
   }
}


var _init, _start, _run, _stop, _handle;
(function() {
    var initData = {"events":["post_deleted","post_edited","posted"],"name":"MatterlessBot","token":"n1wmzk17qpnei8hh6kmticnuzy","url":"http://localhost:8065"};
    // Initialization
    try {
        _init = init.bind(null, initData);
    } catch (e) {
        _init = () => {
        };
    }

// Functions
    try {
        _handle = handle;
    } catch (e) {
        _handle = () => {
        };
    }

// Jobs
    try {
        _start = start;
    } catch (e) {
        _start = () => {
        };
    }
    try {
        _run = run;
    } catch (e) {
        _run = () => {
        };
    }

    try {
        _stop = stop;
    } catch (e) {
        _stop = () => {
        };
    }
})();


export {
    _init as init,
    _start as start,
    _stop as stop,
    _run as run,
    _handle as handle
};
