package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestModelContextLengthPersistsAndCanBeCleared(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Model{}))

	previousDB := DB
	DB = db
	t.Cleanup(func() { DB = previousDB })

	entry := &Model{
		ModelName:     "context-test-model",
		Status:        1,
		SyncOfficial:  1,
		ContextLength: 128000,
	}
	require.NoError(t, entry.Insert())

	var stored Model
	require.NoError(t, db.First(&stored, entry.Id).Error)
	assert.Equal(t, 128000, stored.ContextLength)

	entry.ContextLength = 0
	require.NoError(t, entry.Update())
	require.NoError(t, db.First(&stored, entry.Id).Error)
	assert.Equal(t, 0, stored.ContextLength)
}
