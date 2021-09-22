export async function ensureConsoleChannel(mmClient, teamId, appId, appName) {
    let channelName = `matterless-console-${appId}`.toLowerCase();
    let channelInfo;
    try {
        channelInfo = await mmClient.getChannelByNameCached(teamId, channelName);
    } catch (e) {
        // Doesn't exist yet, let's create
        channelInfo = await mmClient.createChannel({
            team_id: teamId,
            name: channelName,
            display_name: `Matterless : Logs : ${appName}`,
            header: `Logs for your matterless app ${appName}`,
            type: 'P'
        })
    }

    return channelInfo;
}

export function postLink(url, team, postId) {
    return `${url}/${team.name}/pl/${postId}`;
} 
