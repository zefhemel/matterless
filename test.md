# Import
* file:lib/mattermost.md
* file:lib/cron.md

Or URL: https://raw.githubusercontent.com/zefhemel/matterless/reset/lib/mattermost.md

# MattermostListener: MyClient
```yaml
url: ${config:url}
token: ${config:token}
events:
  hello:
    - MyCustomFunc
  posted:
    - MyCustomFunc
```

# Function: MyCustomFunc
```javascript
import {invokeFunction} from "./matterless.ts";

async function handle(event) {
    console.log("Got custom event", event);
    return {
        secretMessag: "Yo" + await invokeFunction("MyAuxFunction")
    };
}
```

# Function: MyAuxFunction

```javascript
function handle() {
    return {number: 22};
}
```

