# macro mattermostListener
Implements a mattermost event listener, authenticating to a specific `url` using a `token` listening to `events` and triggering subscribed functions appropriately.

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

This is the macro:


    # job {{$name}}
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
    import {events} from "./matterless.ts";

    let socket;
 
    function init(config) {
        console.log("Starting mattermost client with config", config);
        if(!config.token || !config.url) {
           console.error("Token and URL not configured yet.");
           return
        }
        return new Promise((resolve, reject) => {
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
                    events.publish(`${config.name}:${parsedEvent.event}`, parsedEvent);
                }
            });
        });
    }

    function stop() {
       console.log("Shutting down Mattermost client");
       if(socket) {
          socket.close();
       }
    }
    ```

    # events
    ```yaml
    {{range $eventName, $fns := $input.events}}
    "{{$name}}:{{$eventName}}":
    {{range $fns}}  - {{.}}{{end}} 
    {{end}}
    ```

# Bonus: Mattermost client

You can connect to Mattermost as follows:

```javascript
class Mattermost {
    constructor(url, token) {
        this.url = `${url}/api/v4`;
        this.token = token;
    }

    async performFetch(path, method, body) {
        let result = await fetch(`${this.url}${path}`, {
            method: method,
            headers: {
                'Authorization': `bearer ${this.token}`
            },
            body: body ? JSON.stringify(body) : undefined
        });
        return result.json();
    }

    async getMe() {
        return this.performFetch("/users/me", "GET");
    }

    async getUserTeams(userId) {
        return this.performFetch(`/users/${userId}/teams`, "GET");
    }

    async getPrivateChannels(teamId) {
        return this.performFetch(`/teams/${teamId}/channels/private`, "GET");
    }

    async getChannelByName(teamId, name) {
        return this.performFetch(`/teams/${teamId}/channels/name/${name}`, "GET")
    }

    async createChannel(channel) {
        return this.performFetch("/channels", "POST", channel);
    }
    
    async createPost(post) {
        return this.performFetch("/posts", "POST", post);
    }

    async updatePost(post) {
        return this.performFetch(`/posts/${post.id}`, "PUT", post);
    }

    async deletePost(post)  {
        return this.performFetch(`/posts/${post.id}`, "DELETE");
    }
}
```
