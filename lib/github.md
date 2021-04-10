# Github Repository Watcher
Watches a Github repo for events through polling. Publishes events via Matterless events, prefixed with the watcher name.

Example use:

    # githubRepoEvents Matterless
    
    ```yaml
    repo: zefhemel/matterless
    token: someToken
    poll_schedule: "*/60 * * * *"
    ```
    
    # events
    ```yaml
    Matterless:*:
      - DebugEvent
    ```
    
    # function DebugEvent
    ```javascript
    function handle(event) {
        console.log("Got event", event);
    }
    ```


[List of github event types](https://docs.github.com/en/developers/webhooks-and-events/github-event-types)

# import
We need cron, so let's import it

* https://raw.githubusercontent.com/zefhemel/matterless/master/lib/cron.md

# macro githubRepoEvents
Listens to events for a specific 

```yaml
input_schema:
  type: object
  properties:
    repo:
      type: string
    token:
      type: string
    poll_schedule:
      type: string
  required:
  - repo
  - token
```

And the template itself:

    # function {{$name}}PollGithubEvents
    
    ```yaml
    init:
      token: {{$input.token}}
      repo: {{$input.repo}}
      event_prefix: {{$name}}
    ```
    
    ```javascript
    import { Octokit } from "https://cdn.skypack.dev/@octokit/rest";
    import { store, events } from "./matterless.ts";
    
    let octokit, config;
    
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
                await events.publish(`${config.event_prefix}:${event.type}`, event);
                newEvents++;
            }
            if(results.data.length > 0) {
                await store.put("github:lastSeenEvent", results.data[0].id);
            }
            return {
                newEvents
            }
        }
    }
    ```

    {{if $input.poll_schedule}}

    # cron {{$name}}GithubEventCron
    ```
    {{$name}}GithubCron:
      schedule: {{yaml $input.poll_schedule}}
      function: {{$name}}PollGithubEvents
    ```
    {{end}}

