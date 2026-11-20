package database

import (
	"errors"
	"strings"
	"testing"

	"app/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func testConfig() *config.Config {
	return &config.Config{AdminUsername: "admin", AdminPassword: "admin", DBHost: "127.0.0.1", DBPort: "1",
		DBUsername: "x", DBPassword: "x", DBDatabase: "x", DBSSLMode: "disable"}
}

func closeDB(db *gorm.DB) {
	sqlDB, _ := db.DB()
	_ = sqlDB.Close()
}

func TestOpenSQLiteAndMigrate(t *testing.T) {
	db, err := OpenSQLite()
	require.NoError(t, err)
	require.NoError(t, Migrate(db))
	assert.True(t, db.Migrator().HasTable(&User{}))
	assert.True(t, db.Migrator().HasTable(&Post{}))
}

func TestMigrateError(t *testing.T) {
	db, err := OpenSQLite()
	require.NoError(t, err)
	closeDB(db)
	assert.Error(t, Migrate(db))
}

func TestConnectUsesOpener(t *testing.T) {
	orig := openPostgres
	defer func() { openPostgres = orig }()

	var gotDSN string
	openPostgres = func(dsn string) (*gorm.DB, error) {
		gotDSN = dsn
		return OpenSQLite()
	}
	db, err := Connect(testConfig())
	require.NoError(t, err)
	assert.NotNil(t, db)
	assert.True(t, strings.Contains(gotDSN, "dbname=x"))

	openPostgres = func(dsn string) (*gorm.DB, error) { return nil, errors.New("fail") }
	_, err = Connect(testConfig())
	assert.Error(t, err)
}

func TestDefaultOpenPostgresFails(t *testing.T) {
	_, err := openPostgres("host=127.0.0.1 port=1 user=x password=x dbname=x sslmode=disable connect_timeout=1")
	assert.Error(t, err)
}

func TestSeed(t *testing.T) {
	db, err := OpenSQLite()
	require.NoError(t, err)
	require.NoError(t, Migrate(db))

	require.NoError(t, Seed(db, testConfig()))
	require.NoError(t, Seed(db, testConfig())) // idempotent

	var users []User
	require.NoError(t, db.Order("id").Find(&users).Error)
	require.Len(t, users, 2)
	assert.Equal(t, "admin", users[0].Role)
	assert.Equal(t, "member", users[1].Username)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(users[0].Password), []byte("admin")))
}

func TestSeedErrors(t *testing.T) {
	db, err := OpenSQLite()
	require.NoError(t, err)
	require.NoError(t, Migrate(db))

	cfg := testConfig()
	cfg.AdminPassword = strings.Repeat("x", 100) // bcrypt menolak password > 72 byte
	assert.Error(t, Seed(db, cfg))

	closeDB(db)
	assert.Error(t, Seed(db, testConfig()))
}
