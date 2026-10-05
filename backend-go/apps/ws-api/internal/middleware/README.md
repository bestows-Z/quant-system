# middleware

JWT validation is implemented in the WebSocket handler because websocket upgrade requests can pass tokens through either the `Authorization` header or the `token` query parameter.
