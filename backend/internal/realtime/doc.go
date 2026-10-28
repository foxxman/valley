// Package realtime is a session message bus.
//
// A connection joins rooms and receives envelopes addressed to those rooms.
// The bus does not interpret payloads: game movement, chat, and later features
// register their own message types on Router.
package realtime
