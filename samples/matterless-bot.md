# Matterless bot

To configure this DB bot you need to set the following configuration variables in the store:

* `config:url`: URL to your Mattermost installation
* `config:admin_token`: Personal access token for an admin user, enabling this app to create the necessary "
  matterless-bot" bot.
* `config:team`: Name of the team for the bot to join

# import

* ../lib/mattermost.md

## mattermostBot MatterlessBot

Defines the bot using the `mattermostBot` macro.

```yaml
username: matterless
display_name: Matterless
description: My matterless bot
teams:
  - ${config:team}
url: ${config:url}
admin_token: ${config:admin_token}
events:
  posted:
    - HandleCommand
  post_edited:
    - HandleCommand
  post_deleted:
    - HandleCommand
```

## function HandleCommand

Implements the actual logic for the commands invoked whenever the bot is sent a message in a direct channel.

```yaml
prewarm: true
init:
  url: ${config:url}
  token: ${token:MatterlessBot}
  root_token: ${config:root_token}
```

```javascript
import {store} from "./matterless.ts";
import {Mattermost} from "https://raw.githubusercontent.com/zefhemel/matterless/master/lib/mattermost_client.js";

let client;
let rootToken;

async function init(cfg) {
    console.log("Here is the cfg", cfg)
    if (!cfg.url || !cfg.token) {
        console.error("URL and token not initialized yet");
        return;
    }
    client = new Mattermost(cfg.url, cfg.token);
    rootToken = cfg.root_token;
}

// Main event handler
async function handle(event) {
    console.log("Got event", event);
    if (!client) {
        console.log("Not inited yet");
    }
    let post = JSON.parse(event.data.post);
    let me = await client.getMeCached();

    // Lookup channel
    let channel = await client.getChannelCached(post.channel_id);
    // Ignore bot posts
    if (post.user_id === me.id) return;
    // Skip any message outside a private chat
    if (channel.type != 'D') return;

    let code = post.message;
    let appName = `${post.user_id}_${post.id}`;
    if (event.event === 'post_deleted') {
        console.log("Deleting app:", appName);
        await deleteApp(appName);
    } else {
        console.log("Updating app:", appName, " with code ", code);
        await putApp(appName, code);
    }
}

async function adminCall(path, method, body) {
    let rootUrl = Deno.env.get("API_URL").split('/').slice(0, -1).join('/');
    let result = await fetch(`${rootUrl}${path}`, {
        method: method,
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `bearer ${rootToken}`
        },
        body: body
    })
    let textResult = (await result.text());
    if (textResult.status === "error") {
        throw Error(textResult.error);
    }
    return textResult;
}

async function listApps() {
    return adminCall("", "GET");
}

async function putApp(name, code) {
    return adminCall(`/${name}`, "PUT", code);
}

async function deleteApp(name) {
    return adminCall(`/${name}`, "DELETE");
}
```

# job ListenToLogs

```yaml
init:
  url: ${config:url}
  root_token: ${config:root_token}
```

```javascript
let sockets = [];
let rootToken;

function init(cfg) {
    if (!cfg.url) {
        return
    }
    rootToken = cfg.root_token;
    // ws = new WebSocket(cfg.url);
    //
    // ws.addEventListener("error", function (ev) {
    //     console.error("WS error", ev);
    // })
}
```