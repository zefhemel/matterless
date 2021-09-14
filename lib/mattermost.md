# Mattermost macro library

Implements the following potentially useful Mattermost macros:

* `mattermostListener`
* `mattermostBot`
* `mattermostInstanceWatcher`

Check each macro for more documentation.

# import

* ./cron.md

## macro mattermostListener

Implements a few mattermost event listener, authenticating to a specific `url` using a `token` listening to `events` and
triggering subscribed functions appropriately.

```yaml
schema:
  type: object
  properties:
    url:
      type: string
    token:
      type: string
    events:
      type: object
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
        token: "{{$arg.token}}"
        url: "{{$arg.url}}"
        events:
        {{range $eventName, $v := $arg.events}}
        - {{$eventName}}
        {{end}}
    ```
    
    ```javascript
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
    ```

    # events
    ```yaml
    {{range $eventName, $fns := $arg.events}}
    "{{$name}}:{{$eventName}}":
    {{range $fns}}  - {{.}}{{end}} 
    {{end}}
    ```

# macro mattermostBot

```yaml
schema:
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
    url: {{yaml $arg.url}}
    token: ${token:{{$name}}}
    events:
      {{yaml $arg.events | prefixLines "  " }}
    ```

    # events
    ```yaml
    init:
      - {{$name}}BotCreate
    ```
    
    # function {{$name}}BotCreate
    ```yaml
    init:
      url: {{yaml $arg.url}}
      admin_token: {{yaml $arg.admin_token}}
      username: {{yaml $arg.username}}
      display_name: {{yaml $arg.display_name}}
      description: {{yaml $arg.description}}
      teams:
      {{range $arg.teams}}
      - {{.}}
      {{- end}}
      bot_token_config: "token:{{$name}}"
    ```
    
    ```javascript
    import {store, restartApp} from "./matterless.ts";
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
            user.id = user.user_id;
        }
        // User exists, let's create a token
        let token = await client.createUserAccessToken(user.id, "Matterless generated token");
        await Promise.all(config.teams.map(async (teamName) => {
            return client.addUserToTeam(user.id, (await client.getTeamByName(teamName)).id);
        }));
        await store.put(config.bot_token_config, token.token);
        restartApp();
    }
    ```

# macro mattermostInstanceWatcher

```yaml
schema:
  type: object
  properties:
    url:
      type: string
    events:
      type: object
      additionalProperties:
        type: array
        items:
          type: string
```

    # cron {{$name}}Cron
    ```yaml
    schedule: "0 * * * * *"
    function: "{{$name}}CheckUpgrade"
    ```

    # events
    ```yaml
    {{range $eventName, $fns := $arg.events}}
    "{{$name}}:{{$eventName}}":
    {{range $fns}}  - {{.}}{{end}} 
    {{end}}
    ```

    ## function {{$name}}CheckUpgrade
    ```yaml
    init:
      url: {{yaml $arg.url}}
      ns: {{$name}}
    ```
    
    ```javascript
    import {store, events} from "./matterless.ts";
    let config;
    
    function init(cfg) {
        config = cfg;
    }
    
    async function handle() {
        let result = await fetch(`${config.url}/api/v4/config/client?format=old`);
        let json = await result.json();
        let featureFlags = {};
        for(const [key, value] of Object.entries(json)) {
            if(key.indexOf("FeatureFlag") === 0) {
                const flagName = key.substring("FeatureFlag".length);
                let previousValue = await store.get(`${config.ns}:flag:${flagName}`);
                if(previousValue !== json[key]) {
                    await events.publish(`${config.ns}:flag:${flagName}`, {
                        flag: flagName,
                        oldValue: previousValue,
                        newValue: value
                    });
                    await store.put(`${config.ns}:flag:${flagName}`, value);
                }
            }
        }
        let oldVersion = (await store.get(`${config.ns}:version`)) || "db01f2a91b67e24187294dbe30cca1cf8fc6e494";
        let version = json.BuildHash;
        if(oldVersion != version) {
            await events.publish(`${config.ns}:upgrade`, {
                oldVersion: oldVersion,
                newVersion: version
            });
            await store.put(`${config.ns}:version`, version);
        }
    }
    ```