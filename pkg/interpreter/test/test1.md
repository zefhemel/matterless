# Function: TestFunction1

```javascript
function handler(event) {
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
function handler(event) {
  console.log("Hello world 2!");
}
```