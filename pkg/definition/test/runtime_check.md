# Function: TestFunction1

```javascript
function handle(event) {
  console.log("Hello world!");
  if(JSON.stringify(event) === '{}') {
    // Test event
    return true;
  } else {
    return false;
  }
}
```

------
# Function: TestFunction2

```JavaScript
import {myCall} from "my-module";
function handle(event) {
  return myCall();
}
```
----
# Function: FailFunction

```JavaScript
function handle(event) {
  console.
}
```

---
# Function: MyHTTPTest
```javascript
function handle(event) {
    console.log("HTTP event", event);
    return {
        status: 200,
        body: "Hello world!"
    };
}
```

# Module: my-module
```javascript
export function myCall() {
    return "Sup";
}
```