# events

```yaml
http:GET:/:
  - WebFunction
```

# function WebFunction

```yaml
hot: true
instances: 3
```

```javascript
import {store} from "./matterless.ts";

let myNodeId = Math.floor(Math.random() * 10000);

async function handle(evt) {
    let counter = (await store.get("counter")) || 0;
    counter++;
    await store.put("counter", counter);
    return {
        status: 200,
        body: `Hello world! ${counter} my node id is ${myNodeId}`
    };
}
```