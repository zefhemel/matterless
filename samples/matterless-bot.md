# Matterless bot

To configure this DB bot you need to set the following configuration variables in the store:

* `config:url`: URL to your Mattermost installation
* `config:admin_token`: Personal access token for an admin user, enabling this app to create the necessary "
  matterless-bot" bot.
* `config:team`: Name of the team for the bot to join
* `config:root_token`: Matterless root token

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
hot: true

init:
  url: ${config:url}
  token: ${token:MatterlessBot}
  root_token: ${config:root_token}
  team: ${config:team}
```

```javascript
import {store} from "./matterless.ts";
import {Mattermost, seenEvent} from "./mattermost_client.js";
import {buildAppName, putApp, deleteApp} from "./matterless_root.js";
import {ensureLogChannel, postLink} from "./util.js";

let mmClient;
let rootToken;
let team;
let url;

async function init(cfg) {
    console.log("Here is the cfg", cfg)
    if (!cfg.url || !cfg.token) {
        console.error("URL and token not initialized yet");
        return;
    }
    url = cfg.url;
    mmClient = new Mattermost(cfg.url, cfg.token);
    rootToken = cfg.root_token;
    team = await mmClient.getTeamByName(cfg.team);
}

// Main event handler
async function handle(event) {
    if (!mmClient) {
        console.log("Not inited yet");
    }
    // Deduplicate events (bug in editing posts in Mattermost)
    if (seenEvent(event)) {
        return;
    }

    let post = JSON.parse(event.data.post);
    let me = await mmClient.getMeCached();

    // Lookup channel
    let channel = await mmClient.getChannelCached(post.channel_id);
    // Ignore bot posts
    if (post.user_id === me.id) return;
    console.log("Got event", event);
    // Skip any message outside a private chat
    if (channel.type != 'D') return;

    let code = post.message;
    let appName = buildAppName(post.user_id, post.id);
    let userId = post.user_id;
    if (event.event === 'post_deleted') {
        console.log("Deleting app:", appName);
        await deleteApp(rootToken, appName);
    } else {
        console.log("Updating app:", appName);
        // try {
        //     await mmClient.removeReaction(me.id, post.id, "white_check_mark");
        // } catch (e) {
        //     console.log("Remove reaction error", e);
        // }

        // Attempt to extract an appname
        let friendlyAppName = post.id;
        let matchGroups = /#\s*([A-Z][^\n]+)/.exec(code);
        if (matchGroups) {
            friendlyAppName = matchGroups[1];
        }
        let channelInfo = await ensureLogChannel(mmClient, team.id, post.id, friendlyAppName);
        channelInfo.display_name = `
Matterless : Logs : ${friendlyAppName}`;
        channelInfo.header = `
Logs
for your matterless
app [${friendlyAppName}](${postLink(url, team, post.id)})`;
        await mmClient.updateChannel(channelInfo);

        await mmClient.addUserToChannel(channelInfo.id, userId);
        await putApp(rootToken, appName, code);
        // await mmClient.addReaction(me.id, post.id, "white_check_mark");
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
import {listApps, globalEventSocket} from "./matterless_root.js";
import {Mattermost} from "./mattermost_client.js";
import {ensureLogChannel} from "./util.js";

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
    if (!url || !rootToken) {
        return;
    }
    let ws = globalEventSocket(rootToken);
    console.log("Subscribing to all log events");
    try {
        ws.addEventListener('open', function () {
            ws.send(JSON.stringify({pattern: '*.*.log'}));
        });
        ws.addEventListener('message', function (event) {
            let logJSON = JSON.parse(event.data);
            if (!logJSON.app.startsWith('mls_')) {
                return
            }
            // console.log("Got a lot event", logJSON);
            let [_, userId, appId] = logJSON.app.split('_');
            let functionName = logJSON.name.split('.')[0];
            publishLog(userId, appId, functionName, logJSON.data.message);
        });
    } catch (e) {
        console.error(e);
    }
}

async function publishLog(userId, appId, functionName, message) {
    // console.log(`${userId}: ${appId} [${functionName}] ${message}`);
    let channelInfo = await ensureLogChannel(mmClient, team.id, appId);
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
    return adminCall(rootToken, `/${name}`, "DELETE");
}

export async function getAppCode(rootToken, name) {
    return adminCall(rootToken, `/${name}`, "GET");
}

export function appEventSocket(rootToken, appName) {
    console.log("Now going to subscribe: ", `${rootUrl}/${appName}/_events`)
    let ws = new WebSocket(`${rootUrl.replace('http://', 'ws://')}/${appName}/_events`);
    ws.addEventListener('error', function (ev) {
        console.error("Got websocket error", ev);
    });
    return ws;
}

export function globalEventSocket(rootToken) {
    console.log("Now going to subscribe: ", `${rootUrl}/_events`)
    let ws = new WebSocket(`${rootUrl.replace('http://', 'ws://')}/_events`);
    ws.addEventListener('error', function (ev) {
        console.error("Got websocket error", ev);
    });
    return ws;
}
```

# library util.js

```javascript
export async function ensureLogChannel(mmClient, teamId, appId, appName) {
    let channelName = `matterless-logs-${appId}`.toLowerCase();
    let channelInfo;
    try {
        channelInfo = await mmClient.getChannelByNameCached(teamId, channelName);
    } catch (e) {
        // Doesn't exist yet, let's create
        channelInfo = await mmClient.createChannel({
            team_id: teamId,
            name: channelName,
            display_name: `Matterless : Logs : ${appName}`,
            header: `Logs for your matterless app ${appName}`,
            type: 'P'
        })
    }

    return channelInfo;
}

export function postLink(url, team, postId) {
    return `${url}/${team.name}/pl/${postId}`;
} 
```