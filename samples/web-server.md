# events

```yaml
http:GET:/:
  - WebFunction
custom-event:
  - MyCustomFunction
store:put:counter:
- MyPutListener
```

# function WebFunction
```javascript
import {store, events} from "./matterless.ts";

async function handle(evt) {
    let counter = (await store.get("counter")) || 0;
    counter++;
    await store.put("counter", counter);
    events.publish("custom-event", {from: "me"});
    return {
        status: 200,
        body: `Hello world! ${counter}`
    };
}
```

# function MyCustomFunction

```javascript
function handle(evt) {
    console.log("Sup", evt);
}
```

# function MyPutListener

```javascript
function handle(evt) {
    console.log("Putting", evt.key, evt.new_value);
}
```