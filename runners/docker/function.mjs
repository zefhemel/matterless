async function init() {
    console.error("Not properly inited!");
}

function handle(event) {
    return event;
}

var _init;
try {
    _init = init;
} catch (e) {
    _init = () => {
    };
}

export {
    _init as init,
    handle
};
