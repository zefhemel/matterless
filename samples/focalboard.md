# Focalboard
Super elementary focalboard integration example. Requires the following store variables to be set:

* `config:focalboard.url`
* `config:focalboard.token`

Publishes `focalboard:create`, `focalboard:update` and `focalboard:delete` events.

# events
```
focalboard:create:
- MyFocalCreateFunction
focalboard:update:
- MyFocalUpdateFunction
focalboard:delete:
- MyFocalDeleteFunction
```

# function MyFocalCreateFunction
```javascript
function handle(event) {
    console.log("Card created", event.block);
}
```

# function MyFocalUpdateFunction
```javascript
function handle(event) {
    console.log("Card updated", event.block);
}
```

# function MyFocalDeleteFunction
```javascript
function handle(event) {
    console.log("Card deleted", event.block);
}
```







# job FocalBoardListener
```yaml
init:
  url: ${config:focalboard.url}
  token: ${config:focalboard.token}
  workspaceId: "0"
  create_event: "focalboard:create"
  update_event: "focalboard:update"
  delete_event: "focalboard:delete"
```

```javascript
import {events} from "./matterless.ts";

let config;
let fb, socket;

async function init(cfg) {
    config = cfg;
    console.log("Starting focalboard listener with config", config);
    if(!config.token || !config.url) {
       console.error("Token and URL not configured yet.");
       return
    }
    fb = new Focalboard(config.url, config.token, config.workspaceId);
    const wsUrl = `${config.url}/ws/onchange`.replaceAll("https://", "wss://").replaceAll("http://", "ws://");
    socket = new WebSocket(wsUrl);
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
        // console.log("Got event", event)
        let parsedMessage = JSON.parse(event.data);
        if(parsedMessage.action === 'UPDATE_BLOCK' && parsedMessage.block.deleteAt) {
            events.publish(config.delete_event, parsedMessage);
        } else if(parsedMessage.block.updateAt === parsedMessage.block.createAt) {
            events.publish(config.create_event, parsedMessage);
        } else {
            events.publish(config.update_event, parsedMessage);
        }
    });
    socket.addEventListener('close', function() {
        console.error("Connection closed");
    })
}

function stop() {
    console.log("Shutting down focalboard listener");
    socket.close();
}

export class Focalboard {
    constructor(url, token, workspaceId) {
        this.url = url;
        this.token = token;
        this.workspaceId = workspaceId;
    }

    async allBlocks() {
        let result = await fetch(`${this.url}/api/v1/workspaces/${this.workspaceId}/blocks?type=board`, {
            headers: {
                "accept": "application/json",
                "authorization": `Bearer ${this.token}`,
                "x-requested-with": "XMLHttpRequest"
            },
            "method": "GET",
        });
        return await result.json();
    }

    async addBlock(blocks) {
        let result = fetch(`${this.url}/api/v1/workspaces/${this.workspaceId}/blocks`, {
            "headers": {
                "accept": "application/json",
                "authorization": `Bearer ${this.token}`,
                "content-type": "application/json",
                "x-requested-with": "XMLHttpRequest"
            },
            "body": JSON.stringify(blocks),
            "method": "POST",
        });
        return await result.json();
    }
}
```
