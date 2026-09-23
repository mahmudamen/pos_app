package community

import (
	"context"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

// loadProfile reads one staff row (users LEFT JOIN user_profiles) for userID.
// Returns pgx.ErrNoRows when the user does not exist.
func loadProfile(ctx context.Context, tx pgx.Tx, userID, tenantID string) (StaffProfile, error) {
	var item StaffProfile
	err := tx.QueryRow(ctx, `
		SELECT u.id::text, u.display_name, u.email, COALESCE(p.headline, ''),
			COALESCE(p.bio, ''), COALESCE(p.avatar_url, ''), COALESCE(p.location, ''),
			COALESCE(p.years_experience, 0), COALESCE(p.is_chief, FALSE),
			COALESCE(p.skills, '{}'), COALESCE(p.resume, '{}'::jsonb), (p.user_id IS NOT NULL)
		FROM users u LEFT JOIN user_profiles p ON p.user_id = u.id
		WHERE u.id = $1 AND u.tenant_id = $2`, userID, tenantID).Scan(
		&item.UserID, &item.DisplayName, &item.Email, &item.Headline,
		&item.Bio, &item.AvatarURL, &item.Location, &item.YearsExperience, &item.IsChief,
		&item.Skills, &item.Resume, &item.HasProfile)
	return item, err
}

// loadMemberships returns the caller's active company memberships.
func loadMemberships(ctx context.Context, tx pgx.Tx, userID string) ([]CompanyMember, error) {
	rows, err := tx.Query(ctx, `
		SELECT m.id::text, m.company_id::text, m.user_id::text, u.display_name, m.role, m.title, m.created_at::text
		FROM company_members m
		JOIN users u ON u.id = m.user_id
		WHERE m.user_id = $1 AND m.is_active = TRUE
		ORDER BY m.created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]CompanyMember, 0)
	for rows.Next() {
		var item CompanyMember
		if err := rows.Scan(&item.ID, &item.CompanyID, &item.UserID, &item.Name, &item.Role, &item.Title, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// loadCompanyMembers returns the active members of one company.
func loadCompanyMembers(ctx context.Context, tx pgx.Tx, companyID string) ([]CompanyMember, error) {
	rows, err := tx.Query(ctx, `
		SELECT m.id::text, m.company_id::text, m.user_id::text, u.display_name, m.role, m.title, m.created_at::text
		FROM company_members m
		JOIN users u ON u.id = m.user_id
		WHERE m.company_id = $1 AND m.is_active = TRUE
		ORDER BY m.created_at`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]CompanyMember, 0)
	for rows.Next() {
		var item CompanyMember
		if err := rows.Scan(&item.ID, &item.CompanyID, &item.UserID, &item.Name, &item.Role, &item.Title, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// normalizeSkills trims, lowercases, dedupes and sorts skill labels.
func normalizeSkills(skills []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(skills))
	for _, s := range skills {
		v := strings.ToLower(strings.TrimSpace(s))
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}
