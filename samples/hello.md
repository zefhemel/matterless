# Hello world!
This is your Matterless hello world application. To run it:

```shell
$ mls run samples/hello.md
```

This will kick you into the matterless console, to trigger the function type:

```
hello> invoke HelloWorld
```

You should now see something along the lines of

    INFO[0005] [App: hello | Function: HelloWorld] Starting deno function runtime. 
    INFO[0005] [App: hello | Function: HelloWorld] Hello world!

# function HelloWorld
```javascript
function init() {
    // console.aefae();
}
function handle(event) {
    console.log("Hello world!");
}
```