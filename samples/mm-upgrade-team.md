# events
```yaml
daily:upgrade:
  - Upgraded
daily:upgrade-with-commits:
  - NotifyChannel
```

# function NotifyChannel
```yaml
init:
  url: https://community.mattermost.com
  token: ${config:bot.token}
```

```javascript
import {Mattermost} from "https://raw.githubusercontent.com/zefhemel/matterless/master/lib/mattermost_client.js";

let client;

function init(config) {
    client = new Mattermost(config.url, config.token);
}

function handle(commits) {
    console.log(commits);
}
```

# function Upgraded
```yaml
init:
  token: ${config:gh.token}
  repo: mattermost/mattermost-server
  interesting_users:
    - jespino
    - ashishbhate
    - calebroseland
    - reflog
    - zefhemel
```

```javascript
import {Octokit} from "https://cdn.skypack.dev/@octokit/rest";

let octokit, config;

function init(cfg) {
    config = cfg;
    octokit = new Octokit({
        auth: config.token
    });
}

async function handle(event) {
    let [owner, repo] = config.repo.split('/');
    let fromCommit = await octokit.rest.git.getCommit({
        owner: owner,
        repo: repo,
        commit_sha: event.oldVersion
    });

    let interestingCommits = [];
    let page = 0;

    let payingAttention = false;
    commitListLoop:
    while (true) {
        let commits = await octokit.rest.repos.listCommits({
            owner: owner,
            repo: repo,
            per_page: 100,
            page: page
        });
        // commits are in reverse chronological order, let's first wait for the newest
        for (let commit of commits.data) {
            if (commit.sha === event.newVersion) {
                payingAttention = true;
            }
            if (commit.sha === event.oldVersion) {
                break commitListLoop;
            }
            if (!payingAttention) {
                continue;
            }

            if (config.interesting_users.indexOf(commit.author.login) !== -1) {
                interestingCommits.push(commit);
            }
        }
        page++;
    }
    
    // console.log("Interesting commits", page, "count", interestingCommits.length);
    await events.publish("daily:upgrade-with-commits", interestingCommits.map(commit => ({
        author: commit.author.login,
        message: cleanMessage(commit.commit.message)
    })));
}

function cleanMessage(msg) {
    return msg.split("\n")[0];
}
```

## function CheckUpgrade

```yaml
init:
  url: https://community-daily.mattermost.com
  ns: daily
```

```javascript
import {store, events} from "./matterless.ts";

let config;

function init(cfg) {
    config = cfg;
}

async function handle() {
    let result = await fetch(`${config.url}/api/v4/config/client?format=old`);
    let json = await result.json();
    let featureFlags = {};
    for (const [key, value] of Object.entries(json)) {
        if (key.indexOf("FeatureFlag") === 0) {
            const flagName = key.substring("FeatureFlag".length);
            let previousValue = await store.get(`${config.ns}:flag:${flagName}`);
            if (previousValue !== json[key]) {
                await events.publish(`${config.ns}:flag:${flagName}`, {
                    flag: flagName,
                    oldValue: previousValue,
                    newValue: value
                });
                await store.put(`${config.ns}:flag:${flagName}`, value);
            }
        }
    }
    let oldVersion = (await store.get(`${config.ns}:version`)) || "343c51830f8c877dde1dc2d2c71a50783b81dc7f";
    let version = json.BuildHash;
    if (oldVersion != version) {
        await events.publish(`${config.ns}:upgrade`, {
            oldVersion: oldVersion,
            newVersion: version
        });
        await store.put(`${config.ns}:version`, version);
    }
}
```
