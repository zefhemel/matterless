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
import {buildAppName, putApp, deleteApp} from "./matterless_root.js";

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
    console.log("Got event", event);

    let code = post.message;
    let appName = buildAppName(post.user_id, post.id);
    if (event.event === 'post_deleted') {
        console.log("Deleting app:", appName);
        await deleteApp(rootToken, appName);
    } else {
        console.log("Updating app:", appName, " with code ", code);
        await putApp(rootToken, appName, code);
    }
}
```

# job ListenToLogs

```yaml
init:
  url: ${config:url}
  root_token: ${config:root_token}
  bot_token: ${token:MatterlessBot}
  team: ${config:team}
```

```javascript
import {listApps, appEventSocket} from "./matterless_root.js";
import {Mattermost} from "https://raw.githubusercontent.com/zefhemel/matterless/master/lib/mattermost_client.js";

let sockets = [];
let rootToken;
let mmClient;
let team;
let url;

async function init(cfg) {
    if (!cfg.url) {
        return
    }
    url = cfg.url;
    rootToken = cfg.root_token;
    mmClient = new Mattermost(cfg.url, cfg.bot_token);
    team = await mmClient.getTeamByName(cfg.team);
}


async function run() {
    subscribeAll();
    let ws = appEventSocket(rootToken, 'matterless-bot');
    console.log("Subscribing to app update events");
    try {
        ws.addEventListener('open', function () {
            ws.send(JSON.stringify({pattern: 'event'}));
        });
        ws.addEventListener('message', function (event) {
            let parsed = JSON.parse(event.data);
            if (parsed.name === 'event') {
                console.log(parsed)
            }
        });
    } catch (e) {
        console.error(e);
    }
}

function unsubscribeAll() {
    for (let socket of sockets) {
        socket.close();
    }
    sockets = [];
}

async function subscribeAll() {
    unsubscribeAll();
    for (let appName of await listApps(rootToken)) {
        console.log("Evaluating app", appName)
        if (appName.startsWith('mls_')) {
            let ws = appEventSocket(rootToken, appName);
            console.log("Subscribed to logs for an app");
            try {
                ws.addEventListener('open', function () {
                    ws.send(JSON.stringify({pattern: 'function.*.log'}));
                });
                ws.addEventListener('message', function (event) {
                    let logJSON = JSON.parse(event.data);
                    // let eventNameParts = ;
                    // console.log("Event parts", JSON.stringify(eventNameParts))
                    let [_, userId, appId] = appName.split('_');
                    let functionName = logJSON.name.split('.')[1];
                    publishLog(userId, appId, functionName, logJSON.data.message);
                });
            } catch (e) {
                console.error(e);
            }
            sockets.push(ws);
        }
    }
}


async function publishLog(userId, appId, functionName, message) {
    console.log(`${userId}: ${appId} [${functionName}] ${message}`);
    let channelName = `matterless-logs-${appId}`.toLowerCase();
    let channelInfo;
    try {
        channelInfo = await mmClient.getChannelByName(team.id, channelName);
    } catch (e) {
        // Doesn't exist yet, let's create
        channelInfo = await mmClient.createChannel({
            team_id: team.id,
            name: channelName,
            display_name: `Matterless : Logs : ${appId}`,
            header: `Logs for matterless application [${appId}](${url}/${team.name}/pl/${appId})`,
            type: 'P'
        })
    }
    await mmClient.addUserToChannel(channelInfo.id, userId);
    await mmClient.createPost({
        channel_id: channelInfo.id,
        message: "From `" + functionName + "`:\n```\n" + message + "\n```"
    });
}
```

# library matterless_root.js

```javascript
const rootUrl = Deno.env.get("API_URL").split('/').slice(0, -1).join('/');

export function buildAppName(userId, postId) {
    return `mls_${userId}_${postId}`;
}

export async function adminCall(rootToken, path, method, body) {
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

export async function listApps(rootToken) {
    return JSON.parse(await adminCall(rootToken, "", "GET"));
}

export async function putApp(rootToken, name, code) {
    return adminCall(rootToken, `/${name}`, "PUT", code);
}

export async function deleteApp(rootToken, name) {
    return adminCall(`/${name}`, "DELETE");
}

export function appEventSocket(rootToken, appName) {
    console.log("Now going to subscribe: ", `${rootUrl}/${appName}/_events`)
    let ws = new WebSocket(`${rootUrl.replace('http://', 'ws://')}/${appName}/_events`);
    ws.addEventListener('error', function (ev) {
        console.error("Got websocket error", ev);
    });
    return ws;
}
```

