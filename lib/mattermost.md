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
            });
            socket.addEventListener('message', function (event) {
                const parsedEvent = JSON.parse(event.data)
                console.log('Message from server ', parsedEvent);
                if(parsedEvent.seq_reply === 1) {
                    // Auth response
                    if(parsedEvent.status === "OK") {
                        return resolve();
                    } else {
                        return reject(event);
                    }
                }
                if(config.events.indexOf(parsedEvent.event) !== -1) {
                    events.publish(`${config.name}:${parsedEvent.event}`, parsedEvent);
                }
            });
            socket.addEventListener('close', function(event) {
               console.error("Connection closed, authentication failed?");
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

# macro mattermostBot
```yaml
input_schema:
  type: object
  properties:
    url:
      type: string
    admin_token:
      type: string
    username:
      type: string
    display_name:
      type: string
    description:
      type: string
    teams:
      type: array
      items:
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
    - admin_token
    - username
    - teams
```

Template:

    # mattermostListener {{$name}}
    ```yaml
    url: {{yaml $input.url}}
    token: ${config:{{$name}}.token}
    events:
      {{yaml $input.events | prefixLines "  " }}
    ```

    # events
    ```yaml
    init:
      - {{$name}}BotCreate
    ```
    
    # function {{$name}}BotCreate
    ```yaml
    init:
      url: {{yaml $input.url}}
      admin_token: {{yaml $input.admin_token}}
      username: {{yaml $input.username}}
      display_name: {{yaml $input.display_name}}
      description: {{yaml $input.description}}
      teams:
      {{range $input.teams}}
      - {{.}}
      {{- end}}
      bot_token_config: "config:{{$name}}.token"
    ```
    
    ```javascript
    import {store} from "./matterless.ts";
    import {Mattermost} from "https://raw.githubusercontent.com/zefhemel/matterless/master/lib/mattermost_client.js";
    
    let client;
    let config;
    
    function init(cfg) {
        config = cfg;
        client = new Mattermost(config.url, config.admin_token);
    }
    
    async function handle() {
        if(await store.get(config.bot_token_config)) {
            // Bot token config already configured, done!
            console.log("Bot token already present, skipping.");
            return;
        }
        let user;
        try {
            user = await client.getUserByUsername(config.username);
            console.log("Existing user", user);
        } catch(e) {
            user = await client.createBot({
                username: config.username,
                display_name: config.display_name,
                description: config.description
            });
            user.id = user.bot_user_id;
        }
        // User exists, let's create a token
        let token = await client.createUserAccessToken(user.id, "Matterless generated token");
        await Promise.all(config.teams.map(async (teamName) => {
            return client.addUserToTeam(user.id, (await client.getTeamByName(teamName)).id);
        }));
        await store.put(config.bot_token_config, token.token);
    }
    ```