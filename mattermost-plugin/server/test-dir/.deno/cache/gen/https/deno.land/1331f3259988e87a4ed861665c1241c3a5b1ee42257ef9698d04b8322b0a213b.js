export var Status;
(function (Status) {
    Status[Status["Continue"] = 100] = "Continue";
    Status[Status["SwitchingProtocols"] = 101] = "SwitchingProtocols";
    Status[Status["Processing"] = 102] = "Processing";
    Status[Status["EarlyHints"] = 103] = "EarlyHints";
    Status[Status["OK"] = 200] = "OK";
    Status[Status["Created"] = 201] = "Created";
    Status[Status["Accepted"] = 202] = "Accepted";
    Status[Status["NonAuthoritativeInfo"] = 203] = "NonAuthoritativeInfo";
    Status[Status["NoContent"] = 204] = "NoContent";
    Status[Status["ResetContent"] = 205] = "ResetContent";
    Status[Status["PartialContent"] = 206] = "PartialContent";
    Status[Status["MultiStatus"] = 207] = "MultiStatus";
    Status[Status["AlreadyReported"] = 208] = "AlreadyReported";
    Status[Status["IMUsed"] = 226] = "IMUsed";
    Status[Status["MultipleChoices"] = 300] = "MultipleChoices";
    Status[Status["MovedPermanently"] = 301] = "MovedPermanently";
    Status[Status["Found"] = 302] = "Found";
    Status[Status["SeeOther"] = 303] = "SeeOther";
    Status[Status["NotModified"] = 304] = "NotModified";
    Status[Status["UseProxy"] = 305] = "UseProxy";
    Status[Status["TemporaryRedirect"] = 307] = "TemporaryRedirect";
    Status[Status["PermanentRedirect"] = 308] = "PermanentRedirect";
    Status[Status["BadRequest"] = 400] = "BadRequest";
    Status[Status["Unauthorized"] = 401] = "Unauthorized";
    Status[Status["PaymentRequired"] = 402] = "PaymentRequired";
    Status[Status["Forbidden"] = 403] = "Forbidden";
    Status[Status["NotFound"] = 404] = "NotFound";
    Status[Status["MethodNotAllowed"] = 405] = "MethodNotAllowed";
    Status[Status["NotAcceptable"] = 406] = "NotAcceptable";
    Status[Status["ProxyAuthRequired"] = 407] = "ProxyAuthRequired";
    Status[Status["RequestTimeout"] = 408] = "RequestTimeout";
    Status[Status["Conflict"] = 409] = "Conflict";
    Status[Status["Gone"] = 410] = "Gone";
    Status[Status["LengthRequired"] = 411] = "LengthRequired";
    Status[Status["PreconditionFailed"] = 412] = "PreconditionFailed";
    Status[Status["RequestEntityTooLarge"] = 413] = "RequestEntityTooLarge";
    Status[Status["RequestURITooLong"] = 414] = "RequestURITooLong";
    Status[Status["UnsupportedMediaType"] = 415] = "UnsupportedMediaType";
    Status[Status["RequestedRangeNotSatisfiable"] = 416] = "RequestedRangeNotSatisfiable";
    Status[Status["ExpectationFailed"] = 417] = "ExpectationFailed";
    Status[Status["Teapot"] = 418] = "Teapot";
    Status[Status["MisdirectedRequest"] = 421] = "MisdirectedRequest";
    Status[Status["UnprocessableEntity"] = 422] = "UnprocessableEntity";
    Status[Status["Locked"] = 423] = "Locked";
    Status[Status["FailedDependency"] = 424] = "FailedDependency";
    Status[Status["TooEarly"] = 425] = "TooEarly";
    Status[Status["UpgradeRequired"] = 426] = "UpgradeRequired";
    Status[Status["PreconditionRequired"] = 428] = "PreconditionRequired";
    Status[Status["TooManyRequests"] = 429] = "TooManyRequests";
    Status[Status["RequestHeaderFieldsTooLarge"] = 431] = "RequestHeaderFieldsTooLarge";
    Status[Status["UnavailableForLegalReasons"] = 451] = "UnavailableForLegalReasons";
    Status[Status["InternalServerError"] = 500] = "InternalServerError";
    Status[Status["NotImplemented"] = 501] = "NotImplemented";
    Status[Status["BadGateway"] = 502] = "BadGateway";
    Status[Status["ServiceUnavailable"] = 503] = "ServiceUnavailable";
    Status[Status["GatewayTimeout"] = 504] = "GatewayTimeout";
    Status[Status["HTTPVersionNotSupported"] = 505] = "HTTPVersionNotSupported";
    Status[Status["VariantAlsoNegotiates"] = 506] = "VariantAlsoNegotiates";
    Status[Status["InsufficientStorage"] = 507] = "InsufficientStorage";
    Status[Status["LoopDetected"] = 508] = "LoopDetected";
    Status[Status["NotExtended"] = 510] = "NotExtended";
    Status[Status["NetworkAuthenticationRequired"] = 511] = "NetworkAuthenticationRequired";
})(Status || (Status = {}));
export const STATUS_TEXT = new Map([
    [Status.Continue, "Continue"],
    [Status.SwitchingProtocols, "Switching Protocols"],
    [Status.Processing, "Processing"],
    [Status.EarlyHints, "Early Hints"],
    [Status.OK, "OK"],
    [Status.Created, "Created"],
    [Status.Accepted, "Accepted"],
    [Status.NonAuthoritativeInfo, "Non-Authoritative Information"],
    [Status.NoContent, "No Content"],
    [Status.ResetContent, "Reset Content"],
    [Status.PartialContent, "Partial Content"],
    [Status.MultiStatus, "Multi-Status"],
    [Status.AlreadyReported, "Already Reported"],
    [Status.IMUsed, "IM Used"],
    [Status.MultipleChoices, "Multiple Choices"],
    [Status.MovedPermanently, "Moved Permanently"],
    [Status.Found, "Found"],
    [Status.SeeOther, "See Other"],
    [Status.NotModified, "Not Modified"],
    [Status.UseProxy, "Use Proxy"],
    [Status.TemporaryRedirect, "Temporary Redirect"],
    [Status.PermanentRedirect, "Permanent Redirect"],
    [Status.BadRequest, "Bad Request"],
    [Status.Unauthorized, "Unauthorized"],
    [Status.PaymentRequired, "Payment Required"],
    [Status.Forbidden, "Forbidden"],
    [Status.NotFound, "Not Found"],
    [Status.MethodNotAllowed, "Method Not Allowed"],
    [Status.NotAcceptable, "Not Acceptable"],
    [Status.ProxyAuthRequired, "Proxy Authentication Required"],
    [Status.RequestTimeout, "Request Timeout"],
    [Status.Conflict, "Conflict"],
    [Status.Gone, "Gone"],
    [Status.LengthRequired, "Length Required"],
    [Status.PreconditionFailed, "Precondition Failed"],
    [Status.RequestEntityTooLarge, "Request Entity Too Large"],
    [Status.RequestURITooLong, "Request URI Too Long"],
    [Status.UnsupportedMediaType, "Unsupported Media Type"],
    [Status.RequestedRangeNotSatisfiable, "Requested Range Not Satisfiable"],
    [Status.ExpectationFailed, "Expectation Failed"],
    [Status.Teapot, "I'm a teapot"],
    [Status.MisdirectedRequest, "Misdirected Request"],
    [Status.UnprocessableEntity, "Unprocessable Entity"],
    [Status.Locked, "Locked"],
    [Status.FailedDependency, "Failed Dependency"],
    [Status.TooEarly, "Too Early"],
    [Status.UpgradeRequired, "Upgrade Required"],
    [Status.PreconditionRequired, "Precondition Required"],
    [Status.TooManyRequests, "Too Many Requests"],
    [Status.RequestHeaderFieldsTooLarge, "Request Header Fields Too Large"],
    [Status.UnavailableForLegalReasons, "Unavailable For Legal Reasons"],
    [Status.InternalServerError, "Internal Server Error"],
    [Status.NotImplemented, "Not Implemented"],
    [Status.BadGateway, "Bad Gateway"],
    [Status.ServiceUnavailable, "Service Unavailable"],
    [Status.GatewayTimeout, "Gateway Timeout"],
    [Status.HTTPVersionNotSupported, "HTTP Version Not Supported"],
    [Status.VariantAlsoNegotiates, "Variant Also Negotiates"],
    [Status.InsufficientStorage, "Insufficient Storage"],
    [Status.LoopDetected, "Loop Detected"],
    [Status.NotExtended, "Not Extended"],
    [Status.NetworkAuthenticationRequired, "Network Authentication Required"],
]);
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiaHR0cF9zdGF0dXMuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJodHRwX3N0YXR1cy50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFHQSxNQUFNLENBQU4sSUFBWSxNQWdJWDtBQWhJRCxXQUFZLE1BQU07SUFFaEIsNkNBQWMsQ0FBQTtJQUVkLGlFQUF3QixDQUFBO0lBRXhCLGlEQUFnQixDQUFBO0lBRWhCLGlEQUFnQixDQUFBO0lBRWhCLGlDQUFRLENBQUE7SUFFUiwyQ0FBYSxDQUFBO0lBRWIsNkNBQWMsQ0FBQTtJQUVkLHFFQUEwQixDQUFBO0lBRTFCLCtDQUFlLENBQUE7SUFFZixxREFBa0IsQ0FBQTtJQUVsQix5REFBb0IsQ0FBQTtJQUVwQixtREFBaUIsQ0FBQTtJQUVqQiwyREFBcUIsQ0FBQTtJQUVyQix5Q0FBWSxDQUFBO0lBR1osMkRBQXFCLENBQUE7SUFFckIsNkRBQXNCLENBQUE7SUFFdEIsdUNBQVcsQ0FBQTtJQUVYLDZDQUFjLENBQUE7SUFFZCxtREFBaUIsQ0FBQTtJQUVqQiw2Q0FBYyxDQUFBO0lBRWQsK0RBQXVCLENBQUE7SUFFdkIsK0RBQXVCLENBQUE7SUFHdkIsaURBQWdCLENBQUE7SUFFaEIscURBQWtCLENBQUE7SUFFbEIsMkRBQXFCLENBQUE7SUFFckIsK0NBQWUsQ0FBQTtJQUVmLDZDQUFjLENBQUE7SUFFZCw2REFBc0IsQ0FBQTtJQUV0Qix1REFBbUIsQ0FBQTtJQUVuQiwrREFBdUIsQ0FBQTtJQUV2Qix5REFBb0IsQ0FBQTtJQUVwQiw2Q0FBYyxDQUFBO0lBRWQscUNBQVUsQ0FBQTtJQUVWLHlEQUFvQixDQUFBO0lBRXBCLGlFQUF3QixDQUFBO0lBRXhCLHVFQUEyQixDQUFBO0lBRTNCLCtEQUF1QixDQUFBO0lBRXZCLHFFQUEwQixDQUFBO0lBRTFCLHFGQUFrQyxDQUFBO0lBRWxDLCtEQUF1QixDQUFBO0lBRXZCLHlDQUFZLENBQUE7SUFFWixpRUFBd0IsQ0FBQTtJQUV4QixtRUFBeUIsQ0FBQTtJQUV6Qix5Q0FBWSxDQUFBO0lBRVosNkRBQXNCLENBQUE7SUFFdEIsNkNBQWMsQ0FBQTtJQUVkLDJEQUFxQixDQUFBO0lBRXJCLHFFQUEwQixDQUFBO0lBRTFCLDJEQUFxQixDQUFBO0lBRXJCLG1GQUFpQyxDQUFBO0lBRWpDLGlGQUFnQyxDQUFBO0lBR2hDLG1FQUF5QixDQUFBO0lBRXpCLHlEQUFvQixDQUFBO0lBRXBCLGlEQUFnQixDQUFBO0lBRWhCLGlFQUF3QixDQUFBO0lBRXhCLHlEQUFvQixDQUFBO0lBRXBCLDJFQUE2QixDQUFBO0lBRTdCLHVFQUEyQixDQUFBO0lBRTNCLG1FQUF5QixDQUFBO0lBRXpCLHFEQUFrQixDQUFBO0lBRWxCLG1EQUFpQixDQUFBO0lBRWpCLHVGQUFtQyxDQUFBO0FBQ3JDLENBQUMsRUFoSVcsTUFBTSxLQUFOLE1BQU0sUUFnSWpCO0FBRUQsTUFBTSxDQUFDLE1BQU0sV0FBVyxHQUFHLElBQUksR0FBRyxDQUFpQjtJQUNqRCxDQUFDLE1BQU0sQ0FBQyxRQUFRLEVBQUUsVUFBVSxDQUFDO0lBQzdCLENBQUMsTUFBTSxDQUFDLGtCQUFrQixFQUFFLHFCQUFxQixDQUFDO0lBQ2xELENBQUMsTUFBTSxDQUFDLFVBQVUsRUFBRSxZQUFZLENBQUM7SUFDakMsQ0FBQyxNQUFNLENBQUMsVUFBVSxFQUFFLGFBQWEsQ0FBQztJQUNsQyxDQUFDLE1BQU0sQ0FBQyxFQUFFLEVBQUUsSUFBSSxDQUFDO0lBQ2pCLENBQUMsTUFBTSxDQUFDLE9BQU8sRUFBRSxTQUFTLENBQUM7SUFDM0IsQ0FBQyxNQUFNLENBQUMsUUFBUSxFQUFFLFVBQVUsQ0FBQztJQUM3QixDQUFDLE1BQU0sQ0FBQyxvQkFBb0IsRUFBRSwrQkFBK0IsQ0FBQztJQUM5RCxDQUFDLE1BQU0sQ0FBQyxTQUFTLEVBQUUsWUFBWSxDQUFDO0lBQ2hDLENBQUMsTUFBTSxDQUFDLFlBQVksRUFBRSxlQUFlLENBQUM7SUFDdEMsQ0FBQyxNQUFNLENBQUMsY0FBYyxFQUFFLGlCQUFpQixDQUFDO0lBQzFDLENBQUMsTUFBTSxDQUFDLFdBQVcsRUFBRSxjQUFjLENBQUM7SUFDcEMsQ0FBQyxNQUFNLENBQUMsZUFBZSxFQUFFLGtCQUFrQixDQUFDO0lBQzVDLENBQUMsTUFBTSxDQUFDLE1BQU0sRUFBRSxTQUFTLENBQUM7SUFDMUIsQ0FBQyxNQUFNLENBQUMsZUFBZSxFQUFFLGtCQUFrQixDQUFDO0lBQzVDLENBQUMsTUFBTSxDQUFDLGdCQUFnQixFQUFFLG1CQUFtQixDQUFDO0lBQzlDLENBQUMsTUFBTSxDQUFDLEtBQUssRUFBRSxPQUFPLENBQUM7SUFDdkIsQ0FBQyxNQUFNLENBQUMsUUFBUSxFQUFFLFdBQVcsQ0FBQztJQUM5QixDQUFDLE1BQU0sQ0FBQyxXQUFXLEVBQUUsY0FBYyxDQUFDO0lBQ3BDLENBQUMsTUFBTSxDQUFDLFFBQVEsRUFBRSxXQUFXLENBQUM7SUFDOUIsQ0FBQyxNQUFNLENBQUMsaUJBQWlCLEVBQUUsb0JBQW9CLENBQUM7SUFDaEQsQ0FBQyxNQUFNLENBQUMsaUJBQWlCLEVBQUUsb0JBQW9CLENBQUM7SUFDaEQsQ0FBQyxNQUFNLENBQUMsVUFBVSxFQUFFLGFBQWEsQ0FBQztJQUNsQyxDQUFDLE1BQU0sQ0FBQyxZQUFZLEVBQUUsY0FBYyxDQUFDO0lBQ3JDLENBQUMsTUFBTSxDQUFDLGVBQWUsRUFBRSxrQkFBa0IsQ0FBQztJQUM1QyxDQUFDLE1BQU0sQ0FBQyxTQUFTLEVBQUUsV0FBVyxDQUFDO0lBQy9CLENBQUMsTUFBTSxDQUFDLFFBQVEsRUFBRSxXQUFXLENBQUM7SUFDOUIsQ0FBQyxNQUFNLENBQUMsZ0JBQWdCLEVBQUUsb0JBQW9CLENBQUM7SUFDL0MsQ0FBQyxNQUFNLENBQUMsYUFBYSxFQUFFLGdCQUFnQixDQUFDO0lBQ3hDLENBQUMsTUFBTSxDQUFDLGlCQUFpQixFQUFFLCtCQUErQixDQUFDO0lBQzNELENBQUMsTUFBTSxDQUFDLGNBQWMsRUFBRSxpQkFBaUIsQ0FBQztJQUMxQyxDQUFDLE1BQU0sQ0FBQyxRQUFRLEVBQUUsVUFBVSxDQUFDO0lBQzdCLENBQUMsTUFBTSxDQUFDLElBQUksRUFBRSxNQUFNLENBQUM7SUFDckIsQ0FBQyxNQUFNLENBQUMsY0FBYyxFQUFFLGlCQUFpQixDQUFDO0lBQzFDLENBQUMsTUFBTSxDQUFDLGtCQUFrQixFQUFFLHFCQUFxQixDQUFDO0lBQ2xELENBQUMsTUFBTSxDQUFDLHFCQUFxQixFQUFFLDBCQUEwQixDQUFDO0lBQzFELENBQUMsTUFBTSxDQUFDLGlCQUFpQixFQUFFLHNCQUFzQixDQUFDO0lBQ2xELENBQUMsTUFBTSxDQUFDLG9CQUFvQixFQUFFLHdCQUF3QixDQUFDO0lBQ3ZELENBQUMsTUFBTSxDQUFDLDRCQUE0QixFQUFFLGlDQUFpQyxDQUFDO0lBQ3hFLENBQUMsTUFBTSxDQUFDLGlCQUFpQixFQUFFLG9CQUFvQixDQUFDO0lBQ2hELENBQUMsTUFBTSxDQUFDLE1BQU0sRUFBRSxjQUFjLENBQUM7SUFDL0IsQ0FBQyxNQUFNLENBQUMsa0JBQWtCLEVBQUUscUJBQXFCLENBQUM7SUFDbEQsQ0FBQyxNQUFNLENBQUMsbUJBQW1CLEVBQUUsc0JBQXNCLENBQUM7SUFDcEQsQ0FBQyxNQUFNLENBQUMsTUFBTSxFQUFFLFFBQVEsQ0FBQztJQUN6QixDQUFDLE1BQU0sQ0FBQyxnQkFBZ0IsRUFBRSxtQkFBbUIsQ0FBQztJQUM5QyxDQUFDLE1BQU0sQ0FBQyxRQUFRLEVBQUUsV0FBVyxDQUFDO0lBQzlCLENBQUMsTUFBTSxDQUFDLGVBQWUsRUFBRSxrQkFBa0IsQ0FBQztJQUM1QyxDQUFDLE1BQU0sQ0FBQyxvQkFBb0IsRUFBRSx1QkFBdUIsQ0FBQztJQUN0RCxDQUFDLE1BQU0sQ0FBQyxlQUFlLEVBQUUsbUJBQW1CLENBQUM7SUFDN0MsQ0FBQyxNQUFNLENBQUMsMkJBQTJCLEVBQUUsaUNBQWlDLENBQUM7SUFDdkUsQ0FBQyxNQUFNLENBQUMsMEJBQTBCLEVBQUUsK0JBQStCLENBQUM7SUFDcEUsQ0FBQyxNQUFNLENBQUMsbUJBQW1CLEVBQUUsdUJBQXVCLENBQUM7SUFDckQsQ0FBQyxNQUFNLENBQUMsY0FBYyxFQUFFLGlCQUFpQixDQUFDO0lBQzFDLENBQUMsTUFBTSxDQUFDLFVBQVUsRUFBRSxhQUFhLENBQUM7SUFDbEMsQ0FBQyxNQUFNLENBQUMsa0JBQWtCLEVBQUUscUJBQXFCLENBQUM7SUFDbEQsQ0FBQyxNQUFNLENBQUMsY0FBYyxFQUFFLGlCQUFpQixDQUFDO0lBQzFDLENBQUMsTUFBTSxDQUFDLHVCQUF1QixFQUFFLDRCQUE0QixDQUFDO0lBQzlELENBQUMsTUFBTSxDQUFDLHFCQUFxQixFQUFFLHlCQUF5QixDQUFDO0lBQ3pELENBQUMsTUFBTSxDQUFDLG1CQUFtQixFQUFFLHNCQUFzQixDQUFDO0lBQ3BELENBQUMsTUFBTSxDQUFDLFlBQVksRUFBRSxlQUFlLENBQUM7SUFDdEMsQ0FBQyxNQUFNLENBQUMsV0FBVyxFQUFFLGNBQWMsQ0FBQztJQUNwQyxDQUFDLE1BQU0sQ0FBQyw2QkFBNkIsRUFBRSxpQ0FBaUMsQ0FBQztDQUMxRSxDQUFDLENBQUMifQ==