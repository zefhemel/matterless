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

    // Me
    async getMe() {
        return this.performFetch("/users/me", "GET");
    }

    async getMeCached() {
        if(!this.meCache) {
            this.meCache = await this.getMe();
        }
        return this.meCache;
    }

    // Users
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

    async getUserByUsername(username) {
        return this.performFetch(`/users/username/${username}`, "GET");
    }

    async createUserAccessToken(userId, description) {
        return this.performFetch(`/users/${userId}/tokens`, "POST", {
            description
        });
    }

    async getUserAccessTokens(userId) {
        return this.performFetch(`/users/${userId}/tokens`, "GET");
    }

    async revokeUserAccessToken(userId, tokenId) {
        return this.performFetch(`/users/${userId}/tokens/revoke`, "POST", {
            token_id: tokenId
        });
    }

    // Teams
    async getTeamByName(name) {
        return this.performFetch(`/teams/name/${name}`, "GET");
    }

    async addUserToTeam(userId, teamId) {
        return this.performFetch(`/teams/${teamId}/members`, "POST", {
            team_id: teamId,
            user_id: userId
        });
    }

    // Bots
    async getBots() {
        return this.performFetch(`/bots`, "GET");
    }

    async createBot(bot) {
        return this.performFetch(`/bots`, "POST", bot);
    }

    async updateBot(bot) {
        return this.performFetch(`/bots/${bot.user_id}`, "PUT", bot);
    }

    // Channels

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

    // Posts
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
