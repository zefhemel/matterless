import 'babel-polyfill';
import 'isomorphic-fetch';
import client4 from 'mattermost-redux/client/client4.js';

export class Mattermost extends client4['default'] {
    constructor(url, token) {
        super();
        this.setUrl(url);
        this.setToken(token);

        this.channelCache = {};
        this.userCache = {};
        this.meCache = null;
    }

    async getChannelCached(channelId) {
        if(!this.channelCache[channelId]) {
            this.channelCache[channelId] = await this.getChannel(channelId);
        }
        return this.channelCache[channelId];
    }

    async getUserCached(userId) {
        if(!this.userCache[userId]) {
            this.userCache[userId] = await this.getUser(userId);
        }
        return this.userCache[userId];
    }

    async getMeCached() {
        if(!this.meCache) {
            this.meCache = await this.getMe();
        }
        return this.meCache;
    }
}

export class Store {
    constructor() {

    }

    async get(key) {
        return (await this.performOp("get", key)).value;
    }

    async put(key, value) {
        return this.performOp("put", key, value);
    }

    async del(key) {
        return this.performOp("del", key);
    }

    async queryPrefix(prefix) {
        return (await this.performOp("query-prefix", prefix)).results;
    }

    async performOp(...args) {
        let result = await fetch(`${process.env.API_URL}/_store`, {
            method: "POST",
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `bearer ${process.env.API_TOKEN}`
            },
            body: JSON.stringify([args])
        })
        let jsonResult = (await result.json())[0];
        if(jsonResult.status === "error") {
            throw Error(jsonResult.error);
        }
        return jsonResult;
    }
}