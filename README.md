# Matterless: put serverless on your server
_Serverless_ enables you to build applications that automatically scale with demand, and your wallet. Within seconds, serverless application can scale from handling 0 requests per day to thousands of requests per second. This is the power of the cloud at its best.

But what if all you want is check open pull requests on your Github repo at 9am every morning, and send a reminder message on your team’s chat channel? What if you want to create a chat bot that adds a “high five” reaction to any message containing the phrase “high five”? What if all you want to do is blink your smart lights whenever your backup drive runs out of disk space? What if you’re too lazy to create an AWS, Azure or GCP account or scared to connect it to your credit card? What if you have a Raspberry Pi sitting in your closet that’s not doing anything useful? What if you prefer to be in full control of your code, infrastructure and data? What if you’re up for something _different_?

_Matterless_ may be for you.

Matterless brings the powerful serverless programming model to your own server, laptop or even Raspberry Pi. It is simple to use, fast to deploy, and just... fun:

1. Matterless is distributed as a **single binary** with no required dependencies (although to use docker as a runtime, you will need… docker).
2. Matterless requires **zero configuration**  to run (although it does give you options).
3. Matterless is **light-weight**: it runs fine on a Raspberry Pi. There's no fancy container orchestration, kubernetes or firecrackers involved.
4. Matterless enables **extremely rapid iteration**: Matterless applications tend to (re)deploy within seconds. A common mode of develoment is to have Matterless watch for file changes and reload on every file save.

Matterless is not attempting to be a replacement for AWS, Azure or GCP. If you need to scale from 0 to thousands of requests per second, Matterless likely won't cut it.
Matterless' sweetspot is in building scratch-your-own-itch micro applications you may have a need for, but wouldn't require scalability the full cloud provides.

Nevertheless, its programming model is very serverless-esque:

* Use Matterless **functions** to respond to events.
* Use Matterless **events** to glue different parts of your application together.
* Use the Matterless **store API** (a simple key-value store) to store persistent application data.
* (Coming soon) Use Matterless **queues** to schedule work to be performed asynchronously.

In addition, to enable extending Matterless in Matterless (it’s [Matterless all the way down](https://en.wikipedia.org/wiki/Turtles_all_the_way_down)), Matterless adds:

* Matterless **jobs** to write code that runs constantly in the background and connects with external systems, generally exposing anything interesting inside your application as events (e.g. via a (web)socket connection, or polling).
* A **macro system** based on [Go template](https://golang.org/pkg/text/template/) syntax to create new, higher-level definition types (we’ll get to that).

Under the hood, Matterless relies on the following technologies:

1. Matterless is written in [Go](https://golang.org/).
2. Matterless' default runtime is [Deno](https://deno.land). Deno runs JavaScript and Typescript code in a secure sandbox. Therefore functions and jobs don't have access to the local file system and cannot spawn local processes. And don't worry, you don't need to have Deno installed, it will be downloaded automatically for you on first launch.

Sounds interesting? You would be correct. You have good judgement.

## What is a Matterless application
A Matterless application consists of declarative _definitions_ written in a _matterless definition_ file. One matterless definition file defines one application, although you can import other files via URLs. Naturally, matterless definition files use the `.md` file extension. You may think: "Hey, but that’s already used by Markdown!" Conveniently, Matterless' application format **is** markdown with specific semantics, so that all works out well — and it looks great when rendered on Github (and ultimately it's all about what code looks like on Github).

In principle, arbitrary Markdown is allowed in a `.md` file and Matterless will accept it. Documenting your application this way is encouraged. It looks like [literate programming](https://en.wikipedia.org/wiki/Literate_programming) is finally coming to fruition (you’re welcome, Donald).

However, when you use headers (`#` nested at any level) _and_ the first word of the header starts with a lowercase letter (which is a big no in regular writing anyway, capitalize your headers, people!), Matterless interprets it as a Matterless definition.

So. This is the point in the README where it is revealed that the README.md file you're reading right now, is in fact a valid and even somewhat useful Mattermost application! Try it out with `mls run README.md`!

I'll wait until you put together your head, which has just been blown.

## Back to the primitives

Matterless currently supports the following **core primitive definition types**:

* `function` (or `func` if you're lazy): for defining short-running functions that can be triggered e.g. when certain events occur.
* `job`: for defining long-running background processes that for instance connect to external systems, and trigger events as a result.
* `events`: for mapping events to functions to be triggered. There are certain built-in events that will automically trigger under certain conditions (e.g. when writing to the data store, or when certain URLs are called on Matterless’s HTTP Gateway).
* `macro`: for defining new abstractions that map to a combination of existing matterless definitions.
* `imports`: for importing externally defined (addressed via URLs) matterless definitions into your application (often used to import macros).
* Custom definition types previously defined using `macro`s.

## Matterless APIs
Inside of `function` and `job` code (which in the future will be able to use multiple runtimes, but use Deno for now), you have access to a few [Matterless APIs](https://github.com/zefhemel/matterless/blob/master/pkg/sandbox/deno/matterless.ts):

* `store`: a simple key-value store with operations to:
    * `store.put(key, value)` a specific value for a key.
    * `store.get(key)` to fetch the value for a specific key.
    * `store.del(key)` to delete a key from the database.
    * `store.queryPrefix(prefix)` to fetch all keys and their values prefixed with `prefix`.
* `events`: to publish events and respond to them (in an RPC setup):
    * `events.publish(eventName, eventData)` to publish a custom event (that can be listened to via a `event` definition in your definition file).
    * `events.respond(toEvent, eventData)` to respond to a specific event (currently only used to respond to HTTP request events).
* `functions`: invoke other functions by name (rarely needed, but supported)
   * `functions.invoke(functionName, eventData)` invoke function `functionName` with `eventData`.

But any arbitrary deno libraries can be imported as well.

## Matterless 101

Here is "Hello world" in Matterless:

----
    # function HelloWorld
	```javascript
    function handle(event) {
        console.log("Hello world!");
    }
    ```
---- -

Save this to `hello.md` and run it as follows:

```shell
$ mls run hello.md
```

This will do shockingly little, because nothing is invoking this function yet. However, Matterless comes with a simple console we can use to manually invoke this function (if you don't see the `>` prompt hit Enter first). First we need to switch to the `hello` application:

```
> use hello
```

Then, we can invoke our function with an empty object:

```
hello> invoke HelloWorld {}
```

This will print something along the lines of:

```
INFO[0015] [App: hello | Function: HelloWorld] Starting deno function runtime. 
INFO[0016] [App: hello | Function: HelloWorld] Hello world! 
```

Success!

For the remainder of this README Matterless definitions will be inlined as Markdown, so they're easier to read. A horizontal rule will be used to make clear where the application code starts and ends.

Let's look at the function definition type and other support definition types more closely.

# Matterless Definition Types
These are the _definition types_ currently supported and how to use them.

## function MyFunction
As of this writing, JavaScript is the primary supported language. More runtimes (based on docker) will be added in the future.

The JavaScript function that will be invoked needs to be called `handle` and take a single argument: `event`, which will receive event data (depending on how the function will be triggered) and may or may not return a result.

While technically in most cases a Deno process instance with your function code inside it will be reused (it's not relaunched for every invocation), you should assume a stateless environment. While technically you have full access to all Deno APIs, your function may be killed at any time along with all its in-memory and disk state. In fact, in the current implementation will indeed happen after a brief amount of time of inactivity.

A function definition may contain an optional configuration YAML block:

```yaml
init:
   name: Donald Knuth
runtime: deno
```

The values put into `init` (which usually would be an object, but it could be an YAML array as well) will be passed to the `init` JavaScript function upon cold start:

```javascript
function init(config) {
    console.log(`Hello there, I'm initing for ${config.name}`);
}

function handle(event) {
    console.log("I was just run with", event);
}
```

When first invoked, this will log something along the lines of:

    INFO[0017] [App: README | Function: MyFunction] Hello there, I'm initing for Donald Knuth
    INFO[0017] [App: README | Function: MyFunction] I was just run with {}

Subsequent invocations will skip the initialization.

## job StarGazerPoll
Jobs are much like `function`s, except they boot up immediately upon the application start and keep running during the lifetime of the application. Like `function`s, jobs support an optional YAML configuration block:

```yaml
init:
   repo: zefhemel/matterless
   pollInterval: 60
   event: starschanged
```

Rather than implementing the `handle` JavaScript function, a job implements `start` and (optionally) `stop`.

In this example we're going to poll the Github API every 60 seconds to see if the number of stars on the matterless repository has changed and publishing a `starschanged` event when it does. Note that this example uses various core Matterless APIs: `store` and `events` to track state between runs. Theoretically a global variable could be used, but this value would be lost between restarts of the app:

```javascript
import {store, events} from "./matterless.ts";

let config;

function init(cfg) {
    console.log("Inited with", cfg);
    config = cfg;
}

function start() {
    setInterval(async () => {
        // Pull old star count from the store (or set to 0 if no value)
        let oldStarCount = (await store.get("stars")) || 0;
        
        // Talk to Github API to fetch new value
        let result = await fetch(`https://api.github.com/repos/${config.repo}`);
        let json = await result.json();
        let newCount = json.stargazers_count;
        
        // It changed!
        if(newCount !== oldStarCount) {
            // Publish event
            await events.publish(config.event, {
                stars: newCount
            });
            // Store new value in store
            await store.put("stars", newCount);
        } else {
            console.log("No change :-(");
        }
    }, config.pollInterval * 1000);
}

function stop() {
    console.log("Shutting down stargazer poller");
}
```

## events
Using events mappings we define which events should invoke which functions. Multiple functions can be invoked in response to a single event, therefore we specify them as a list:

```yaml
starschanged:
  - StarGazeReporter
"http:GET:/myAPI":
  - MyHTTPAPI
```

There are a few built-in events. One example is the `http:GET:/myAPI` event, which 

## function MyHTTPAPI
```javascript
import {events} from "./matterless.ts";

function handle(req) {
    events.respond(req, {
        status: 200,
        body: "Hello there!"
    });
}
```

## function StarGazeReporter
```javascript
function handle(event) {
    console.log("Number of stars changed to", event.stars);
}
```

## macro httpApi
```yaml
input_schema:
  type: object
  properties:
    path:
      type: string
    method:
      type: string
    function:
      type: string
  required:
    - path
    - method
    - function
```

And the template:

    ## events
    ```yaml
    "http:{{$input.method}}:{{$input.path}}":
    - {{$input.function}}
    ```


And the instantiation

## httpApi MyAPI
```yaml
path: /anotherAPI
method: GET
function: MyHTTPAPI
```



# Installation
Requirements:
* Go 1.16 or newer

Tested on Mac (Apple Silicon) and Linux (AMD64), although other platforms should work as well.

**Warning:** This is still early stage software, I recommend you only use it with development or testing instances of Mattermost, not production ones.

To install Matterless:

```shell
$ go get github.com/zefhemel/matterless/...
$ go install github.com/zefhemel/matterless/cmd/mls@latest
````

This will install the binaries in your `$GOPATH/bin`.

# Running Matterless
Matterless has three modes of operation:

1. All-in-one mode via `mls run`, this will run both the server and client in a single process. This is useful for development, especially with the `-w` option that watches the files you point to for changes:
    ```shell
   $ mls run -w myapp.md 
   ```
2. Server mode by simply running `mls` optionally with arguments like `-p` to bind to a specific port (defaults to `8222`), `--data` to select the data directory (default: `./mls-data`) and `--token` to use a specific admin token (generates one by default):
    ```shell
    $ mls --data /var/data/mls 
    ```
3. Client/deploy mode to deploy code to a remote (or local) matterless server:
    ```shell
    $ mls deploy --url http://mypi:8222 --token mysecrettoken -w myapp.md
    ```

Enjoy!