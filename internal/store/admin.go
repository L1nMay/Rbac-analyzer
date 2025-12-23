package store

import (
	"context"
	"time"
)

type AdminUserRow struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	IsAdmin   bool      `json:"isAdmin"`
	CreatedAt time.Time `json:"createdAt"`
}

type AdminOrgRow struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	OwnerUserID string    `json:"ownerUserId"`
	OwnerEmail  string    `json:"ownerEmail"`
	PlanID      string    `json:"planId"`
	Status      string    `json:"status"`
	MaxClusters int       `json:"maxClusters"`
	ClustersCnt int       `json:"clustersCount"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (s *Store) AdminListUsers(ctx context.Context, limit int) ([]AdminUserRow, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}

	rows, err := s.DB.Query(ctx, `
		SELECT id, email, is_admin, created_at
		FROM users
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]AdminUserRow, 0, 64)
	for rows.Next() {
		var r AdminUserRow
		if err := rows.Scan(&r.ID, &r.Email, &r.IsAdmin, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) AdminSetUserAdmin(ctx context.Context, userID string, isAdmin bool) error {
	_, err := s.DB.Exec(ctx, `
		UPDATE users
		SET is_admin = $2
		WHERE id = $1
	`, userID, isAdmin)
	return err
}

func (s *Store) AdminListOrgs(ctx context.Context, limit int) ([]AdminOrgRow, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}

	rows, err := s.DB.Query(ctx, `
		SELECT
			o.id,
			o.name,
			o.owner_user_id,
			COALESCE(u.email, '') as owner_email,
			s.plan_id,
			s.status,
			p.max_clusters,
			(SELECT COUNT(*) FROM clusters c WHERE c.org_id=o.id) as clusters_cnt,
			o.created_at
		FROM orgs o
		LEFT JOIN users u ON u.id = o.owner_user_id
		LEFT JOIN subscriptions s ON s.org_id = o.id
		LEFT JOIN plans p ON p.id = s.plan_id
		ORDER BY o.created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]AdminOrgRow, 0, 64)
	for rows.Next() {
		var r AdminOrgRow
		if err := rows.Scan(
			&r.ID,
			&r.Name,
			&r.OwnerUserID,
			&r.OwnerEmail,
			&r.PlanID,
			&r.Status,
			&r.MaxClusters,
			&r.ClustersCnt,
			&r.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) AdminSetOrgPlan(ctx context.Context, orgID string, planID string) error {
	_, err := s.DB.Exec(ctx, `
		UPDATE subscriptions
		SET plan_id = $2
		WHERE org_id = $1
	`, orgID, planID)
	return err
}
