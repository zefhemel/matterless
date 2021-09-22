import {store} from "./matterless.ts";
import {Mattermost, seenEvent} from "./mattermost_client.js";
import {buildAppName, putApp, deleteApp} from "./matterless_root.js";
import {ensureConsoleChannel, postLink} from "./util.js";

let mmClient;
let rootToken;
let team;
let url;

async function init(cfg) {
    url = cfg.url;
    mmClient = new Mattermost(cfg.url, cfg.token);
    rootToken = cfg.root_token;
    team = await mmClient.getTeamByName(cfg.team);
}

// Main event handler
async function handle(event) {
    let post = JSON.parse(event.data.post);
    let me = await mmClient.getMeCached();

    // Lookup channel
    let channel = await mmClient.getChannelCached(post.channel_id);
    // Ignore bot posts
    if (post.user_id === me.id) return;
    // Skip any message outside a private chat with the bot
    if (channel.type != 'D') return;
    // Deduplicate events (bug in editing posts in Mattermost)
    if (seenEvent(event)) {
        return;
    }

    // Check permissions
    let allowedUserNames = (await store.get('config:allowed_users')) || [];
    let allowedUserIds = {};

    for (let username of allowedUserNames) {
        let user = await mmClient.getUserByUsernameCached(username);
        allowedUserIds[user.id] = true;
    }
    if (!allowedUserIds[post.user_id]) {
        await ensureMyReply(me, post, "You're not on the allowed list :thumbsdown:");
        return;
    }
    let appName = buildAppName(post.user_id, post.id);
    let code = post.message;
    let userId = post.user_id;
    if (event.event === 'post_deleted') {
        console.log("Deleting app:", appName);
        await deleteApp(rootToken, appName);
    } else {
        if (!post.root_id) {
            console.log("Updating app:", appName, event);
            // Attempt to extract an appname
            let friendlyAppName = post.id;
            let matchGroups = /#\s*([A-Z][^\n]+)/.exec(code);
            if (matchGroups) {
                friendlyAppName = matchGroups[1];
            }
            let channelInfo = await ensureConsoleChannel(mmClient, team.id, post.id, friendlyAppName);
            channelInfo.display_name = `Matterless : ${friendlyAppName}`;
            channelInfo.header = `Console for your matterless app [${friendlyAppName}](${postLink(url, team, post.id)})`;
            await mmClient.updateChannel(channelInfo);

            await mmClient.addUserToChannel(channelInfo.id, userId);
            try {
                let result = await putApp(rootToken, appName, code);
                await mmClient.addReaction(me.id, post.id, "white_check_mark");
                await mmClient.removeReaction(me.id, post.id, "octagonal_sign");
                await ensureMyReply(me, post, 'All good to go :thumbsup:');
            } catch (e) {
                try {
                    let errorData = JSON.parse(e.message);
                    if (errorData.error === 'config-errors') {
                        let toConfigure = errorData.data;
                        // Pick any to configure
                        let first = Object.keys(toConfigure)[0];
                        await mmClient.createPost({
                            channel_id: post.channel_id,
                            root_id: post.id,
                            message: `Some configuration is required, please enter a value for **${first}**:`,
                            props: {
                                configName: first
                            }
                        });
                    }
                } catch (e) {
                    console.error(e);
                    await ensureMyReply(me, post, `ERROR: ${e.message}`);
                }
                await mmClient.addReaction(me.id, post.id, "octagonal_sign");
                await mmClient.removeReaction(me.id, post.id, "white_check_mark");
            }
        } else {
            // Reply
            console.log("Reply post:", post);
            let thread = await mmClient.getThread(post.root_id);
            let lastQuestion;
            for (let postId of thread.order) {
                let threadPost = thread.posts[postId];
                if (threadPost.user_id == me.id) {
                    lastQuestion = threadPost;
                }
            }
            console.log("This is in response to", lastQuestion);

        }


    }
}

async function ensureMyReply(me, post, message) {
    let thread = await mmClient.getThread(post.id);
    let replyPost;
    for (let postId of thread.order) {
        let threadPost = thread.posts[postId];
        if (threadPost.user_id == me.id) {
            replyPost = threadPost;
        }
    }
    if (replyPost) {
        replyPost.message = message;
        await mmClient.updatePost(replyPost);
    } else {
        await mmClient.createPost({
            channel_id: post.channel_id,
            root_id: post.id,
            message: message
        });
    }
}


var _init, _start, _run, _stop, _handle;
(function() {
    var initData = {"root_token":"zef","team":"test","token":"n1wmzk17qpnei8hh6kmticnuzy","url":"http://localhost:8065"};
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
