{{.Code}}

let result = handle({{.Event}});

if (result === undefined) {
    console.error(JSON.stringify({}));
} else if (result.then) {
    result.then((result) => {
        if (result === undefined) {
            console.error(JSON.stringify({}));
        } else {
            console.error(JSON.stringify(result));
        }
    });
} else {
    console.error(JSON.stringify(result));
}