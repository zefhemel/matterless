class API {
    url;
    token;
    constructor(url, token) {
        this.url = url;
        this.token = token;
    }
    getStore() {
        return new Store(this.url, this.token);
    }
    getEvents() {
        return new Events(this.url, this.token);
    }
    getFunctions() {
        return new Functions(this.url, this.token);
    }
    getApplication() {
        return new Application(this.url, this.token);
    }
}
class Store {
    url;
    token;
    constructor(url, token) {
        this.url = url;
        this.token = token;
    }
    async get(key) {
        return (await this.performOp("get", key)).value;
    }
    async put(key, value) {
        await this.performOp("put", key, value);
    }
    async del(key) {
        await this.performOp("del", key);
    }
    async queryPrefix(prefix) {
        return (await this.performOp("query-prefix", prefix)).results || [];
    }
    async performOp(...args) {
        let result = await fetch(`${this.url}/_store`, {
            method: "POST",
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `bearer ${this.token}`
            },
            body: JSON.stringify([args])
        });
        let jsonResult = (await result.json())[0];
        if (jsonResult.status === "error") {
            throw Error(jsonResult.error);
        }
        return jsonResult;
    }
}
class Events {
    url;
    token;
    constructor(url, token) {
        this.url = url;
        this.token = token;
    }
    async publish(eventName, eventData) {
        let result = await fetch(`${this.url}/_event/${eventName}`, {
            method: "POST",
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `bearer ${this.token}`
            },
            body: JSON.stringify(eventData || {})
        });
        let jsonResult = await result.json();
        if (jsonResult.status === "error") {
            throw Error(jsonResult.error);
        }
    }
}
class Functions {
    url;
    token;
    constructor(url, token) {
        this.url = url;
        this.token = token;
    }
    async invoke(name, eventData) {
        let result = await fetch(`${this.url}/_function/${name}`, {
            method: "POST",
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `bearer ${this.token}`
            },
            body: JSON.stringify(eventData || {})
        });
        if (result.status == 200) {
            let jsonResult = await result.json();
            if (jsonResult.status === "error") {
                throw Error(jsonResult.error);
            }
            return jsonResult;
        }
        else {
            throw new Error(`HTTP request not ok: ${await result.text()}`);
        }
    }
}
class Application {
    url;
    token;
    constructor(url, token) {
        this.url = url;
        this.token = token;
    }
    async restart() {
        let result = await fetch(`${this.url}/_restart`, {
            method: "POST",
            headers: {
                'Authorization': `bearer ${this.token}`
            },
        });
        if (result.status != 200) {
            throw new Error(`HTTP request not ok: ${await result.text()}`);
        }
    }
}
const defaultApi = new API(Deno.env.get("API_URL"), Deno.env.get("API_TOKEN")), store = defaultApi.getStore(), events = defaultApi.getEvents(), functions = defaultApi.getFunctions(), application = defaultApi.getApplication();
export { store, events, functions, application, API };
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibWF0dGVybGVzcy5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbIm1hdHRlcmxlc3MudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQUEsTUFBTSxHQUFHO0lBQ0wsR0FBRyxDQUFTO0lBQ1osS0FBSyxDQUFTO0lBRWQsWUFBWSxHQUFXLEVBQUUsS0FBYTtRQUNsQyxJQUFJLENBQUMsR0FBRyxHQUFHLEdBQUcsQ0FBQztRQUNmLElBQUksQ0FBQyxLQUFLLEdBQUcsS0FBSyxDQUFDO0lBQ3ZCLENBQUM7SUFFRCxRQUFRO1FBQ0osT0FBTyxJQUFJLEtBQUssQ0FBQyxJQUFJLENBQUMsR0FBRyxFQUFFLElBQUksQ0FBQyxLQUFLLENBQUMsQ0FBQztJQUMzQyxDQUFDO0lBRUQsU0FBUztRQUNMLE9BQU8sSUFBSSxNQUFNLENBQUMsSUFBSSxDQUFDLEdBQUcsRUFBRSxJQUFJLENBQUMsS0FBSyxDQUFDLENBQUM7SUFDNUMsQ0FBQztJQUVELFlBQVk7UUFDUixPQUFPLElBQUksU0FBUyxDQUFDLElBQUksQ0FBQyxHQUFHLEVBQUUsSUFBSSxDQUFDLEtBQUssQ0FBQyxDQUFDO0lBQy9DLENBQUM7SUFFRCxjQUFjO1FBQ1YsT0FBTyxJQUFJLFdBQVcsQ0FBQyxJQUFJLENBQUMsR0FBRyxFQUFFLElBQUksQ0FBQyxLQUFLLENBQUMsQ0FBQztJQUNqRCxDQUFDO0NBQ0o7QUFFRCxNQUFNLEtBQUs7SUFDUCxHQUFHLENBQVM7SUFDWixLQUFLLENBQVM7SUFFZCxZQUFZLEdBQVcsRUFBRSxLQUFhO1FBQ2xDLElBQUksQ0FBQyxHQUFHLEdBQUcsR0FBRyxDQUFDO1FBQ2YsSUFBSSxDQUFDLEtBQUssR0FBRyxLQUFLLENBQUM7SUFDdkIsQ0FBQztJQUVELEtBQUssQ0FBQyxHQUFHLENBQUMsR0FBVztRQUNqQixPQUFPLENBQUMsTUFBTSxJQUFJLENBQUMsU0FBUyxDQUFDLEtBQUssRUFBRSxHQUFHLENBQUMsQ0FBQyxDQUFDLEtBQUssQ0FBQztJQUNwRCxDQUFDO0lB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