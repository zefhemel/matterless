
# MattermostClient: MyClient
```yaml
url: http://pi-jay:8065
token: cu7f3goontys8ctra5nd8hy59y
events:
  hello:
    - MyCustomFunc
  posted:
    - MyCustomFunc
```

# Function: MyCustomFunc
```javascript

async function handle(event) {
    console.log("Got custom event", event);
}
```

# Macro: MattermostClient
Here's the schema
```yaml
input_schema:
   type: object
   properties:
      url:
         type: string
      token:
         type: string
      events:
        type: object
        propertyNames:
          pattern: "^[A-Za-z_][A-Za-z0-9_]*$"
        additionalProperties:
          type: array
          items:
            type: string
   additionalProperties: false
   required:
   - url
   - token
```

And the actual template implementation:

    # Job: {{$name}}
    ```yaml
    init:
        name: "{{$name}}"
        token: "{{$input.token}}"
        url: "{{$input.url}}"
        events:
        {{range $eventName, $v := $input.events}}
        - {{$eventName}}
        {{end}}
    ```
    
    ```javascript
    import {publishEvent} from "./matterless.ts";
    
    function init(config) {
        console.log("Starting mattermost client with config", config);
        return new Promise((resolve, reject) => {
            const url = `${config.url}/api/v4/websocket`.replaceAll("https://", "wss://").replaceAll("http://", "ws://");
            const socket = new WebSocket(url);
            socket.addEventListener('open', e => {
                socket.send(JSON.stringify({
                        "seq": 1,
                        "action": "authentication_challenge",
                        "data": {
                            "token": config.token
                        }
                    }
                ));
            })
            socket.addEventListener('message', function (event) {
                const parsedEvent = JSON.parse(event.data)
                if(parsedEvent.seq_reply === 1) {
                    // Auth response
                    if(parsedEvent.status === "OK") {
                        return resolve();
                    } else {
                        return reject(event);
                    }
                }
                console.log('Message from server ', parsedEvent);
                if(config.events.indexOf(parsedEvent.event) !== -1) {
                    publishEvent(`${config.name}:${parsedEvent.event}`, parsedEvent);
                }
            });
        });
    }
    
    function stop() {
        console.log("Shutting down mattermost client");
        wsClient.close();
    }
    ```

    # Events
    ```yaml
    {{range $eventName, $fns := $input.events}}
    "{{$name}}:{{$eventName}}":
    {{range $fns}}  - {{.}}{{end}} 
    {{end}}
    ```
