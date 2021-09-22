import {listApps, globalEventSocket} from "./matterless_root.js";
import {Mattermost} from "./mattermost_client.js";
import {ensureConsoleChannel} from "./util.js";

let sockets = [];
let rootToken;
let mmClient;
let team;
let url;

async function init(cfg) {
    url = cfg.url;
    rootToken = cfg.root_token;
    mmClient = new Mattermost(cfg.url, cfg.bot_token);
    team = await mmClient.getTeamByName(cfg.team);
}


async function run() {
    let ws = globalEventSocket(rootToken);
    console.log("Subscribing to all log events");
    try {
        ws.addEventListener('open', function () {
            ws.send(JSON.stringify({type: "authenticate", token: rootToken}));
        });
        ws.addEventListener('message', function (event) {
            let logJSON = JSON.parse(event.data);
            switch (logJSON.type) {
                case 'authenticated':
                    console.log("Log listener authenticated");
                    ws.send(JSON.stringify({
                        type: 'subscribe',
                        pattern: '*.*.log'
                    }));
                    break;
                case 'error':
                    console.error("Got WS error:", logJSON.error);
                    break
                case 'event':
                    if (!logJSON.app.startsWith('mls_')) {
                        return
                    }
                    // console.log("Got a lot event", logJSON);
                    let [_, appId] = logJSON.app.split('_');
                    let functionName = logJSON.name.split('.')[0];
                    publishLog(appId, functionName, logJSON.data.message);
                    break;
            }
        });
    } catch (e) {
        console.error(e);
    }
}

async function publishLog(appId, functionName, message) {
    // console.log(`${userId}: ${appId} [${functionName}] ${message}`);
    let channelName = `matterless-console-${appId}`.toLowerCase();
    let channelInfo = await mmClient.getChannelByName(team.id, channelName);
    await mmClient.createPost({
        channel_id: channelInfo.id,
        message: "From `" + functionName + "`:\n```\n" + message + "\n```"
    });
}


var _init, _start, _run, _stop, _handle;
(function() {
    var initData = {"bot_token":"n1wmzk17qpnei8hh6kmticnuzy","root_token":"zef","team":"test","url":"http://localhost:8065"};
    // Initialization
    try {
        _init = init.bind(null, initData);
    } catch (e) {
        _init = () => {
        };
    }

// Functions
    try {
        _handle = handle;
    } catch (e) {
        _handle = () => {
        };
    }

// Jobs
    try {
        _start = start;
    } catch (e) {
        _start = () => {
        };
    }
    try {
        _run = run;
    } catch (e) {
        _run = () => {
        };
    }

    try {
        _stop = stop;
    } catch (e) {
        _stop = () => {
        };
    }
})();


export {
    _init as init,
    _start as start,
    _stop as stop,
    _run as run,
    _handle as handle
};
