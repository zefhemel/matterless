# Self-censorship for Mattermost
For the foul-mouthed. Auto self-censors the phrases of your choice.

## Configuration
To configure this, you need to set the following configuration variables in the store:

* `config:url`: URL to your Mattermost installation.
* `config:token`: your personal access token

And then any phrases or words to censor:

* `censor:Slack` with e.g. `S**ck` as value, would replace "Slack" with "S**ck"

That's it. Have fun!

The remainder of this document is the actual Matterless implementation of this awesome concept.

# import
* https://raw.githubusercontent.com/zefhemel/matterless/master/lib/mattermost.md

## mattermostListener MattermostCensorListener
```yaml
url: ${config:url}
token: ${config:token}
events:
  posted:
    - CensorPost
  post_edited:
    - CensorPost
```

## function CensorPost
```yaml
init:
  url: ${config:url}
  token: ${config:token}
```

```javascript
import {Mattermost} from "https://raw.githubusercontent.com/zefhemel/matterless/master/lib/mattermost_client.js";
import {store} from "./matterless.ts";

let client;

function init(cfg) {
    if(!cfg.url || !cfg.token) {
        console.error("URL and token not initialized yet");
        return;
    }
    client = new Mattermost(cfg.url, cfg.token);
}

async function handle(event) {
    console.log("Got event", event);
    if(!client) {
        console.log("Not inited yet");
    }
    let me = await client.getMeCached();
    let post = JSON.parse(event.data.post);
    
    if(post.user_id !== me.id) {
        // Not my post, ignore
        return;
    }
        
    let censoredMessage = post.message;
    for(const [word, censoredWord] of await store.queryPrefix("censor:")) {
        censoredMessage = censoredMessage.replaceAll(word.replace(/^censor:/, ""), censoredWord);
    }
    
    if(post.message !== censoredMessage) {
        post.message = censoredMessage;
        await client.updatePost(post);
    }
    
    console.log("Censored message", censoredMessage);
}
```

