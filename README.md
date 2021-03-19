# Matterless
Matterless is a tool to quickly and iteratively develop and run Mattermost "applications": extensions of Mattermost that may include bots, slash commands and other types of integrations. When run in "bot mode", users can manage
Matterless applications as direct messages to the `@matterless` bot, which will immediately parse, validate and
activate them. Generally this happens in a matter of seconds, allowing for a **very rapid code and run cycle**.

![](https://raw.githubusercontent.com/zefhemel/matterless/master/screenshots/matterless-bot-success.png)

Matterless can also be run in CLI-mode, in which case it is pointed at an application file (which you can edit in your favorite IDE) that will be hot-reloaded whenever it has been updated.

Conceptually Matterless is inspired by _serverless_ (hence the name), where applications are built using many off-the-shelf services and components glued together by lambda functions that respond to events.

**Note:** Matterless is early stage of development, please only use it for development and testing purposes. The APIs here may be changed at any time.

There is a [video demo of a somewhat earlier version available here](https://cln.sh/gnC8Md). To have a look at some simple example applications, check out the [samples](https://github.com/zefhemel/matterless/tree/master/samples) folder.

## What is a Matterless application
A Matterless application consists of a number of _definitions_ written in Markdown. Currently the following _definition types_ are supported:

Logic:
* `Function`: for defining functions (snippets of code that are run when certain events occur, similar to AWS lambda functions)
* `Library`: a convenient way to write reusable code once that is automatically available (conceptually: appended) in all functions in this application
* `Environment`:  defines environment variables available to all functions in the application.
  
Event sources:
* `MattermostClient`: an event source connecting to a Mattermost instance (based on an access token), triggering functions based on specified events.
* `SlashCommand`: an event source defining a slash command (e.g. `/my-command`), triggering a function when run.
* `Bot`: an event source defining a Mattermost bot account, triggering functions based on specified events. Conceptually this is a `MattermostClient` wrapped in API calls that create the bot and managed tokens automatically.
* `Cron`: an event source triggering functions at a specified schedule.
* `API`: enables to connect functions to a HTTP server run by Matterlesss, that can be called by external systems to trigger logic.

Matterless applications are written in markdown, following certain conventions. Markdown is used because it actually fits quite well, and it renders nicely in a Mattermost post.

Here is "Hello world" in Matterless:

-----
    # Function: HelloWorld
    ```javascript
    function handle(event) {
        console.log("Hello world!");
    }
    ```
-----

For the remainder of this README these definitions will be inlined as Markdown, so they're easier to read. A horizontal rule will be used to make clear where the application code starts and ends.

This application defines a single Matterless function called `HelloWorld` written in JavaScript. Functions are the core building blocks used to build matterless applications. When loaded by Matterless, this function will be invoked with an empty object (`{}`) as a warm-up call once, to ensure the code doesn't instantly crash. You can check if the event is a warm-up event and ignore it as follows (but you could also abuse it do perform initialization logic):

-----
## Function: HelloFunction
```javascript
function handle(event) {
    if(isWarmupEvent(event)) return;
    console.log("Hello world!");
}
```
-----

The general structure for a Mattermost definition is a header (at any level, e.g. `#`, `##` or `###`) prefixed with a _definition type_, a colon (`:`) and a name, then a braced code block typically using YAML or JavaScript dependent on the definition type. You can put arbitrary other Markdown text, markup, links, lists etc. around these — they will be ignored, making this also a good environment for [literate programming](https://en.wikipedia.org/wiki/Literate_programming). In fact... this very README.md is a valid Mattermost application :mindblown: (albeit not a very useful one).


# Matterless Definition Types
These are the _definition types_ currently supported and how to use them.

## Function: MyFunction
Currently the only language supported is JavaScript, which is run using node.js (in ES6 with modules mode) that is run in a docker container. The function that will be invoked needs to be called `handle` and take a single argument: `event`, which will receive event data (depending on how the function will be triggered) and may or may not return a result.

While technically in most cases a node.js process instance with your function code inside it will be reused (it's not relaunched for every invocation), you should assume a stateless environment. While technically you have full access to all node.js APIS and the entire docker file system (yes, you could probably run a bitcoin miner in there), your function may be killed at any time along with all its in-memory and disk state. In fact, in the current implementation will indeed happen after a brief amount of time of inactivity.

Here is an example function that uses various Matterless APIs:
* The `Mattermost` API, which is essentially just the JavaScript [MatterMost client](https://github.com/mattermost/mattermost-redux/blob/master/src/client/client4.ts) with some niceties added (like caching versions of some calls).
* The `Store` API, which is a [super simple key-value store](https://github.com/zefhemel/matterless/blob/master/runners/docker/node_modules/matterless/index.mjs) you can use to keep some state (currently implemented using LevelDB). State is stored on the Matterless server-side, and therefore persistent (but not shared between applications).

In addition, a few other npm modules are included in the runtime environment, you can [see the list and versions here](https://github.com/zefhemel/matterless/blob/master/runners/docker/package.json).

Here is a function that demonstrates some of the APIs:
```javascript
import {Store, Mattermost} from "matterless";

// Instantiates a new store API client
let store = new Store();
// Connect to the mattermost API authenticated as the bot account (defined later)
let client = new Mattermost(process.env.MYBOT_URL, process.env.MYBOT_TOKEN);

async function handle(event) {
    if(isWarmupEvent(event)) return;
    
    let post = JSON.parse(event.data.post);

    // Lookup channel
    let channel = await client.getChannelCached(post.channel_id);
    // Ignore posts sent by myself
    let botUser = await client.getMeCached();
    if(post.user_id === botUser.id) return;

    let counter = (await store.get("silly-counter")) || 0;
    await store.put("silly-counter", ++counter);
    await client.createPost({
        channel_id: post.channel_id,
        root_id: post.id,
        parent_id: post.id,
        message: `I am ${botUser.username} and the counter is ${counter}. Also: ${reusableFunction()}`
    });
}
```

As mentioned, right now functions run are run in docker containers locally, in the future there may be other sandboxes implemented, such one based on AWS lambda, or Kubernetes.

## Library
It is likely to happen that you'll want to share some code between multiple functions. To do this, you can use the Library. In essence, any code you define in a Library will be appended to all other function code (note that in the previous example the below `reusableFunction` is invoked). 

```javascript
function reusableFunction() {
    return "yo";
}
```

### MattermostClient: MyMattermostClient
The MattermostClient event source can connect to any Mattermost instance you can authenticate with using a token. It then starts to listen to specific or all (websocket) events. Use `all` as a catch-all (mostly useful for debugging and exploration). You can connect multiple functions to a single event, therefore you specify them as a list in YAML.

When defining a MattermostClient, two new environment variables will be defined (accessible in node.js via `process.env`): `MYMATTERMOSTCLIENT_URL` (in this case, always all-caps) and `MYMATTERMOSTCLIENT_TOKEN` containing the URL and token, respecitively, which can be used to authenticate as this user and e.g. reply to a post in case of a `posted` event.

```yaml
url: http://localhost:8065
token: 1234
events:
  all:
  - HelloFunction
  posted:
  - HelloFunction
```

### Bot: MyBot
A bot uses Matterless' admin account (see the _Running Matterless_ section below) to automatically create or update a bot with a specific username. It will also manage tokens for you.

In a sense a Bot is the same as `MattermostClient`, but it manages the tokens for you (and always connects to the main configured Mattermost instance). The same `MYBOT_URL` and `MYBOT_TOKEN` environment variables will be exposed as for MattermostClient.

```yaml
username: my-bot
events:
  posted:
    - MyFunction
```

### SlashCommand: MySlashCommand
A slash command defines... a new slash command. In the case below a `/my-command` command. When a user triggers it, it will invoke the `MyCommand` func.
```yaml
trigger: my-command
auto_complete: true
auto_complete_desc: My awesome command
auto_complete_hint: who
function: MyCommandFunc
team_name: Dev
```

#### Function: MyCommandFunc
When triggered by a SlashCommand, our function's event contains all the top-level keys documented [in the Mattermost documentation](https://docs.mattermost.com/developer/slash-commands.html#custom-slash-command) under item (8), so e.g. `channel_id`, `text`, `token` etc. The function's return value is also in the same format as documented.
```javascript
function handle(event) {
    return {
        text: "Hey, " + event.text
    };
}
```

### Cron
Crons can be used to schedule regularly recurring tasks. The `schedule` is written in a regular crontab format. Check [this reference for all options available](https://pkg.go.dev/github.com/robfig/cron?utm_source=godoc#hdr-CRON_Expression_Format).
```yaml
- schedule: "0 * * * * *"
  function: HelloFunction
```

## API
For certain use cases it will be required to expose an HTTP endpoint that either Mattermost can call, or that is called externally. For this, you can use an API.
```yaml
- path: /hello
  function: HelloWorldHTTP
  methods:
    - GET
    - POST
```

### Function: HelloWorldHTTP
The `event` data you will receive for an HTTP event is [as follows](https://github.com/zefhemel/matterless/blob/master/pkg/definition/http.go):
* `path`: the fully path for the request
* `method`: the request method (e.g. `POST` or `GET`)
* `headers`: a map with header -> value mappings
* `form_values`: if the content-type of the request is for a form, this will contain a map of parsed values
* `request_params`: contains a map of request parameters passed via the URL
* `json_body`: will contain a parsed JSON body when the content-type is `application/json`

A function is supposed to return a HTTP response object with:

* `status`: HTTP status code
* `headers`:  a map of headers
* `body`: can be either a string, or a JSON object

```javascript
function handle(event) {
    return {
        status: 200,
        headers: {
            "Content-type": "text/plain"
        },
        body: "Hello world!"
    }
}
```
# Installation
Requirements:
* Go 1.16 or newer
* Docker
* A Mattermost install where you have access to an admin account (or at least an _personal access token_ for one)

Tested on Mac (Apple Sillicon) and Linux (intel), although other platforms should work as well.

**Warning:** This is still early stage software, I recommend you only use it with development or testing instances of Mattermost, not production ones.

Make sure you have a personal access token for an admin account. Matterless will use this to create the matterless bot, and later to have permission to create all the resources required to run matterless apps. If you don't have one, create one via: `Account settings > Security > Personal Access Token` If you don't have this option, you need to enable "Enable Personal Access Tokens" under "Integration Management" in the Console.

To install matterless:

```shell
$ go get github.com/zefhemel/matterless/...
$ go install github.com/zefhemel/matterless/cmd/{mls,mls-bot}@latest
```

This will install the binaries in your `$GOPATH/bin`. Then, create a directory to keep matterless state data (configuration and data):

```shell
$ mkdir mls
$ cd mls
```

In this folder create a `.env` file (or, alternatively set these as environment variables if you prefer):

```
mattermost_url=https://your.matterless.site.com
admin_token=my-admin-token
team_name=your-team-name
api_bind_port=8222
api_url=http://server.running.matterless.com:8222
leveldb_databases_path=data
```

If you run everything locally, these are likely accurate values:
```
mattermost_url=http://localhost:8065
admin_token=my-admin-token
team_name=your-team-name
api_bind_port=8222
api_url=http://localhost:8222
leveldb_databases_path=data
```

**Important:** Matterless will expose a (plain) HTTP server, binding to the configured port (`api_bind_port`). The use case is to provide various callback URLs e.g. for slash commands and accessing Matterless APIs like the store API. In a default Mattermost configuration _this will not work out of the box_, because no untrusted calls are allowed from Mattermost to HTTP urls. There are two ways to solve this:

1. For development: add your matterless hostname (e.g. `localhost`) to "Allow untrusted internal connections to" in the Console (under Environment > Developer).
2. For production: put a HTTPS proxy on top of this port, and point `api_url` to the resulting `https://` URL.

# Running Matterless
Matterless has two modes of operation:

1. As a bot (`mls-bot`) that users can send matterless application definitions to that are subqequently deployed on the instance. The application is reloaded when the post containing the definition is edited, and unloaded when the post is deleted. Logs appear in dedicated channels per function.
2. As a command-line tool (`mls`) pointing at a markdown file containing a matterless application and operates it. Changes to the markdown file are hot reloaded for quick iteration. All logs are sent to CLI console.


To run it as a bot:
```shell
$ mls-bot
```

You should now get a bunch of output in the console, and be pinged on Mattermost about the creation of the `matterless` bot! To test if it works, send it a "ping" message (it should attach a ping-pong reaction). Then to build your first app, copy and paste the example at the top this README.

Alternatively, copy & paste your code into a file and run it via `mls`:

```shell
mls my-app.md
```

This will hot-reload whenever you make changes to `my-app.md`.

Enjoy!