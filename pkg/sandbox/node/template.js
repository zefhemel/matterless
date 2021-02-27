{{.Code}}

function getClient() {
	require('isomorphic-fetch');
	const Client4 = require('./mattermost-redux/mattermost.client4.js').default;
	const client = new Client4();
	client.setUrl(process.env.URL);
	client.setToken(process.env.TOKEN);
	return client;
}

let result = handler({{.Event}});
if(result !== undefined) {
	console.error(JSON.stringify(result));
} else {
	console.error(JSON.stringify({}));
}