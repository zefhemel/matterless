# Github stalker

"Stalks" (monitors some activity events) a set of github usernames, and posts those updates to a specific Mattermost
channel.

# import

* ../lib/mattermost.md
* ../lib/cron.md

# config

```yaml
config:mm-url:
  type: string
  description: URL to Mattermost instance
config:mm-channel-id:
  type: string
  description: ID of channel to post updates to
config:mm-bot-token:
  type: string
  description: Token for bot user to use
config:usernames:
  type: string
  description: Comma separated list of github usernames to stalk
config:org:
  type: string
  description: Github organization name to filter on
```

# cron MyCron

```
schedule: "0 */5 * * * *"
function: Poll
```

# events

```yaml
action:
  - Event
```

# function Event

```yaml
init:
  bot_token: ${config:mm-bot-token}
  url: ${config:mm-url}
  channel_id: ${config:mm-channel-id}
```

```javascript
import {Mattermost} from "./mattermost_client.js";

let mmClient, channelId;

function init(cfg) {
    mmClient = new Mattermost(cfg.url, cfg.bot_token);
    channelId = cfg.channel_id;
}

async function handle(event) {
    console.log("Got event", event);
    await mmClient.createPost({
        channel_id: channelId,
        message: `${event.username} in [${event.repo}](https://github.com/${event.repo}): ${event.description}`
    });
}
```

# function Poll

```javascript
import {store, events} from "./matterless.ts";

async function handle() {
    let allUsers = await store.get('config:usernames');
    for (let username of allUsers.split(',')) {
        console.log("Checking", username)
        let eventList = await stalkUser('mattermost', username);
        let prevLastId = await store.get(`lastId:${username}`);
        for (let event of eventList) {
            if (event.id === prevLastId) {
                // We saw this one earlier, break
                break;
            }
            await events.publish(`action`, event);
        }
        // Persist last seen id
        if (eventList.length > 0) {
            await store.put(`lastId:${username}`, eventList[0].id);
        }
    }
}

async function stalkUser(orgName, username) {
    let events = await (await fetch(`https://api.github.com/users/${username}/events`)).json();
    let interestingEvents = [];
    for (let event of events) {
        if (!event.repo.name.startsWith(`${orgName}/`)) {
            continue;
        }
        switch (event.type) {
            case 'PullRequestReviewCommentEvent': {
                let repo = event.repo.name;
                // console.log(event, event.payload.pull_request)
                let {html_url, title, user: {login}} = event.payload.pull_request;
                interestingEvents.push(eventObj(event, `Commented on PR ['${title}'](${html_url}) by ${login}`));
            }
                break;
            case 'PullRequestEvent': {
                let action = event.payload.action;
                let {url, title, user: {login}} = event.payload.pull_request;
                interestingEvents.push(eventObj(event, `Pull request ${action} ['${title}'](${url}) by ${login}`));
            }
                break;
            case 'PushEvent': {
                // console.log("Push event", event);
                let {ref, size} = event.payload;
                let refParts = ref.split('/');
                ref = refParts[refParts.length - 1];
                interestingEvents.push(eventObj(event, `${size} commits pushed to branch ${ref}`));
            }
                break;
            case 'IssueCommentEvent': {
                if (event.payload.action !== 'created') {
                    continue;
                }
                // console.log("Issue comment", event.payload);
                let {issue: {html_url, title}} = event.payload;
                interestingEvents.push(eventObj(event, `Commented on issue ['${title}'](${html_url})`));
            }
                break;
        }

        // console.log(event)
    }
    return interestingEvents;
}

function eventObj(event, description) {
    return {
        id: event.id,
        username: event.actor.login,
        date: event.created_at,
        repo: event.repo.name,
        description: description
    };
}
```