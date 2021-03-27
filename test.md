# Job: AClient
```yaml
config:
    token: cu7f3goontys8ctra5nd8hy59y
    url: "100.111.247.128:8065"
    events:
      - hello
      - posted
```

```javascript
import {publishEvent} from "matterless";

import 'babel-polyfill';
import 'isomorphic-fetch';
import ws from "ws";
global.WebSocket = ws;
import websocketClient from 'mattermost-redux/client/websocket_client.js';

let wsClient = websocketClient['default'];

async function init(config) {
    console.log("Starting mattermost client with config", config)
    try {
        wsClient.setEventCallback(function(msg) {
            if(config.events.indexOf(msg.event) != -1) {
                publishEvent(`mattermost:${msg.event}`, msg);
            }
        })
        await wsClient.initialize(config.token, {
            connectionUrl: `ws://${config.url}/api/v4/websocket`
        });
    } catch(e) {
        console.error(e);
    }
}

function stop() {
    console.log("Shutting down mattermost client");
    wsClient.close();
}
```
# Events
```yaml
mattermost:hello:
  - MyCustomEventFunc
mattermost:posted:
  - MyCustomEventFunc
```

# Function: MyCustomEventFunc
```javascript
function handle(evt) {
    console.log("Custom event trigggered", evt);
}
```
