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
function handle(event) {
  console.log("Hello world 2!"); 
}
```
----
# Function: FailFunction

```JavaScript
function handle(event) {
  console.
}
```