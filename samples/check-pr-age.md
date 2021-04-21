
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

# function CheckMergedPRs
```yaml
init:
  token: ${config:gh.token}
  repo: mattermost/mattermost-server
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

function bucket(prs) {
    const ranges = [
        [0, 4],
        [4, 8],
        [8, 12],
        [12, 24],
        [24, 48],
        [48, 5 * 24],
        [5 * 24, 7 * 24],
        [7 * 24, 14 * 24],
        [14 * 24, 1000 * 24]
    ];
    let bucketCounts = {};
    for(let pr of prs) {
        let createdAt = Date.parse(pr.created_at);
        if(!pr.merged_at) {
            // Closed without merge, skip
            continue;
        }
        let mergedAt = Date.parse(pr.merged_at);
        let hourDiff = (mergedAt - createdAt) / (1000 * 60 * 60);
        for(const range of ranges) {
            if(hourDiff >= range[0] && hourDiff < range[1]) {
                bucketCounts[range] = (bucketCounts[range] || 0) + 1; 
            }
        }
    }
    return bucketCounts;
}

function niceHours(hs) {
    if(hs < 48) {
        return `${hs}h`;
    }
    let days = Math.floor(hs / 24);
    hs -= days * 24;
    if(hs === 0) {
        return `${days}d`;
    }
    return `${days}d${hs}h`;
}

async function handle(event) {
    let [owner, repo] = config.repo.split('/');
    let mergedPRs = await octokit.rest.pulls.list({
        owner: owner,
        repo: repo,
        state: 'closed',
        base: 'master',
        sort: 'created',
        direction: 'desc',
        per_page: 100,
    });

    let buckets = bucket(mergedPRs.data);
    for(let prop of Object.keys(buckets).sort((k1, k2) => {
        let fr1 = k1.split(',')[0];
        let fr2 = k2.split(',')[0];
        return +fr1 - +fr2;
    })) {
        let [fr, to] = prop.split(',');
        console.log(`${niceHours(fr)}-${niceHours(to)}: ${buckets[prop]}`);
    }
}

function timeSince(d1, d2) {

    var seconds = Math.floor((d2 - d1) / 1000);

    var interval = Math.floor(seconds / 31536000);

    if (interval > 1) {
        return interval + " years";
    }
    interval = Math.floor(seconds / 2592000);
    if (interval > 1) {
        return interval + " months";
    }
    interval = Math.floor(seconds / 86400);
    if (interval > 1) {
        return interval + " days";
    }
    interval = Math.floor(seconds / 3600);
    if (interval > 1) {
        return interval + " hours";
    }
    interval = Math.floor(seconds / 60);
    if (interval > 1) {
        return interval + " minutes";
    }
    return Math.floor(seconds) + " seconds";
}
```