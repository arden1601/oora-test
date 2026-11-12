package audit

import (
	"context"
	"log/slog"

	"staffportal/internal/store"
)

// Refused records one refused access: who, which page, when (set by the database)
// and a reason code. Only IDs and codes are written; never names or request bodies.
// staffID is empty when nobody was signed in.
func Refused(ctx context.Context, st *store.Store, staffID, page, action, reason string) {
	slog.Info("access refused", "staff_id", staffID, "page", page, "action", action, "reason", reason)
	if err := st.InsertAudit(ctx, staffID, page, action, reason); err != nil {
		// The request is still refused; the failure to record it must be visible.
		slog.Error("audit row not written", "page", page, "reason", reason, "error", err)
	}
}
