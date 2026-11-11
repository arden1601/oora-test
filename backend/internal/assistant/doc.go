// Package assistant is the AI assistant, backed by the DeepSeek API.
//
// The assistant has no access of its own. Each tool call runs through the same
// access decision and branch scope as the signed-in person, and results are
// de-identified before they are sent to the model.
package assistant
