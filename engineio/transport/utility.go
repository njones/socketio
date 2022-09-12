package transport

// TODO(njones): fix this so there is an error side channel that can be used.
// This error will show up in the socketio server through the following trail.
//
//  1. The error is a eio SocketClose{error}
//  2. The eio transport sends this in a NOOP packet back to the sio recieve channel
//  3. The sio recieve transport takes the erorr and packages it in a
//     sio packet data as a readWriteError{error} type, this is so that it can pass
//     through the sio Data packet io.ReadWriter interface
//  4. The ReadWriteError is packed as a ErrorPacket for the sio protocol to consume as
//     a packet.
//  5. The erorr from the Data interface of the ErrorPacket type is then returned
//     back through the sio server .run() method
//  6. Finally the sio ServeHTTP method picks up the returned error and decides how to
//     exit as a 200, 400 or 500 HTTP status code.
//
// The above is much to complicated and should be simplified at the first chance.
// This is an internal construct, and not used by users of the SocketIO library so
// it can wait until there is a clear proposal on how this can work cleaner.

type socketClose struct{ error }

func (sc socketClose) SocketCloseChannel() error { return sc.error }

type WriteClose struct{ error }

func (wc WriteClose) SocketCloseChannel() error { return wc.error }
