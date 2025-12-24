package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type User struct {
	ID           string
	Email        string
	PasswordHash string
	IsAdmin      bool
	CreatedAt    time.Time
}

type Org struct {
	ID   string
	Name string
}

type Cluster struct {
	ID        string
	OrgID     string
	Name      string
	Notes     string
	CreatedAt time.Time
}

type Scan struct {
	ID        string
	OrgID     string
	ClusterID string
	CreatedAt time.Time
	Source    string
}

type Subscription struct {
	OrgID                string
	PlanID               string
	Status               string
	StripeCustomerID     string
	StripeSubscriptionID string
	CurrentPeriodEnd     *time.Time
}
type OrgAdminView struct {
	OrgID       string
	OrgName     string
	OwnerEmail  string
	PlanID      string
	ClustersCnt int
}

func (s *Store) ListOrgsAdmin(ctx context.Context) ([]OrgAdminView, error) {
	rows, err := s.DB.Query(ctx,
		`SELECT
			o.id,
			o.name,
			u.email,
			s.plan_id,
			(SELECT COUNT(*) FROM clusters c WHERE c.org_id=o.id)
		FROM orgs o
		JOIN users u ON u.id=o.owner_user_id
		JOIN subscriptions s ON s.org_id=o.id
		ORDER BY o.created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []OrgAdminView
	for rows.Next() {
		var r OrgAdminView
		if err := rows.Scan(&r.OrgID, &r.OrgName, &r.OwnerEmail, &r.PlanID, &r.ClustersCnt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) SetOrgPlan(ctx context.Context, orgID, planID string) error {
	_, err := s.DB.Exec(ctx,
		`UPDATE subscriptions SET plan_id=$1 WHERE org_id=$2`,
		planID, orgID,
	)
	return err
}

func (s *Store) CreateUser(ctx context.Context, email, passwordHash string) (User, error) {
	var u User
	err := s.DB.QueryRow(ctx,
		`INSERT INTO users(email, password_hash)
		 VALUES($1,$2)
		 RETURNING id, email, password_hash, is_admin, created_at`,
		email, passwordHash,
	).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.IsAdmin,
		&u.CreatedAt,
	)
	return u, err
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := s.DB.QueryRow(ctx,
		`SELECT id, email, password_hash, is_admin, created_at FROM users WHERE email=$1`,
		email,
	).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.IsAdmin,
		&u.CreatedAt,
	)
	return u, err
}

func (s *Store) CreateOrgForOwner(ctx context.Context, ownerUserID, orgName string) (Org, error) {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return Org{}, err
	}
	defer tx.Rollback(ctx)

	var org Org
	if err := tx.QueryRow(ctx,
		`INSERT INTO orgs(name, owner_user_id) VALUES($1,$2) RETURNING id, name`,
		orgName, ownerUserID,
	).Scan(&org.ID, &org.Name); err != nil {
		return Org{}, err
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO org_members(org_id, user_id, role) VALUES($1,$2,'owner')`,
		org.ID, ownerUserID,
	); err != nil {
		return Org{}, err
	}

	// по умолчанию подписка free
	_, _ = tx.Exec(ctx,
		`INSERT INTO subscriptions(org_id, plan_id, status) VALUES($1,'free','active')
		 ON CONFLICT (org_id) DO NOTHING`,
		org.ID,
	)

	if err := tx.Commit(ctx); err != nil {
		return Org{}, err
	}
	return org, nil
}

func (s *Store) GetOwnerOrg(ctx context.Context, userID string) (Org, error) {
	var org Org
	err := s.DB.QueryRow(ctx,
		`SELECT o.id, o.name
		 FROM orgs o
		 JOIN org_members m ON m.org_id=o.id
		 WHERE m.user_id=$1
		 ORDER BY o.created_at ASC
		 LIMIT 1`,
		userID,
	).Scan(&org.ID, &org.Name)
	return org, err
}

func (s *Store) GetSubscription(ctx context.Context, orgID string) (Subscription, error) {
	var sub Subscription
	err := s.DB.QueryRow(ctx,
		`SELECT org_id, plan_id, status, stripe_customer_id, stripe_subscription_id, current_period_end
		 FROM subscriptions WHERE org_id=$1`,
		orgID,
	).Scan(&sub.OrgID, &sub.PlanID, &sub.Status, &sub.StripeCustomerID, &sub.StripeSubscriptionID, &sub.CurrentPeriodEnd)
	return sub, err
}

func (s *Store) PlanMaxClusters(ctx context.Context, planID string) (int, error) {
	var max int
	err := s.DB.QueryRow(ctx, `SELECT max_clusters FROM plans WHERE id=$1`, planID).Scan(&max)
	return max, err
}

func (s *Store) CountClusters(ctx context.Context, orgID string) (int, error) {
	var c int
	err := s.DB.QueryRow(ctx, `SELECT COUNT(*) FROM clusters WHERE org_id=$1`, orgID).Scan(&c)
	return c, err
}

func (s *Store) CreateCluster(ctx context.Context, orgID, name, notes string) (Cluster, error) {
	var c Cluster
	err := s.DB.QueryRow(ctx,
		`INSERT INTO clusters(org_id, name, notes) VALUES($1,$2,$3)
		 RETURNING id, org_id, name, notes, created_at`,
		orgID, name, notes,
	).Scan(&c.ID, &c.OrgID, &c.Name, &c.Notes, &c.CreatedAt)
	return c, err
}

func (s *Store) ListClusters(ctx context.Context, orgID string) ([]Cluster, error) {
	rows, err := s.DB.Query(ctx,
		`SELECT id, org_id, name, notes, created_at FROM clusters WHERE org_id=$1 ORDER BY created_at DESC`,
		orgID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Cluster
	for rows.Next() {
		var c Cluster
		if err := rows.Scan(&c.ID, &c.OrgID, &c.Name, &c.Notes, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) CreateScan(ctx context.Context, orgID, clusterID, source string) (Scan, error) {
	var sc Scan
	err := s.DB.QueryRow(ctx,
		`INSERT INTO scans(org_id, cluster_id, source) VALUES($1,$2,$3)
		 RETURNING id, org_id, cluster_id, created_at, source`,
		orgID, clusterID, source,
	).Scan(&sc.ID, &sc.OrgID, &sc.ClusterID, &sc.CreatedAt, &sc.Source)
	return sc, err
}

func (s *Store) UpsertScanResult(ctx context.Context, scanID string, summary any, full any) error {
	sumB, _ := json.Marshal(summary)
	fullB, _ := json.Marshal(full)

	_, err := s.DB.Exec(ctx,
		`INSERT INTO scan_results(scan_id, summary, full_report)
		 VALUES($1,$2,$3)
		 ON CONFLICT (scan_id) DO UPDATE SET summary=EXCLUDED.summary, full_report=EXCLUDED.full_report`,
		scanID, sumB, fullB,
	)
	return err
}

func (s *Store) ListScans(ctx context.Context, orgID, clusterID string) ([]Scan, error) {
	rows, err := s.DB.Query(ctx,
		`SELECT id, org_id, cluster_id, created_at, source
		 FROM scans
		 WHERE org_id=$1 AND cluster_id=$2
		 ORDER BY created_at DESC
		 LIMIT 50`,
		orgID, clusterID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Scan
	for rows.Next() {
		var sc Scan
		if err := rows.Scan(&sc.ID, &sc.OrgID, &sc.ClusterID, &sc.CreatedAt, &sc.Source); err != nil {
			return nil, err
		}
		out = append(out, sc)
	}
	return out, rows.Err()
}

func (s *Store) GetScanReport(ctx context.Context, scanID string) (map[string]any, map[string]any, error) {
	var sumB, fullB []byte
	err := s.DB.QueryRow(ctx,
		`SELECT summary, full_report FROM scan_results WHERE scan_id=$1`,
		scanID,
	).Scan(&sumB, &fullB)
	if err != nil {
		return nil, nil, err
	}

	var sum map[string]any
	var full map[string]any
	_ = json.Unmarshal(sumB, &sum)
	_ = json.Unmarshal(fullB, &full)
	return sum, full, nil
}

func IsNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
func (s *Store) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.DB.Query(ctx,
		`SELECT id, email, password_hash, is_admin, created_at
		 FROM users
		 ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.IsAdmin, &u.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *Store) SetUserAdmin(ctx context.Context, userID string, admin bool) error {
	_, err := s.DB.Exec(ctx,
		`UPDATE users SET is_admin=$1 WHERE id=$2`,
		admin, userID,
	)
	return err
}
func (s *Store) ListAllUsers(ctx context.Context) ([]User, error) {
	rows, err := s.DB.Query(ctx,
		`SELECT id, email, is_admin, created_at
		 FROM users
		 ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.IsAdmin, &u.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *Store) ListAllOrgs(ctx context.Context) ([]Org, error) {
	rows, err := s.DB.Query(ctx,
		`SELECT id, name FROM orgs ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Org
	for rows.Next() {
		var o Org
		if err := rows.Scan(&o.ID, &o.Name); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}
func (s *Store) ToggleAdmin(ctx context.Context, userID string) error {
	_, err := s.DB.Exec(ctx,
		`UPDATE users
		 SET is_admin = NOT is_admin
		 WHERE id = $1`,
		userID,
	)
	return err
}
func (s *Store) AddAdminAuditLog(
	ctx context.Context,
	actorUserID string,
	targetUserID string,
	action string,
) error {
	_, err := s.DB.Exec(ctx,
		`INSERT INTO admin_audit_log(actor_user_id, target_user_id, action)
		 VALUES ($1,$2,$3)`,
		actorUserID, targetUserID, action,
	)
	return err
}

type AdminAuditEvent struct {
	ID          string
	AdminUserID string
	Action      string
	TargetType  string
	TargetID    string
	Meta        map[string]any
	CreatedAt   time.Time
}

func (s *Store) InsertAdminAudit(
	ctx context.Context,
	adminUserID string,
	action string,
	targetType string,
	targetID string,
	meta map[string]any,
) error {
	metaB, _ := json.Marshal(meta)

	_, err := s.DB.Exec(ctx,
		`INSERT INTO admin_audit_log
		 (admin_user_id, action, target_type, target_id, meta)
		 VALUES ($1,$2,$3,$4,$5)`,
		adminUserID,
		action,
		targetType,
		targetID,
		metaB,
	)
	return err
}

func (s *Store) ListAdminAudit(ctx context.Context, limit int) ([]AdminAuditEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	rows, err := s.DB.Query(ctx,
		`SELECT id, admin_user_id, action, target_type, target_id, meta, created_at
		 FROM admin_audit_log
		 ORDER BY created_at DESC
		 LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AdminAuditEvent
	for rows.Next() {
		var e AdminAuditEvent
		var metaB []byte
		if err := rows.Scan(
			&e.ID,
			&e.AdminUserID,
			&e.Action,
			&e.TargetType,
			&e.TargetID,
			&metaB,
			&e.CreatedAt,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(metaB, &e.Meta)
		out = append(out, e)
	}
	return out, rows.Err()
}
