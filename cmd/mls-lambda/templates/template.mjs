{{.Code}}

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
