# Job: AClient
```yaml
token: cu7f3goontys8ctra5nd8hy59y
url: "100.111.247.128:8065"
```
```javascript
import {publishEvent} from "matterless";

import 'babel-polyfill';
import 'isomorphic-fetch';
import ws from "ws";
global.WebSocket = ws;
import websocketClient from 'mattermost-redux/client/websocket_client.js';

let wsClient = websocketClient['default'];

async function run() {
    try {
        wsClient.setEventCallback(function(msg) {
            publishEvent(`mattermost:${msg.event}`, msg);
        })
        await wsClient.initialize(process.env.TOKEN, {
            connectionUrl: `ws://${process.env.URL}/api/v4/websocket`
        });
    } catch(e) {
        console.error(e);
    }
}

function stop() {
    console.log("Shutting down");
    wsClient.close();
}
```
# Events
```yaml
mattermost:hello:
  - MyCustomEventFunc
```

# Function: MyCustomEventFunc
```javascript
function handle(evt) {
    console.log("Custom event trigggered", evt);
}
```
