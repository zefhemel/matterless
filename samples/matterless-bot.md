# Matterless bot

To configure this DB bot you need to set the following configuration variables in the store:

* `config:url`: URL to your Mattermost installation
* `config:admin_token`: Personal access token for an admin user, enabling this app to create the necessary "
  matterless-bot" bot.
* `config:team`: Name of the team for the bot to join
* `config:root_token`: Matterless root token
* `config:allowed_users`: list of usernames who are allowed to create matterless apps

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
    - HandleConsoleCommand
  post_edited:
    - HandleCommand
    - HandleConsoleCommand
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
import {ensureConsoleChannel, postLink} from "./util.js";

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

    let post = JSON.parse(event.data.post);
    let me = await mmClient.getMeCached();

    // Lookup channel
    let channel = await mmClient.getChannelCached(post.channel_id);
    // Ignore bot posts
    if (post.user_id === me.id) return;
    // Skip any message outside a private chat with the bot
    if (channel.type != 'D') return;
    // Deduplicate events (bug in editing posts in Mattermost)
    if (seenEvent(event)) {
        return;
    }

    // Check permissions
    let allowedUserNames = (await store.get('config:allowed_users')) || [];
    let allowedUserIds = {};

    for (let username of allowedUserNames) {
        let user = await mmClient.getUserByUsernameCached(username);
        allowedUserIds[user.id] = true;
    }
    if (!allowedUserIds[post.user_id]) {
        await ensureMyReply(me, post, "You're not on the allowed list :thumbsdown:");
        return;
    }
    let appName = buildAppName(post.user_id, post.id);
    let code = post.message;
    let userId = post.user_id;
    if (event.event === 'post_deleted') {
        console.log("Deleting app:", appName);
        await deleteApp(rootToken, appName);
    } else {
        console.log("Updating app:", appName);

        // Attempt to extract an appname
        let friendlyAppName = post.id;
        let matchGroups = /#\s*([A-Z][^\n]+)/.exec(code);
        if (matchGroups) {
            friendlyAppName = matchGroups[1];
        }
        let channelInfo = await ensureConsoleChannel(mmClient, team.id, post.id, friendlyAppName);
        channelInfo.display_name = `Matterless : ${friendlyAppName}`;
        channelInfo.header = `Console for your matterless app [${friendlyAppName}](${postLink(url, team, post.id)})`;
        await mmClient.updateChannel(channelInfo);

        await mmClient.addUserToChannel(channelInfo.id, userId);
        try {
            let result = await putApp(rootToken, appName, code);
            await mmClient.addReaction(me.id, post.id, "white_check_mark");
            await mmClient.removeReaction(me.id, post.id, "octagonal_sign");
            await ensureMyReply(me, post, 'All good to go :thumbsup:');
        } catch (e) {
            await mmClient.addReaction(me.id, post.id, "octagonal_sign");
            await mmClient.removeReaction(me.id, post.id, "white_check_mark");
            await ensureMyReply(me, post, `ERROR: ${e.message}`);
        }
    }
}

async function ensureMyReply(me, post, message) {
    let thread = await mmClient.getThread(post.id);
    let replyPost;
    for (let postId of thread.order) {
        let threadPost = thread.posts[postId];
        if (threadPost.user_id == me.id) {
            replyPost = threadPost;
        }
    }
    if (replyPost) {
        replyPost.message = message;
        await mmClient.updatePost(replyPost);
    } else {
        await mmClient.createPost({
            channel_id: post.channel_id,
            root_id: post.id,
            message: message
        });
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
import {ensureConsoleChannel} from "./util.js";

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
            let [_, appId] = logJSON.app.split('_');
            let functionName = logJSON.name.split('.')[0];
            publishLog(appId, functionName, logJSON.data.message);
        });
    } catch (e) {
        console.error(e);
    }
}

async function publishLog(appId, functionName, message) {
    // console.log(`${userId}: ${appId} [${functionName}] ${message}`);
    let channelName = `matterless-console-${appId}`.toLowerCase();
    let channelInfo = await mmClient.getChannelByName(team.id, channelName);
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
    return `mls_${postId}`;
}

export async function adminCall(rootToken, path, method, body) {
    let result = await fetch(`${rootUrl}${path}`, {
        method: method,
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `bearer ${rootToken}`
        },
        body: body
    });
    let textResult = await result.text();
    if (result.status > 300 || result.status < 200) {
        throw Error(textResult);
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

export function appApiUrl(appName) {
    return `${rootUrl}/${appName}`;
}
```

# library util.js

```javascript
export async function ensureConsoleChannel(mmClient, teamId, appId, appName) {
    let channelName = `matterless-console-${appId}`.toLowerCase();
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

# function HandleConsoleCommand

Implements the actual logic for the commands invoked whenever the bot is sent a message in a direct channel.

```yaml
init:
  url: ${config:url}
  token: ${token:MatterlessBot}
  root_token: ${config:root_token}
```

```javascript
import {appApiUrl} from "./matterless_root.js";
import {API} from "./matterless.ts";
import {Mattermost} from "./mattermost_client.js";

let mmClient;
let rootToken;

function init(cfg) {
    if (!cfg.url || !cfg.token) {
        console.error("URL and token not initialized yet");
        return;
    }
    mmClient = new Mattermost(cfg.url, cfg.token);
    rootToken = cfg.root_token;
}

// Main event handler
async function handle(event) {
    let post = JSON.parse(event.data.post);
    let me = await mmClient.getMeCached();

    // Lookup channel
    let channel = await mmClient.getChannelCached(post.channel_id);
    // Ignore bot posts
    if (post.user_id === me.id) return;
    // Ignore any non-console commands
    if (!channel.name.startsWith('matterless-console-')) {
        return;
    }

    let appName = channel.name.substring('matterless-console-'.length);
    let api = new API(appApiUrl(`mls_${appName}`), rootToken);
    let store = api.getStore();

    let words = post.message.split(' ');
    let key, result, val, prop, eventName, functionName, jsonData;
    switch (words[0]) {
        case "get":
            key = words[1];
            result = await store.get(key);
            await mmClient.createPost({
                channel_id: post.channel_id,
                root_id: post.id,
                parent_id: post.id,
                message: `${result}`
            });
            break;
        case "put":
            key = words[1];
            val = words.slice(2).join(' ');
            await store.put(key, val);
            await mmClient.addReaction(me.id, post.id, "white_check_mark");
            break;
        case "del":
            key = words[1];
            await store.del(key);
            await mmClient.addReaction(me.id, post.id, "white_check_mark");
            break;
        case "keys":
            result = (await store.queryPrefix(words[1] || "")) || [];
            await mmClient.createPost({
                channel_id: post.channel_id,
                root_id: post.id,
                parent_id: post.id,
                message: `- ${result.map(([k, v]) => k).join("\n- ")}`
            });
            break;
        case "all":
            result = (await store.queryPrefix("")) || [];
            await mmClient.createPost({
                channel_id: post.channel_id,
                root_id: post.id,
                parent_id: post.id,
                message: `- ${result.map(([k, v]) => `${k}: ${v}`).join("\n- ")}`
            });
            break;
        case "trigger":
            eventName = words[1];
            jsonData = words.slice(2).join(' ');
            await api.publishEvent(eventName, jsonData ? JSON.parse(jsonData) : {});
            await mmClient.addReaction(me.id, post.id, "white_check_mark");
            break;
        case "invoke":
            functionName = words[1];
            jsonData = words.slice(2).join(' ');
            let result = await api.invokeFunction(functionName, jsonData ? JSON.parse(jsonData) : {});
            await mmClient.createPost({
                channel_id: post.channel_id,
                root_id: post.id,
                parent_id: post.id,
                message: `${JSON.stringify(result)}`
            });
            break;
        default:
            await mmClient.createPost({
                channel_id: post.channel_id,
                root_id: post.id,
                parent_id: post.id,
                message: "Not sure what you're saying. I only understand the commands `all`, `keys`, `put`, `get`, and `del`, so perhaps try one of those."
            });
    }
}
```