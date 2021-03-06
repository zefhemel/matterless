# Function: TestFunction1

```javascript
export default function(event) {
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
export default function(event) {
  console.log("Hello world 2!"); 
}
```
----
# Function: FailFunction

```JavaScript
export default function(event) {
  console.
}
```