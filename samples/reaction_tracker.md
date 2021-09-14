# Mattermost Reaction Tracker

Tracks all reactions added or removed for a specific user, and notifies the user about this via a bot account.

Required store values for this to work:

* `config:url`: Mattermost URL
* `config:meToken`: Personal access token for the user to track reactions for (you)
* `config:botToken`: Personal access token for bot user to send you messages as (create a separate bot account for this)

# import

* ../lib/mattermost.md

# mattermostListener TrackerClient

```yaml
url: ${config:url}
token: ${config:meToken}
events:
  reaction_added:
    - ReactionChange
  reaction_removed:
    - ReactionChange
```

# function ReactionChange

```javascript
import {store, events} from "./matterless.ts";
import {Mattermost} from "https://raw.githubusercontent.com/zefhemel/matterless/master/lib/mattermost_client.js";

let botClient, meClient, directChannel, me, bot;

async function init() {
    // Fetch credentials from store and setup client
    let url = await store.get("config:url"),
        meToken = await store.get("config:meToken"),
        botToken = await store.get("config:botToken");

    // Create Mattermost clients
    botClient = new Mattermost(url, botToken);
    meClient = new Mattermost(url, meToken);

    // Cache often used values
    bot = await botClient.getMeCached();
    me = await meClient.getMeCached();
    directChannel = await botClient.createDirectChannel(bot.id, me.id);
}

async function handle(event) {
    console.log("Event", event);

    let reaction = JSON.parse(event.data.reaction);
    let reacter = await meClient.getUserCached(reaction.user_id);
    let post = await meClient.getPost(reaction.post_id);
    if (post.user_id !== me.id) {
        return;
    }

    await botClient.createPost({
        channel_id: directChannel.id,
        message: `${event.event} :${reaction.emoji_name}: by @${reacter.username} to your post "${post.message}"`
    });
}
```

