export class Store {
    async get(key : string) : Promise<any> {
        return (await this.performOp("get", key)).value;
    }

    async put(key : string, value : any) {
        await this.performOp("put", key, value);
    }

    async del(key : string) {
        await this.performOp("del", key);
    }

    async queryPrefix(prefix : string) : Promise<[[string, any]]> {
        return (await this.performOp("query-prefix", prefix)).results;
    }

    async performOp(...args : any[]) : Promise<any> {
        let result = await fetch(`${Deno.env.get("API_URL")}/_store`, {
            method: "POST",
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `bearer ${Deno.env.get("API_TOKEN")}`
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

export async function publishEvent(eventName : string, eventData : object) {
    let result = await fetch(`${Deno.env.get("API_URL")}/_event/${eventName}`, {
        method: "POST",
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `bearer ${Deno.env.get("API_TOKEN")}`
        },
        body: JSON.stringify(eventData)
    })
    let jsonResult = await result.json();
    if(jsonResult.status === "error") {
        throw Error(jsonResult.error);
    }
}

export async function respondToEvent(toEventData : any, eventData : any) {
    let result = await fetch(`${Deno.env.get("API_URL")}/_event/${toEventData['$response_event']}`, {
        method: "POST",
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `bearer ${Deno.env.get("API_TOKEN")}`
        },
        body: JSON.stringify(eventData)
    })
    if(result.status == 200) {
        let jsonResult = await result.json();
        if(jsonResult.status === "error") {
            throw Error(jsonResult.error);
        }
    } else {
        throw new Error(`HTTP request not ok: ${await result.text()}`);
    }
}

export async function invokeFunction(name : string, eventData : any) {
    let result = await fetch(`${Deno.env.get("API_URL")}/_function/${name}`, {
        method: "POST",
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `bearer ${Deno.env.get("API_TOKEN")}`
        },
        body: JSON.stringify(eventData || {})
    })
    if(result.status == 200) {
        let jsonResult = await result.json();
        if(jsonResult.status === "error") {
            throw Error(jsonResult.error);
        }
        return jsonResult;
    } else {
        throw new Error(`HTTP request not ok: ${await result.text()}`);
    }
}
