// Package js executes self-contained Veta JavaScript files synchronously.
//
// A Veta JavaScript file is intentionally not a general JavaScript module. It
// must default-export a function, may not import other files or packages, and is
// invoked with the runtime context object as its only argument.
package js
