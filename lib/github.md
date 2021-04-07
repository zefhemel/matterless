# Function: PollGithubRepoEvents
[List of github event types](https://docs.github.com/en/developers/webhooks-and-events/github-event-types)

```yaml
init:
  token: ${config:github.token}
  repo: mattermost/mattermost-server
  event_prefix: mm-server
  events:
    - PushEvent
    - PullRequestEvent 
```

```javascript
import { Octokit } from "https://cdn.skypack.dev/@octokit/rest";
import { Store, publishEvent } from "./matterless.ts";

let octokit, config;
let store = new Store();

function init(cfg) {
    octokit = new Octokit({
        auth: cfg.token
    });
    config = cfg;
}

async function handle() {
    let [org, repo] = config.repo.split('/');
    let results = await octokit.rest.activity.listRepoEvents({
        owner: org,
        repo: repo,
        per_page: 100
    });
    
    let lastSeenEvent = await store.get("github:lastSeenEvent"); 
    
    if(results.status === 200) {
        let newEvents = 0;
        for(let event of results.data) {
            if(event.id === lastSeenEvent) {
                break;
            }
            if(config.events.indexOf(event.type) !== -1) {
                await publishEvent(`${config.event_prefix}:${event.type}`, event);
            }
            newEvents++;
        }
        if(results.data.length > 0) {
            // await store.put("github:lastSeenEvent", results.data[0].id);
        }
        return {
            newEvents
        }
    }
}
```

# Events
```yaml
"mm-server:PushEvent":
  - NewInterestingEvent
#"mm-server:PullRequestEvent":
#  - NewInterestingEvent
```

# Function: NewInterestingEvent

```javascript
function handle(event) {
    console.log("Got interesting", event.payload.ref);
    if(event.payload.ref === "refs/heads/master") {
        console.log("Master commit", event)
    }
}
```


