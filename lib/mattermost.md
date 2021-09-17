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
    import {publishEvent} from "./matterless.ts";

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
                publishEvent(`${config.name}:${parsedEvent.event}`, parsedEvent);
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
    {{range $fns}}  - {{.}}
    {{end}} 
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
    import {Mattermost} from "./mattermost_client.js";
    
    let client;
    let config;
    console.log("Booting bot create process");
    
    function init(cfg) {
        config = cfg;
        if(!cfg.url) {
            return;
        }
        client = new Mattermost(config.url, config.admin_token);
    }
    
    async function handle() {
        if(!client) {
            return;
        }
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
                    await publishEvent(`${config.ns}:flag:${flagName}`, {
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
            await publishEvent(`${config.ns}:upgrade`, {
                oldVersion: oldVersion,
                newVersion: version
            });
            await store.put(`${config.ns}:version`, version);
        }
    }
    ```

# library mattermost_client.js

```javascript

export class Mattermost {
    constructor(url, token) {
        this.url = `${url}/api/v4`;
        this.token = token;
        this.meCache = null;
        this.userCache = {};
        this.channelCache = {};
        this.channelNameCache = {};
        this.userNameCache = {};
        this.callsMade = 0;
    }

    async performFetch(path, method, body) {
        let result = await fetch(`${this.url}${path}`, {
            method: method,
            headers: {
                'Authorization': `bearer ${this.token}`
            },
            body: body ? JSON.stringify(body) : undefined
        });
        this.callsMade++;
        if (result.status < 200 || result.status > 299) {
            throw Error((await result.json()).message);
        }
        return result.json();
    }

    // Me
    async getMe() {
        return this.performFetch("/users/me", "GET");
    }

    async getMeCached() {
        if (!this.meCache) {
            this.meCache = await this.getMe();
        }
        return this.meCache;
    }

    // Users
    async getUser(userId) {
        return this.performFetch(`/users/${userId}`, "GET");
    }

    async getUserCached(userId) {
        if (!this.userCache[userId]) {
            this.userCache[userId] = await this.getUser(userId);
        }
        return this.userCache[userId];
    }

    async getUserTeams(userId) {
        return this.performFetch(`/users/${userId}/teams`, "GET");
    }

    async getUserByUsername(username) {
        return this.performFetch(`/users/username/${username}`, "GET");
    }

    async getUserByUsernameCached(username) {
        if (!this.userNameCache[username]) {
            this.userNameCache[username] = await this.getUserByUsername(username);
        }
        return this.userNameCache[username];
    }

    _serialize(obj) {
        var str = [];
        for (var p in obj) {
            if (obj.hasOwnProperty(p)) {
                str.push(encodeURIComponent(p) + "=" + encodeURIComponent(obj[p]));
            }
        }
        return str.join("&");
    }

    async getUsers(options) {
        return this.performFetch(`/users?${this._serialize(options)}`, "GET");
    }

    async createUserAccessToken(userId, description) {
        return this.performFetch(`/users/${userId}/tokens`, "POST", {
            description
        });
    }

    async getUserAccessTokens(userId) {
        return this.performFetch(`/users/${userId}/tokens`, "GET");
    }

    async revokeUserAccessToken(userId, tokenId) {
        return this.performFetch(`/users/${userId}/tokens/revoke`, "POST", {
            token_id: tokenId
        });
    }

    // Teams
    async getTeam(teamId) {
        return this.performFetch(`/teams/${teamId}`, "GET");
    }

    async getTeamByName(name) {
        return this.performFetch(`/teams/name/${name}`, "GET");
    }

    async addUserToTeam(userId, teamId) {
        return this.performFetch(`/teams/${teamId}/members`, "POST", {
            team_id: teamId,
            user_id: userId
        });
    }

    // Bots
    async getBots() {
        return this.performFetch(`/bots`, "GET");
    }

    async createBot(bot) {
        return this.performFetch(`/bots`, "POST", bot);
    }

    async updateBot(bot) {
        return this.performFetch(`/bots/${bot.user_id}`, "PUT", bot);
    }

    // Channels

    async getPrivateChannels(teamId) {
        return this.performFetch(`/teams/${teamId}/channels/private`, "GET");
    }

    async getChannelByName(teamId, name) {
        return this.performFetch(`/teams/${teamId}/channels/name/${name}`, "GET")
    }

    async getChannelByNameCached(teamId, name) {
        let key = `${teamId}/${name}`;
        if (!this.channelNameCache[key]) {
            this.channelCache[key] = await this.getChannelByName(teamId, name);
        }
        return this.channelCache[key];
    }

    async getChannel(channelId) {
        return this.performFetch(`/channels/${channelId}`, "GET")
    }

    async getChannelCached(channelId) {
        if (!this.channelCache[channelId]) {
            this.channelCache[channelId] = await this.getChannel(channelId);
        }
        return this.channelCache[channelId];
    }

    async createChannel(channel) {
        return this.performFetch("/channels", "POST", channel);
    }

    async createDirectChannel(userId1, userId2) {
        return this.performFetch(`/channels/direct`, "POST", [userId1, userId2]);
    }

    async updateChannel(channel) {
        return this.performFetch(`/channels/${channel.id}`, "PUT", channel);
    }

    async addUserToChannel(channelId, userId) {
        return this.performFetch(`/channels/${channelId}/members`, "POST", {
            user_id: userId
        });
    }

    // Posts
    async createPost(post) {
        return this.performFetch("/posts", "POST", post);
    }

    async updatePost(post) {
        return this.performFetch(`/posts/${post.id}`, "PUT", post);
    }

    async deletePost(post) {
        return this.performFetch(`/posts/${post.id}`, "DELETE");
    }

    async getPost(postId) {
        return this.performFetch(`/posts/${postId}`, "GET");
    }

    async getThread(postId) {
        return this.performFetch(`/posts/${postId}/thread`, "GET");
    }

    // Reactions
    async addReaction(userId, postId, emoji) {
        return this.performFetch(`/reactions`, "POST", {
            user_id: userId,
            post_id: postId,
            emoji_name: emoji
        });
    }

    async removeReaction(userId, postId, emoji) {
        return this.performFetch(`/users/${userId}/posts/${postId}/reactions/${emoji}`, "DELETE");
    }
}

let _seenEvents = {};
import {Sha256} from "https://deno.land/std/hash/sha256.ts";

export function seenEvent(event) {
    delete event.seq;
    const hash = new Sha256().update(JSON.stringify(event)).hex();
    if (_seenEvents[hash]) {
        return true;
    }
    _seenEvents[hash] = true;
    return false;
}
```