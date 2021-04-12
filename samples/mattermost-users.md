# Mattermost new user registration watcher

# import
* https://raw.githubusercontent.com/zefhemel/matterless/master/lib/cron.md

# events
```
newuser:
- DebugLogger
```

# cron NewUserCheckCron
```yaml
NewUserCron:
  schedule: "*/10 * * * * *"
  function: MattermostWatcher
```

# function DebugLogger
```javascript
function handle(newUser) {
    console.log("New user found!", newUser);
}
```

# function MattermostWatcher
```yaml
init:
  url: ${config:url}
  token: ${config:admin_token}
  event: newuser
```

```javascript
import {store, events} from "./matterless.ts";
let client, eventName;

const lastSeenStoreKey = "mmwatcher_lastSeenNewest";

function init(config) {
    if(!config.url || !config.token) {
        return console.error("url and token not yet configured");
    }
    client = new Mattermost(config.url, config.token);
    eventName = config.event;
}

async function handle() {
    if(!client) return;
    let team = await client.getTeamByName("Matterless");
    let allUsers = await client.getUsers({
        sort: "create_at",
        in_team: team.id
    });
    let lastCreatedData = (await store.get(lastSeenStoreKey)) || 0;
    for(let user of allUsers) {
        if(user.create_at > lastCreatedData) {
            // New!
            await events.publish(eventName, user);
        }
    }
    if(allUsers.length > 0) {
        // Store last seen created date in store
        await store.put(lastSeenStoreKey, allUsers[0].create_at);
    }
}
```