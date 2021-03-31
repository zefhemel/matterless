
export class Mattermost {
    constructor(url, token) {
        this.url = `${url}/api/v4`;
        this.token = token;
    }

    async performFetch(path, method, body) {
        let result = await fetch(`${this.url}${path}`, {
            method: method,
            headers: {
                'Authorization': `bearer ${this.token}`
            },
            body: body ? JSON.stringify(body) : undefined
        });
        return result.json();
    }

    async getMe() {
        return this.performFetch("/users/me", "GET");
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


    async createPost(post) {
        return this.performFetch("/posts", "POST", post);
    }

    async updatePost(post) {
        return this.performFetch(`/posts/${post.id}`, "PUT", post);
    }

    async deletePost(post)  {
        return this.performFetch(`/posts/${post.id}`, "DELETE");
    }
}
