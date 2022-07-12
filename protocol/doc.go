// TODO

// Package protocol provides a object representation of a socket.io packet.
//
// This package currently supports socket.io version 1 to socket.io version 5. It provides
// classes and API to read and write socket.io wire format, as represented by:
//
//     <packet type>[<# of binary attachments>-][<namespace>,][<acknowledgment id>][JSON-stringified payload without binary]
//     [<binary attachment>]
//
// or as a real example:
//
//     1-/admin,456["project:delete",{"_placeholder":true,"num":0}]
//
// this is with the API:
//
//     Packet.Type
//     Packet.AckID
//     Packet.Namespace
//     Packet.Data
//
// This object is expected to be written and read directly to the connection. It's a streaming but transport agnostic.
package protocol

// What is this package for?
// Why does it exist?
// How does it work?
