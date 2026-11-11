// Package auth handles sign-in, password hashes and server-side sessions.
//
// Sessions and staff status are read from the database on every request, so a
// logout or a deactivation takes effect on the very next request.
package auth
