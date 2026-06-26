// Package jsruntime executes self-contained Veta JavaScript files synchronously.
//
// A Veta JavaScript file is intentionally not a general JavaScript module. It
// must default-export a function, may not import other files or packages, and is
// invoked with the Veta runtime object as its only argument. The same runtime
// object is also available as the global Veta value.
package jsruntime
