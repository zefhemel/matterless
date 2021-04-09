# Application Database Bot
This example uses the Mattermost store API to implement a CLI-style database application.
The Matterless store API is a simple key-value store, so operations are limited, but this adds a few niceties.

The application registers a new `@db-bot` bot you can talk to, these are the commands supported:

* `all` — lists all entries in a nice table format
* `keys [prefix]` — lists all keys with a particular prefix
* `put [key] [yaml-data]` — creates/updates a new entry, example (creating a new item with key "zef" with an object with two attributes: name and spouse):
  
      put zef
      name: Zef Hemel
      spouse: Justyna
* `get [key]` — looks up the entry with key `key`
* `del [key]` — deletes entry with key `key` 
* `pput [key] [prop] [yaml-data]` — Sets one particular property of the entry, e.g. `pput zef name Zef Hemel Jr.`
* `pdel [key] [prop]` — deletes a particularly property of the entry with key `key`

That's all!


# import
* file:lib/mattermost.md

## mattermostBot DatabaseBot
Defines the bot, here it is hardcoded to join the "Dev" team, please update that value to your particularly team you want to enable it for.
```yaml
teams:
  - Matterless
username: db-bot
display_name: Database bot
description: My database bot
url: ${config:url}
admin_token: ${config:admin_token}
events:
  posted:
    - HandleCommand
  post_edited:
    - HandleCommand
```

## function HandleCommand
Implements the actual logic for the commands.

```yaml
init:
  url: ${config:url}
  token: ${config:DatabaseBot.token}
```

```javascript
import {store} from "./matterless.ts";
import YAML from "https://cdn.skypack.dev/yaml";
import {Mattermost} from "https://raw.githubusercontent.com/zefhemel/matterless/master/lib/mattermost_client.js";

let client;

function init(cfg) {
    if(!cfg.url || !cfg.token) {
        console.error("URL and token not initialized yet");
        return;
    }
    client = new Mattermost(cfg.url, cfg.token);
}

function jsonToMDTable(jsonArray) {
    let headers = {};
    for(const [key, val] of jsonArray) {
        if(typeof val !== 'object') continue;
        for(const key of Object.keys(val)) {
            headers[key] = true;
        }
    }
    let headerList = Object.keys(headers);
    let lines = [];
    lines.push('|Key|' + headerList.join('|') + '|');
    lines.push('|---|' + headerList.map(title => '----').join('|') + '|');
    for(const [key, val] of jsonArray) {
        if(typeof val !== 'object') continue;
        let el = [];
        for(let prop of headerList) {
            el.push(JSON.stringify(val[prop]));
        }
        lines.push('|' + key + '|' + el.join('|') + '|');
    }
    return lines.join("\n");
}


async function handle(event) {
    console.log("Got event", event);
    if(!client) {
        console.log("Not inited yet");
    }
    let post = JSON.parse(event.data.post);
    let me = await client.getMeCached();
    
    // Lookup channel
    let channel = await client.getChannelCached(post.channel_id);
    // Ignore bot posts
    if(post.user_id === me.id) return;
    // Skip any message outside a private chat
    if(channel.type != 'D') return;
        
    let words = post.message.split(' ');
    let key, result, val, prop;
    switch(words[0]) {
        case "get":
            key = words[1];
            result = await store.get(key);
            await client.createPost({
                channel_id: post.channel_id,
                root_id: post.id,
                parent_id: post.id,
                message: YAML.stringify(result)
            });
            break;
        case "put":
            key = words[1];
            val = words.slice(2).join(' ');
            // console.log("YAML: ", val);
            await store.put(key, YAML.parse(val));
            await client.addReaction(me.id, post.id, "white_check_mark");
            break;
        case "pput":
            key = words[1];
            prop = words[2];
            val = words.slice(3).join(' ');
            result = await store.get(key);
            result[prop] = YAML.parse(val);
            await store.put(key, result);
            await client.addReaction(me.id, post.id, "white_check_mark");
            break;
        case "pdel":
            key = words[1];
            prop = words[2];
            result = await store.get(key);
            delete result[prop];
            result = await store.put(key, result);
            await client.addReaction(me.id, post.id, "white_check_mark");
            break;
        case "pget":
            key = words[1];
            prop = words[2];
            result = await store.get(key);
            await client.createPost({
                channel_id: post.channel_id,
                root_id: post.id,
                parent_id: post.id,
                message: YAML.stringify(result[prop])
            });
            break;
        case "del":
            key = words[1];
            await store.del(key);
            await client.addReaction(me.id, post.id, "white_check_mark");
            break;
        case "keys":
            result = (await store.queryPrefix(words[1] || "")) || [];
            await client.createPost({
                channel_id: post.channel_id,
                root_id: post.id,
                parent_id: post.id,
                message: `- ${result.map(([k, v]) => k).join("\n- ")}`
            });
            break;
        case "all":
            result = (await store.queryPrefix("")) || [];
            await client.createPost({
                channel_id: post.channel_id,
                root_id: post.id,
                parent_id: post.id,
                message: jsonToMDTable(result)
            });
            break;
        default:
            await client.createPost({
                channel_id: post.channel_id,
                root_id: post.id,
                parent_id: post.id,
                message: "Not sure what you're saying. I only understand the commands `all`, `keys`, `put`, `get`, `pput`, `pget`, `del` and `pdel`, so perhaps try one of those."
            });
    }
}
```

