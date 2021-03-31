import {Mattermost} from "./mattermost.js";

let client = new Mattermost("http://pi-jay:8065", "cu7f3goontys8ctra5nd8hy59y");
let me = await client.getMe();
let teams = await client.getUserTeams(me.id);
let testChannel = await client.getChannelByName(teams[0].id, "test");
console.log(testChannel)
let p = await client.createPost({
    channel_id: testChannel!.id,
    message: "Testing this!"
});
p.message = "Testing this edited";
await client.updatePost(p);
console.log((await client.deletePost(p)).status);
