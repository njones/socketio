//go:build gc || eio_svr_v5
// +build gc eio_svr_v5

package engineio

const Version5 EIOVersionStr = "5"

func init() { registery[Version5.Int()] = NewServerV5 }

// https://github.com/socketio/engine.io/tree/fe5d97fc3d7a26d34bce786a97962fae3d7ce17f
// https://github.com/socketio/engine.io/compare/3.5.x...4.1.x

/*
- `initial_headers`
    - Fired on the first request of the connection, before writing the response headers
    - **Arguments**
      - `headers` (`Object`): a hash of headers
      - `req` (`http.IncomingMessage`): the request

- `headers`
    - Fired on the all requests of the connection, before writing the response headers
    - **Arguments**
      - `headers` (`Object`): a hash of headers
      - `req` (`http.IncomingMessage`): the request

- `connection_error`
    - Fired when an error occurs when establishing the connection.
    - **Arguments**
      - `error`: an object with following properties:
        - `req` (`http.IncomingMessage`): the request that was dropped
        - `code` (`Number`): one of `Server.errors`
        - `message` (`string`): one of `Server.errorMessages`
        - `context` (`Object`): extra info about the error

| Code | Message |
| ---- | ------- |
| 0 | "Transport unknown"
| 1 | "Session ID unknown"
| 2 | "Bad handshake method"
| 3 | "Bad request"
| 4 | "Forbidden"
| 5 | "Unsupported protocol version"
*/
type serverV5 struct {
	*serverV4
}

func NewServerV5(opts ...Option) Server { return (&serverV5{}).new(opts...) }

func (v5 *serverV5) new(opts ...Option) *serverV5 {
	v5.serverV4 = (&serverV4{}).new(opts...)

	v5.With(v5, opts...)
	return v5
}
