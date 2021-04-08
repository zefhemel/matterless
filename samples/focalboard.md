# Focalboard
Super elementary focalboard integration example. Requires the following store variables to be set:

* `config:focalboard.url`
* `config:focalboard.token`


# job FocalBoardListener
```yaml
init:
  url: ${config:focalboard.url}
  token: ${config:focalboard.token}
  workspaceId: "0"
  event: "focalboard:update"
```

```javascript
import {publishEvent} from "./matterless.ts";
import { Focalboard } from "https://gist.githubusercontent.com/zefhemel/cdfddf3276524dc2cf18b9d133734e83/raw/81cc9721c20f40e1f04bbc0103797e2e18e9496d/focalboard-deno.js"

let config;
let fb;

async function init(cfg) {
    config = cfg;
    console.log("Starting focalboard listener with config", config);
    if(!config.token || !config.url) {
       console.error("Token and URL not configured yet.");
       return
    }
    fb = new Focalboard(config.url, config.token, config.workspaceId);
    const wsUrl = `${config.url}/ws/onchange`.replaceAll("https://", "wss://").replaceAll("http://", "ws://");
    const socket = new WebSocket(wsUrl);
    socket.addEventListener('open', e => {
        socket.send(JSON.stringify({
            action: "AUTH",
            token: config.token,
            workspaceId: config.workspaceId
        }));
        fb.allBlocks().then(allBlocks => {
            socket.send(JSON.stringify({
                "action":"ADD",
                "blockIds": allBlocks.map(block => block.id),
                "workspaceId":"0",
                "readToken":""
            }));
        });
    })
    socket.addEventListener('message', function (event) {
        let parsedMessage = JSON.parse(event.data);
        publishEvent(config.event, parsedMessage);
    });
}

function stop() {
    console.log("Shutting down focalboard listener");
    // wsClient.close();
}
```

# events
```
"focalboard:update":
- MyFocalFunction
```

# function MyFocalFunction
```javascript
function handle(event) {
    console.log("Received event", event);
}
```
