const rootUrl = Deno.env.get("API_URL").split('/').slice(0, -1).join('/');

export function buildAppName(userId, postId) {
    return `mls_${postId}`;
}

export async function adminCall(rootToken, path, method, body) {
    let result = await fetch(`${rootUrl}${path}`, {
        method: method,
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `bearer ${rootToken}`
        },
        body: body
    });
    let textResult = await result.text();
    if (result.status > 300 || result.status < 200) {
        throw Error(textResult);
    }
    return textResult;
}

export async function listApps(rootToken) {
    return JSON.parse(await adminCall(rootToken, "", "GET"));
}

export async function putApp(rootToken, name, code) {
    return adminCall(rootToken, `/${name}`, "PUT", code);
}

export async function deleteApp(rootToken, name) {
    return adminCall(rootToken, `/${name}`, "DELETE");
}

export async function getAppCode(rootToken, name) {
    return adminCall(rootToken, `/${name}`, "GET");
}

export function appEventSocket(rootToken, appName) {
    console.log("Now going to subscribe: ", `${rootUrl}/${appName}/_events`)
    let ws = new WebSocket(`${rootUrl.replace('http://', 'ws://')}/${appName}/_events`);
    ws.addEventListener('error', function (ev) {
        console.error("Got websocket error", ev);
    });
    return ws;
}

export function globalEventSocket(rootToken) {
    console.log("Now going to subscribe: ", `${rootUrl}/_events`)
    let ws = new WebSocket(`${rootUrl.replace('http://', 'ws://')}/_events`);
    ws.addEventListener('error', function (ev) {
        console.error("Got websocket error", ev);
    });
    return ws;
}

export function appApiUrl(appName) {
    return `${rootUrl}/${appName}`;
}
