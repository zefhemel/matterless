# Mattermost Matterless plugin

Early prototype, does not support clustering yet.

What it does:

1. Runs matterless as a plugin, with the API Gateway server bound to a random port
2. Any HTTP requests sent to `$YOUR_MATTERMOSTSERVER/plugins/com.mattermost.matterless-plugin` are proxied to the API Gateway server
3. Regular Matterless authentication (via admin token) is replaced with Mattermost tokens. Users with ServerAdmin priviledges are authenticated as admins in Matterless as well.
4. You can configure the data path in the console (defaults to `./mls-data` in your mattermost install)

How to deploy an app to it:

Use the regular `mls deploy command`:

```shell
$ mls deploy --url http://localhost:8065/plugins/com.mattermost.matterless-plugin --token $any-mattermost-admin-token yourapp.md
```

The current `go.mod` has a replacement, assuming you have a [matterless](https://github.com/zefhemel/matterless) checkout sitting next to this plugin.


### Deploying with Local Mode

If your Mattermost server is running locally, you can enable [local mode](https://docs.mattermost.com/administration/mmctl-cli-tool.html#local-mode) to streamline deploying your plugin. Edit your server configuration as follows:

```json
{
    "ServiceSettings": {
        ...
        "EnableLocalMode": true,
        "LocalModeSocketLocation": "/var/tmp/mattermost_local.socket"
    }
}
```

and then deploy your plugin:
```
make deploy
```

You may also customize the Unix socket path:
```
export MM_LOCALSOCKETPATH=/var/tmp/alternate_local.socket
make deploy
```

If developing a plugin with a webapp, watch for changes and deploy those automatically:
```
export MM_SERVICESETTINGS_SITEURL=http://localhost:8065
export MM_ADMIN_TOKEN=j44acwd8obn78cdcx7koid4jkr
make watch
```

### Deploying with credentials

Alternatively, you can authenticate with the server's API with credentials:
```
export MM_SERVICESETTINGS_SITEURL=http://localhost:8065
export MM_ADMIN_USERNAME=admin
export MM_ADMIN_PASSWORD=password
make deploy
```

or with a [personal access token](https://docs.mattermost.com/developer/personal-access-tokens.html):
```
export MM_SERVICESETTINGS_SITEURL=http://localhost:8065
export MM_ADMIN_TOKEN=j44acwd8obn78cdcx7koid4jkr
make deploy
```
