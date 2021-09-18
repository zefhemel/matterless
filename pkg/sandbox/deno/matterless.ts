class API {
    url: string;
    token: string;

    constructor(url: string, token: string) {
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
    url: string;
    token: string;

    constructor(url: string, token: string) {
        this.url = url;
        this.token = token;
    }

    async get(key: string): Promise<any> {
        return (await this.performOp("get", key)).value;
    }

    async put(key: string, value: any) {
        await this.performOp("put", key, value);
    }

    async del(key: string) {
        await this.performOp("del", key);
    }

    async queryPrefix(prefix: string): Promise<[[string, any]]> {
        return (await this.performOp("query-prefix", prefix)).results || [];
    }

    async performOp(...args: any[]): Promise<any> {
        let result = await fetch(`${this.url}/_store`, {
            method: "POST",
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `bearer ${this.token}`
            },
            body: JSON.stringify([args])
        })
        let jsonResult = (await result.json())[0];
        if (jsonResult.status === "error") {
            throw Error(jsonResult.error);
        }
        return jsonResult;
    }
}

class Events {
    url: string;
    token: string;

    constructor(url: string, token: string) {
        this.url = url;
        this.token = token;
    }

    async publish(eventName: string, eventData: object) {
        let result = await fetch(`${this.url}/_event/${eventName}`, {
            method: "POST",
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `bearer ${this.token}`
            },
            body: JSON.stringify(eventData || {})
        })
        let jsonResult = await result.json();
        if (jsonResult.status === "error") {
            throw Error(jsonResult.error);
        }
    }
}

class Functions {
    url: string;
    token: string;

    constructor(url: string, token: string) {
        this.url = url;
        this.token = token;
    }

    async invoke(name: string, eventData: any) {
        let result = await fetch(`${this.url}/_function/${name}`, {
            method: "POST",
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `bearer ${this.token}`
            },
            body: JSON.stringify(eventData || {})
        })
        if (result.status == 200) {
            let jsonResult = await result.json();
            if (jsonResult.status === "error") {
                throw Error(jsonResult.error);
            }
            return jsonResult;
        } else {
            throw new Error(`HTTP request not ok: ${await result.text()}`);
        }
    }
}

class Application {
    url: string;
    token: string;

    constructor(url: string, token: string) {
        this.url = url;
        this.token = token;
    }

    async restart() {
        let result = await fetch(`${this.url}/_restart`, {
            method: "POST",
            headers: {
                'Authorization': `bearer ${this.token}`
            },
        })
        if (result.status != 200) {
            throw new Error(`HTTP request not ok: ${await result.text()}`);
        }
    }
}

// @ts-ignore
const defaultApi = new API(Deno.env.get("API_URL")!, Deno.env.get("API_TOKEN")!),
    store = defaultApi.getStore(),
    events = defaultApi.getEvents(),
    functions = defaultApi.getFunctions(),
    application = defaultApi.getApplication();


export {
    store,
    events,
    functions,
    application,
    API
}
