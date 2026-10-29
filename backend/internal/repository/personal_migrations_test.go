package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/privacy/testutil"
	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

func TestPersonalPrivacyPreservesImageGenerationChoices(t *testing.T) {
	testutil.Run(t, func(t *testing.T) {
		const name = "246_enable_gemini_image_generation_groups.sql"
		content, err := migrations.FS.ReadFile(name)
		require.NoError(t, err)
		sum := sha256.Sum256([]byte(strings.TrimSpace(string(content))))
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()
		prepareMigrationsBootstrapExpectations(mock)
		mock.ExpectQuery("SELECT checksum FROM schema_migrations WHERE filename = \\$1").
			WithArgs(name).WillReturnError(sql.ErrNoRows)
		mock.ExpectBegin()
		mock.ExpectExec("^SELECT 1$").WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec("INSERT INTO schema_migrations").
			WithArgs(name, hex.EncodeToString(sum[:])).WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()
		mock.ExpectExec("SELECT pg_advisory_unlock").WithArgs(migrationsAdvisoryLockID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		require.NoError(t, applyMigrationsFS(context.Background(), db, fstest.MapFS{
			name: &fstest.MapFile{Data: content},
		}))
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
