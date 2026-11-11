// Package audit records every refused access: who, which page and when.
//
// Audit rows hold IDs and reason codes only. Never passwords, session tokens
// or customer data.
package audit
