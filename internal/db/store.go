// Package db provides typed query helpers for the Svenskt Vin database.
package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// GetVineyard retrieves a vineyard by ID.
func (s *Store) GetVineyard(ctx context.Context, id int64) (*Vineyard, error) {
	var v Vineyard
	err := s.Pool.QueryRow(ctx, `
		SELECT id, name, county, municipality, lat, lon,
			   established_year, total_area_ha, organic, biodynamic,
			   legal_id, legal_id_type, legal_name, deleted_at
		FROM vineyards
		WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(
		&v.ID, &v.Name, &v.County, &v.Municipality,
		&v.Lat, &v.Lon,
		&v.EstablishedYear, &v.TotalAreaHA,
		&v.Organic, &v.Biodynamic,
		&v.LegalID, &v.LegalIDType, &v.LegalName,
		&v.DeletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get vineyard: %w", err)
	}
	return &v, nil
}

// ListVineyards retrieves all vineyards for a user.
func (s *Store) ListVineyards(ctx context.Context, userID int64) ([]Vineyard, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT v.id, v.name, v.county, v.municipality,
			   v.organic, v.biodynamic, v.total_area_ha,
			   v.established_year, v.deleted_at
		FROM vineyards v
		JOIN vineyard_members vm ON vm.vineyard_id = v.id
		WHERE vm.user_id = $1 AND v.deleted_at IS NULL
		ORDER BY v.name
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list vineyards: %w", err)
	}
	defer rows.Close()

	var vineyards []Vineyard
	for rows.Next() {
		var v Vineyard
		if err := rows.Scan(
			&v.ID, &v.Name, &v.County, &v.Municipality,
			&v.Organic, &v.Biodynamic, &v.TotalAreaHA,
			&v.EstablishedYear, &v.DeletedAt,
		); err != nil {
			continue
		}
		vineyards = append(vineyards, v)
	}
	return vineyards, nil
}

// CreateVineyard creates a new vineyard and auto-assigns the creator as owner.
func (s *Store) CreateVineyard(ctx context.Context, name, county, municipality string,
	lat, lon float64, establishedYear *int, totalAreaHA *float64,
	organic, biodynamic bool, legalID, legalIDType, legalName *string,
	ownerID int64) (int64, error) {

	var vineyardID int64
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO vineyards (
			name, county, municipality, lat, lon,
			established_year, total_area_ha,
			organic, biodynamic,
			legal_id, legal_id_type, legal_name
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`, name, county, municipality, lat, lon,
		establishedYear, totalAreaHA,
		organic, biodynamic,
		legalID, legalIDType, legalName).Scan(&vineyardID)
	if err != nil {
		return 0, fmt.Errorf("create vineyard: %w", err)
	}

	_, err = s.Pool.Exec(ctx, `
		INSERT INTO vineyard_members (vineyard_id, user_id, role)
		VALUES ($1, $2, 'owner')
	`, vineyardID, ownerID)
	if err != nil {
		return 0, fmt.Errorf("create vineyard member: %w", err)
	}

	return vineyardID, nil
}

// GetVineyardRole retrieves a user's role in a vineyard.
func (s *Store) GetVineyardRole(ctx context.Context, vineyardID, userID int64) (string, error) {
	var role string
	err := s.Pool.QueryRow(ctx, `
		SELECT role FROM vineyard_members
		WHERE vineyard_id = $1 AND user_id = $2
	`, vineyardID, userID).Scan(&role)
	if err != nil {
		return "", fmt.Errorf("get vineyard role: %w", err)
	}
	return role, nil
}

// CreateBlock creates a new block in a vineyard.
func (s *Store) CreateBlock(ctx context.Context, vineyardID, varietyID int64,
	blockName string, areaHA float64,
	vineCount *int, plantingYear *int,
	trainingSystem, aspect *string,
	slopeDegrees *float64, elevationM *int) (int64, error) {

	var blockID int64
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO blocks (
			vineyard_id, variety_id, block_name, area_ha,
			vine_count, planting_year, training_system, aspect,
			slope_degrees, elevation_m
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`, vineyardID, varietyID, blockName, areaHA,
		vineCount, plantingYear, trainingSystem, aspect,
		slopeDegrees, elevationM).Scan(&blockID)
	if err != nil {
		return 0, fmt.Errorf("create block: %w", err)
	}
	return blockID, nil
}

// ListBlocks retrieves all blocks for a vineyard.
type BlockSummary struct {
	Block        Block
	VarietyName  string
	VarietyColor string
}

// ListBlocks retrieves all blocks for a vineyard.
func (s *Store) ListBlocks(ctx context.Context, vineyardID int64) ([]BlockSummary, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT b.id, b.block_name, b.area_ha, b.vine_count, b.planting_year,
		       b.training_system, b.aspect, b.slope_degrees, b.elevation_m,
		       b.is_active, v.name AS variety_name, v.color
		FROM blocks b
		JOIN varieties v ON v.id = b.variety_id
		WHERE b.vineyard_id = $1 AND b.deleted_at IS NULL
		ORDER BY b.block_name
	`, vineyardID)
	if err != nil {
		return nil, fmt.Errorf("list blocks: %w", err)
	}
	defer rows.Close()

	var blocks []BlockSummary
	for rows.Next() {
		var b BlockSummary
		if err := rows.Scan(
			&b.Block.ID, &b.Block.BlockName, &b.Block.AreaHA,
			&b.Block.VineCount, &b.Block.PlantingYear,
			&b.Block.TrainingSystem, &b.Block.Aspect,
			&b.Block.SlopeDegrees, &b.Block.ElevationM,
			&b.Block.IsActive,
			&b.VarietyName, &b.VarietyColor,
		); err != nil {
			continue
		}
		blocks = append(blocks, b)
	}
	return blocks, nil
}

// GetBlock retrieves a single block by ID.
func (s *Store) GetBlock(ctx context.Context, blockID, vineyardID int64) (*struct {
	Block        Block
	VarietyName  string
	VarietyColor string
}, error) {
	var result struct {
		Block        Block
		VarietyName  string
		VarietyColor string
	}
	err := s.Pool.QueryRow(ctx, `
		SELECT b.id, b.block_name, b.variety_id, v.name, v.color,
			   b.area_ha, b.vine_count, b.planting_year,
			   b.training_system, b.aspect, b.slope_degrees, b.elevation_m,
			   b.is_active
		FROM blocks b
		JOIN varieties v ON v.id = b.variety_id
		WHERE b.id = $1 AND b.vineyard_id = $2 AND b.deleted_at IS NULL
	`, blockID, vineyardID).Scan(
		&result.Block.ID, &result.Block.BlockName, &result.Block.VarietyID,
		&result.VarietyName, &result.VarietyColor,
		&result.Block.AreaHA, &result.Block.VineCount, &result.Block.PlantingYear,
		&result.Block.TrainingSystem, &result.Block.Aspect,
		&result.Block.SlopeDegrees, &result.Block.ElevationM,
		&result.Block.IsActive,
	)
	if err != nil {
		return nil, fmt.Errorf("get block: %w", err)
	}
	return &result, nil
}

// CreateHarvest creates a new harvest record.
func (s *Store) CreateHarvest(ctx context.Context, blockID int64,
	harvestYear int, harvestDate *string, yieldKG float64,
	brix, acidGL *float64, vineHealthRating *int, notes *string,
	stillWineL, sparklingL, juiceL, soldKG, discardedKG *float64) (int64, error) {

	var harvestID int64
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO harvest_records (
			block_id, harvest_year, harvest_date, yield_kg,
			brix, acid_g_l, vine_health_rating, notes,
			still_wine_l, sparkling_l, juice_l,
			sold_kg, discarded_kg
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id
	`, blockID, harvestYear, harvestDate, yieldKG,
		brix, acidGL, vineHealthRating, notes,
		stillWineL, sparklingL, juiceL,
		soldKG, discardedKG).Scan(&harvestID)
	if err != nil {
		return 0, fmt.Errorf("create harvest: %w", err)
	}
	return harvestID, nil
}

// ListVarieties retrieves all approved varieties.
func (s *Store) ListVarieties(ctx context.Context) ([]Variety, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, name, piwi, color, status
		FROM varieties
		WHERE status = 'approved'
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list varieties: %w", err)
	}
	defer rows.Close()

	var varieties []Variety
	for rows.Next() {
		var v Variety
		if err := rows.Scan(&v.ID, &v.Name, &v.Piwi, &v.Color, &v.Status); err != nil {
			continue
		}
		varieties = append(varieties, v)
	}
	return varieties, nil
}

// SearchVarieties searches for varieties by name similarity.
func (s *Store) SearchVarieties(ctx context.Context, query string) ([]struct {
	ID    int64
	Name  string
	Piwi  bool
	Color string
	Score float64
}, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, name, piwi, color,
			   round(similarity(name, $1)::numeric, 2) AS score
		FROM varieties
		WHERE similarity(name, $1) > 0.4
			AND status = 'approved'
		ORDER BY similarity(name, $1) DESC
		LIMIT 3
	`, query)
	if err != nil {
		return nil, fmt.Errorf("search varieties: %w", err)
	}
	defer rows.Close()

	var matches []struct {
		ID    int64
		Name  string
		Piwi  bool
		Color string
		Score float64
	}
	for rows.Next() {
		var m struct {
			ID    int64
			Name  string
			Piwi  bool
			Color string
			Score float64
		}
		if err := rows.Scan(&m.ID, &m.Name, &m.Piwi, &m.Color, &m.Score); err != nil {
			continue
		}
		matches = append(matches, m)
	}
	return matches, nil
}

// CreateUser creates or upserts a user by email.
func (s *Store) CreateUser(ctx context.Context, email string) (int64, error) {
	var userID int64
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO users (email, name)
		SELECT $1, split_part($1, '@', 1)
		WHERE NOT EXISTS (SELECT 1 FROM users WHERE email = $1)
		RETURNING id
	`, email).Scan(&userID)

	if err != nil && err != pgx.ErrNoRows {
		return 0, fmt.Errorf("create user: %w", err)
	}

	// Handle existing user
	if userID == 0 {
		err = s.Pool.QueryRow(ctx, `
			SELECT id FROM users WHERE email = $1
		`, email).Scan(&userID)
		if err != nil {
			return 0, fmt.Errorf("lookup user: %w", err)
		}
	}

	return userID, nil
}

// GetUserByEmail retrieves a user by email (includes password_hash for login).
func (s *Store) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := s.Pool.QueryRow(ctx, `
		SELECT id, email, password_hash, name, is_admin
		FROM users
		WHERE email = $1 AND active = true
	`, email).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.IsAdmin)
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return &u, nil
}

// CreateUserPasswordHash sets a user's password hash.
func (s *Store) CreateUserPasswordHash(ctx context.Context, userID int64, hash string) error {
	_, err := s.Pool.Exec(ctx, `
		UPDATE users SET password_hash = $1 WHERE id = $2
	`, hash, userID)
	if err != nil {
		return fmt.Errorf("create user password hash: %w", err)
	}
	return nil
}

// UpdateLastLogin updates a user's last login timestamp.
func (s *Store) UpdateLastLogin(ctx context.Context, userID int64) error {
	_, err := s.Pool.Exec(ctx, `
		UPDATE users SET last_login = now() WHERE id = $1
	`, userID)
	return err
}

// GetUserInfo retrieves user info for session context.
func (s *Store) GetUserInfo(ctx context.Context, userID int64) (*User, error) {
	var u User
	err := s.Pool.QueryRow(ctx, `
		SELECT id, email, name, is_admin
		FROM users
		WHERE id = $1 AND active = true
	`, userID).Scan(&u.ID, &u.Email, &u.Name, &u.IsAdmin)
	if err != nil {
		return nil, fmt.Errorf("get user info: %w", err)
	}
	return &u, nil
}

// GetPendingInvite retrieves a pending invite by token.
func (s *Store) GetPendingInvite(ctx context.Context, token string) (*PendingInvite, error) {
	var pi PendingInvite
	err := s.Pool.QueryRow(ctx, `
		SELECT id, email, vineyard_id, role, expires_at
		FROM pending_invites
		WHERE token = $1 AND used = false AND expires_at > now()
	`, token).Scan(&pi.ID, &pi.Email, &pi.VineyardID, &pi.Role, &pi.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("get pending invite: %w", err)
	}
	return &pi, nil
}

// UpdatePendingInviteUsed marks a pending invite as used.
func (s *Store) UpdatePendingInviteUsed(ctx context.Context, inviteID int64) error {
	_, err := s.Pool.Exec(ctx, `
		UPDATE pending_invites SET used = true WHERE id = $1
	`, inviteID)
	if err != nil {
		return fmt.Errorf("update pending invite used: %w", err)
	}
	return nil
}

// GetVineyardName retrieves a vineyard's name by ID.
func (s *Store) GetVineyardName(ctx context.Context, vineyardID int64) (string, error) {
	var name string
	err := s.Pool.QueryRow(ctx, `
		SELECT name FROM vineyards WHERE id = $1 AND deleted_at IS NULL
	`, vineyardID).Scan(&name)
	if err != nil {
		return "", fmt.Errorf("get vineyard name: %w", err)
	}
	return name, nil
}

// DeleteSessionsByUser invalidates all sessions for a user.
func (s *Store) DeleteSessionsByUser(ctx context.Context, userID int64) error {
	_, err := s.Pool.Exec(ctx, `
		DELETE FROM sessions WHERE user_id = $1
	`, userID)
	if err != nil {
		return fmt.Errorf("delete sessions: %w", err)
	}
	return nil
}

// UpsertUser creates or updates a user by email (for enumeration-safe forgot password).
func (s *Store) UpsertUser(ctx context.Context, email string) (int64, error) {
	var userID int64
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO users (email, name, active)
		VALUES ($1, split_part($1, '@', 1), true)
		ON CONFLICT (email) DO UPDATE SET active = true
		RETURNING id
	`, email).Scan(&userID)
	if err != nil {
		return 0, fmt.Errorf("upsert user: %w", err)
	}
	return userID, nil
}

// HarvestSummary represents a block's latest harvest record.
type HarvestSummary struct {
	HarvestYear int
	YieldKg     float64
}

// BlockWithHarvest represents a block with its latest harvest and variety info.
type BlockWithHarvest struct {
	Block           Block
	VarietyName     string
	VarietyColor    string
	VarietyStatus   string
	LatestHarvest   *HarvestSummary
	IsActive        bool
}

// ListBlocksWithHarvest retrieves all blocks for a vineyard with latest harvest info.
func (s *Store) ListBlocksWithHarvest(ctx context.Context, vineyardID int64) ([]BlockWithHarvest, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT b.id, b.block_name, b.area_ha, b.vine_count, b.planting_year,
		       b.training_system, b.aspect, b.slope_degrees, b.elevation_m,
		       b.is_active,
		       v.name AS variety_name, v.color, v.status,
		       hr.harvest_year, hr.yield_kg
		FROM blocks b
		JOIN varieties v ON v.id = b.variety_id
		LEFT JOIN LATERAL (
			SELECT harvest_year, yield_kg
			FROM harvest_records
			WHERE block_id = b.id AND deleted_at IS NULL
			ORDER BY harvest_year DESC
			LIMIT 1
		) hr ON true
		WHERE b.vineyard_id = $1 AND b.deleted_at IS NULL
		ORDER BY b.block_name
	`, vineyardID)
	if err != nil {
		return nil, fmt.Errorf("list blocks with harvest: %w", err)
	}
	defer rows.Close()

	var blocks []BlockWithHarvest
	for rows.Next() {
		var b BlockWithHarvest
		var harvestYear *int
		var yieldKg *float64
		if err := rows.Scan(
			&b.Block.ID, &b.Block.BlockName, &b.Block.AreaHA,
			&b.Block.VineCount, &b.Block.PlantingYear,
			&b.Block.TrainingSystem, &b.Block.Aspect,
			&b.Block.SlopeDegrees, &b.Block.ElevationM,
			&b.IsActive,
			&b.VarietyName, &b.VarietyColor, &b.VarietyStatus,
			&harvestYear, &yieldKg,
		); err != nil {
			continue
		}
		b.Block.IsActive = b.IsActive
		if harvestYear != nil && yieldKg != nil {
			b.LatestHarvest = &HarvestSummary{
				HarvestYear: *harvestYear,
				YieldKg:     *yieldKg,
			}
		}
		blocks = append(blocks, b)
	}
	return blocks, nil
}

// BenchmarkTeaser represents a benchmark teaser on the vineyard dashboard.
type BenchmarkTeaser struct {
	VarietyName   string
	UserYieldKgHa float64
	VineyardCount int
}

// GetBenchmarkTeaser retrieves benchmark data for user's most recent harvest in this county.
func (s *Store) GetBenchmarkTeaser(ctx context.Context, vineyardID, userID int64) (*BenchmarkTeaser, error) {
	var county string
	err := s.Pool.QueryRow(ctx, `SELECT county FROM vineyards WHERE id = $1`, vineyardID).Scan(&county)
	if err != nil {
		return nil, fmt.Errorf("get benchmark teaser county: %w", err)
	}

	var varietyName string
	var userYieldKgHa float64
	err = s.Pool.QueryRow(ctx, `
		SELECT v.name, ROUND(hr.yield_kg / b.area_ha, 2)
		FROM harvest_records hr
		JOIN blocks b ON hr.block_id = b.id
		JOIN varieties v ON b.variety_id = v.id
		WHERE b.vineyard_id = $1 AND hr.deleted_at IS NULL
		ORDER BY hr.harvest_year DESC
		LIMIT 1
	`, vineyardID).Scan(&varietyName, &userYieldKgHa)
	if err != nil {
		return nil, nil
	}

	var vineyardCount int
	err = s.Pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT b.vineyard_id)
		FROM blocks b
		JOIN varieties v ON b.variety_id = v.id
		JOIN harvest_records hr ON hr.block_id = b.id
		JOIN vineyards vy ON vy.id = b.vineyard_id
		WHERE vy.county = $1 AND v.name = $2 AND hr.deleted_at IS NULL
		  AND b.vineyard_id != $3
	`, county, varietyName, vineyardID).Scan(&vineyardCount)
	if err != nil {
		vineyardCount = 0
	}

	return &BenchmarkTeaser{
		VarietyName:   varietyName,
		UserYieldKgHa: userYieldKgHa,
		VineyardCount: vineyardCount,
	}, nil
}

// UserYield represents user harvest data aggregated by variety + year.
type UserYield struct {
	VarietyName string
	Year        int
	YieldKgHa   float64
	TotalYield  float64
	AreaHA      float64
}

// RegionalBenchmark represents county-level benchmark data.
type RegionalBenchmark struct {
	VarietyName   string
	Year          int
	AvgYieldKgHa  float64
	MinYieldKgHa  float64
	MaxYieldKgHa  float64
	VineyardCount int
}

// BenchmarkResult holds all data for the benchmark page.
type BenchmarkResult struct {
	UserYields    []UserYield
	RegionalBench []RegionalBenchmark
	Timeline      []struct {
		Year    int
		YieldKg float64
		Variety string
	}
}

// GetBenchmarkData retrieves all data needed for the benchmark page.
func (s *Store) GetBenchmarkData(ctx context.Context, vineyardID int64) BenchmarkResult {
	var result BenchmarkResult

	// User yields by variety + year
	rows, err := s.Pool.Query(ctx, `
		SELECT v.name, hr.harvest_year, ROUND(hr.yield_kg / b.area_ha, 2),
		       hr.yield_kg, b.area_ha
		FROM harvest_records hr
		JOIN blocks b ON hr.block_id = b.id
		JOIN varieties v ON b.variety_id = v.id
		WHERE b.vineyard_id = $1 AND hr.deleted_at IS NULL
		ORDER BY hr.harvest_year DESC, v.name
	`, vineyardID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var uy UserYield
			if err := rows.Scan(&uy.VarietyName, &uy.Year, &uy.YieldKgHa, &uy.TotalYield, &uy.AreaHA); err != nil {
				continue
			}
			result.UserYields = append(result.UserYields, uy)
		}
	}

	// Regional benchmarks (min 3 vineyards)
	rows, err = s.Pool.Query(ctx, `
		SELECT v.name, hr.harvest_year,
		       ROUND(AVG(hr.yield_kg / b.area_ha), 2),
		       ROUND(MIN(hr.yield_kg / b.area_ha), 2),
		       ROUND(MAX(hr.yield_kg / b.area_ha), 2),
		       COUNT(DISTINCT b.vineyard_id)
		FROM harvest_records hr
		JOIN blocks b ON hr.block_id = b.id
		JOIN varieties v ON b.variety_id = v.id
		JOIN vineyards vy ON vy.id = b.vineyard_id
		WHERE vy.county = (SELECT county FROM vineyards WHERE id = $1) AND hr.deleted_at IS NULL
		GROUP BY v.name, hr.harvest_year
		HAVING COUNT(DISTINCT b.vineyard_id) >= 3
		ORDER BY hr.harvest_year DESC, v.name
	`, vineyardID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var rb RegionalBenchmark
			if err := rows.Scan(&rb.VarietyName, &rb.Year, &rb.AvgYieldKgHa, &rb.MinYieldKgHa, &rb.MaxYieldKgHa, &rb.VineyardCount); err != nil {
				continue
			}
			result.RegionalBench = append(result.RegionalBench, rb)
		}
	}

	// Timeline
	rows, err = s.Pool.Query(ctx, `
		SELECT hr.harvest_year, hr.yield_kg, v.name
		FROM harvest_records hr
		JOIN blocks b ON hr.block_id = b.id
		JOIN varieties v ON b.variety_id = v.id
		WHERE b.vineyard_id = $1 AND hr.deleted_at IS NULL
		ORDER BY hr.harvest_year DESC
	`, vineyardID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var t struct {
				Year    int
				YieldKg float64
				Variety string
			}
			if err := rows.Scan(&t.Year, &t.YieldKg, &t.Variety); err != nil {
				continue
			}
			result.Timeline = append(result.Timeline, t)
		}
	}

	return result
}
