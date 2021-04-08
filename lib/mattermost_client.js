export class Mattermost {
    constructor(url, token) {
        this.url = `${url}/api/v4`;
        this.token = token;
        this.meCache = null;
        this.userCache = {};
    }

    async performFetch(path, method, body) {
        let result = await fetch(`${this.url}${path}`, {
            method: method,
            headers: {
                'Authorization': `bearer ${this.token}`
            },
            body: body ? JSON.stringify(body) : undefined
        });
        if(result.status < 200 || result.status > 299) {
            throw Error((await result.json()).message);
        }
        return result.json();
    }

    async getMe() {
        return this.performFetch("/users/me", "GET");
    }

    async getMeCached() {
        if(!this.meCache) {
            this.meCache = await this.getMe();
        }
        return this.meCache;
    }

    async getUser(userId) {
        return this.performFetch(`/users/${userId}`, "GET");
    }

    async getUserCached(userId) {
        if(!this.userCache[userId]) {
            this.userCache[userId] = await this.getUser(userId);
        }
        return this.userCache[userId];
    }

    async getUserTeams(userId) {
        return this.performFetch(`/users/${userId}/teams`, "GET");
    }

    async getPrivateChannels(teamId) {
        return this.performFetch(`/teams/${teamId}/channels/private`, "GET");
    }

    async getChannelByName(teamId, name) {
        return this.performFetch(`/teams/${teamId}/channels/name/${name}`, "GET")
    }

    async createChannel(channel) {
        return this.performFetch("/channels", "POST", channel);
    }

    async createDirectChannel(userId1, userId2) {
        return this.performFetch(`/channels/direct`, "POST", [userId1, userId2]);
    }

    async createPost(post) {
        return this.performFetch("/posts", "POST", post);
    }

    async updatePost(post) {
        return this.performFetch(`/posts/${post.id}`, "PUT", post);
    }

    async deletePost(post)  {
        return this.performFetch(`/posts/${post.id}`, "DELETE");
    }

    async getPost(postId) {
        return this.performFetch(`/posts/${postId}`, "GET");
    }
}
