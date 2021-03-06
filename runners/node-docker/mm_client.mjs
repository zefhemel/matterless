import 'babel-polyfill';
import 'isomorphic-fetch';
import client4 from 'mattermost-redux/client/client4.js';

export default function(url, token) {
    const client = new client4['default']();
    client.setUrl(url || process.env.SOURCE_URL);
    client.setToken(token || process.env.SOURCE_TOKEN);
    return client;
}