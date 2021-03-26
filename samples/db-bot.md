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



## Bot: DatabaseBot
Defines the bot, here it is hardcoded to join the "Dev" team, please update that value to your particularly team you want to enable it for.
```yaml
team_names:
  - Office
username: db-bot
display_name: Database bot
description: My database bot
events:
  posted:
    - HandleCommand
  post_edited:
    - HandleCommand
```
----
## Function: HandleCommand
Implements the actual logic for the commands.

```javascript
import {Store, Mattermost} from "matterless";
import YAML from "yaml";

let client = new Mattermost(process.env.DATABASEBOT_URL, process.env.DATABASEBOT_TOKEN);

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
    let store = new Store();
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

