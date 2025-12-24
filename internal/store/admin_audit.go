package store

import (
	"context"
	"encoding/json"
)

func (s *Store) AddAdminAudit(
	ctx context.Context,
	adminUserID string,
	action string,
	targetType string,
	targetID string,
	meta map[string]any,
) error {

	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	_, err = s.DB.Exec(ctx, `
		INSERT INTO admin_audit_log
			(admin_user_id, action, target_type, target_id, meta)
		VALUES ($1, $2, $3, $4, $5)
	`,
		adminUserID,
		action,
		targetType,
		targetID,
		metaJSON,
	)

	return err
}
